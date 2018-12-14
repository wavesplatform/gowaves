package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"strconv"
	"strings"
)

const (
	headerSize     = 2
	bodySize       = 20
	checksumSize   = 4
	AddressSize    = headerSize + bodySize + checksumSize
	aliasFixedSize = 4

	addressVersion byte = 0x01

	aliasVersion   byte = 0x02
	aliasMinLength      = 4
	aliasMaxLength      = 30
	aliasAlphabet       = "-.0123456789@_abcdefghijklmnopqrstuvwxyz"
	aliasPrefix         = "alias"

	MainNetScheme byte = 'W'
	TestNetScheme byte = 'T'
	DevNetScheme  byte = 'D'
)

type Address [AddressSize]byte

func (a *Address) String() string {
	return base58.Encode(a[:])
}

func (a Address) MarshalJSON() ([]byte, error) {
	return B58Bytes(a[:]).MarshalJSON()
}

func (a *Address) UnmarshalJSON(value []byte) error {
	var b B58Bytes
	err := b.UnmarshalJSON(value)
	if err != nil {
		return err
	}
	if l := len(b); l != AddressSize {
		return fmt.Errorf("incorrect Address size %d, expected %d", l, AddressSize)
	}
	copy(a[:], b)
	return nil
}

func NewAddressFromPublicKey(scheme byte, publicKey crypto.PublicKey) (Address, error) {
	var a Address
	a[0] = addressVersion
	a[1] = scheme
	h, err := crypto.SecureHash(publicKey[:])
	if err != nil {
		return a, errors.Wrap(err, "failed to produce Digest from PublicKey")
	}
	copy(a[headerSize:], h[:bodySize])
	cs, err := addressChecksum(a[:headerSize+bodySize])
	if err != nil {
		return a, errors.Wrap(err, "failed to calculate Address checksum")
	}
	copy(a[headerSize+bodySize:], cs)
	return a, nil
}

func NewAddressFromString(s string) (Address, error) {
	var a Address
	b, err := base58.Decode(s)
	if err != nil {
		return a, errors.Wrap(err, "invalid Base58 string")
	}
	a, err = NewAddressFromBytes(b)
	if err != nil {
		return a, fmt.Errorf("failed to create an Address from Base58 string: %s", err.Error())
	}
	return a, nil
}

func NewAddressFromBytes(b []byte) (Address, error) {
	var a Address
	if l := len(b); l < AddressSize {
		return a, fmt.Errorf("insufficient array length %d, expected atleast %d", l, AddressSize)
	}
	copy(a[:], b[:AddressSize])
	if ok, err := a.Validate(); !ok {
		return a, fmt.Errorf("invalid address: %s", err.Error())
	}
	return a, nil
}

func (a *Address) Validate() (bool, error) {
	if a[0] != addressVersion {
		return false, fmt.Errorf("unsupported address version")
	}
	hb := a[:headerSize+bodySize]
	ec, err := addressChecksum(hb)
	if err != nil {
		return false, errors.Wrap(err, "failed to calculate Address checksum")
	}
	ac := a[headerSize+bodySize:]
	if !bytes.Equal(ec, ac) {
		return false, fmt.Errorf("invalid Address checksum")
	}
	return true, nil
}

func addressChecksum(b []byte) ([]byte, error) {
	h, err := crypto.SecureHash(b)
	if err != nil {
		return nil, err
	}
	c := make([]byte, checksumSize)
	copy(c, h[:checksumSize])
	return c, nil
}

type Alias struct {
	Version byte
	Scheme  byte
	Alias   string
}

func (a *Alias) String() string {
	sb := new(strings.Builder)
	sb.WriteString(aliasPrefix)
	sb.WriteRune(':')
	sb.WriteByte(a.Scheme)
	sb.WriteRune(':')
	sb.WriteString(a.Alias)
	return sb.String()
}

func (a Alias) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(a.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (a *Alias) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Alias from JSON")
	}
	t, err := NewAliasFromString(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Alias from JSON")
	}
	*a = *t
	return nil
}

func (a *Alias) MarshalBinary() ([]byte, error) {
	al := len(a.Alias)
	buf := make([]byte, aliasFixedSize+al)
	buf[0] = a.Version
	buf[1] = a.Scheme
	PutStringWithUInt16Len(buf[2:], a.Alias)
	return buf, nil
}

func (a *Alias) UnmarshalBinary(data []byte) error {
	dl := len(data)
	if dl < aliasFixedSize+aliasMinLength {
		return errors.Errorf("incorrect alias length %d, should be at least %d bytes", dl, aliasFixedSize+aliasMinLength)
	}
	if data[0] != aliasVersion {
		return errors.Errorf("unsupported alias version %d, expected %d", data[0], aliasVersion)
	}
	a.Version = data[0]
	a.Scheme = data[1]
	al := int(binary.BigEndian.Uint16(data[2:4]))
	data = data[4:]
	if al > aliasMaxLength {
		return errors.Errorf("alias too long, received length %d is bigger then maximum allowed %d", al, aliasMaxLength)
	}
	if l := len(data); l < al {
		return errors.Errorf("incorrect alias length: encoded length %d, actual %d", al, l)
	}
	s := string(data[:al])
	if !correctAlphabet(&s) {
		return errors.Errorf("unsupported symbols in alias '%s', supported symbols '%s", a.Alias, aliasAlphabet)
	}
	a.Alias = s
	return nil
}

func NewAlias(scheme byte, alias string) (*Alias, error) {
	if len(alias) < aliasMinLength || len(alias) > aliasMaxLength {
		return nil, errors.Errorf("alias length should be between %d and %d", aliasMinLength, aliasMaxLength)
	}
	if !correctAlphabet(&alias) {
		return nil, errors.Errorf("alias should contain only following characters: %s", aliasAlphabet)
	}
	return &Alias{aliasVersion, scheme, alias}, nil
}

func NewAliasFromString(s string) (*Alias, error) {
	ps := strings.Split(s, ":")
	if len(ps) != 3 {
		return nil, errors.Errorf("incorrect alias string representation '%s'", s)
	}
	if ps[0] != aliasPrefix {
		return nil, errors.Errorf("alias should start with prefix '%s'", aliasPrefix)
	}
	scheme := ps[1]
	if len(scheme) != 1 {
		return nil, errors.Errorf("incorrect alias chainID '%s'", scheme)
	}
	return NewAlias(scheme[0], ps[2])
}

func NewAliasFromBytes(b []byte) (*Alias, error) {
	var a Alias
	err := a.UnmarshalBinary(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new Alias from bytes")
	}
	return &a, nil
}

func correctAlphabet(s *string) bool {
	for _, c := range *s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && c != '_' && c != '@' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

type Recipient struct {
	Address *Address
	Alias   *Alias
	len     int
}

func NewRecipientFromAddress(a Address) Recipient {
	return Recipient{Address: &a, len: AddressSize}
}

func NewRecipientFromAlias(a Alias) Recipient {
	return Recipient{Alias: &a, len: aliasFixedSize + len(a.Alias)}
}

func (r Recipient) MarshalJSON() ([]byte, error) {
	if r.Alias != nil {
		return r.Alias.MarshalJSON()
	}
	return r.Address.MarshalJSON()
}

func (r *Recipient) UnmarshalJSON(value []byte) error {
	s := string(value)
	if strings.Index(s, aliasPrefix) != -1 {
		var a Alias
		err := a.UnmarshalJSON(value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from JSON")
		}
		r.Alias = &a
		r.len = aliasFixedSize + len(a.Alias)
		return nil
	}
	var a Address
	err := a.UnmarshalJSON(value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Recipient from JSON")
	}
	r.Address = &a
	r.len = AddressSize
	return nil
}

func (r *Recipient) MarshalBinary() ([]byte, error) {
	if r.Alias != nil {
		return r.Alias.MarshalBinary()
	}
	return r.Address[:], nil
}

func (r *Recipient) UnmarshalBinary(data []byte) error {
	switch v := data[0]; v {
	case addressVersion:
		a, err := NewAddressFromBytes(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from bytes")
		}
		r.Address = &a
		r.len = AddressSize
		return nil
	case aliasVersion:
		var a Alias
		err := a.UnmarshalBinary(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from bytes")
		}
		r.Alias = &a
		r.len = aliasFixedSize + len(a.Alias)
		return nil
	default:
		return errors.Errorf("unsupported Recipient version %d", v)
	}
}

func (r *Recipient) String() string {
	if r.Alias != nil {
		return r.Alias.String()
	}
	return r.Address.String()
}

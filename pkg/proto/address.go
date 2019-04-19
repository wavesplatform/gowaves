package proto

import (
	"bytes"
	"encoding/binary"
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

// Address is the transformed Public Key with additional bytes of the version, a blockchain scheme and a checksum.
type Address [AddressSize]byte

// String produces the BASE58 string representation of the Address.
func (a Address) String() string {
	return base58.Encode(a[:])
}

// MarshalJSON is the custom JSON marshal function for the Address.
func (a Address) MarshalJSON() ([]byte, error) {
	return B58Bytes(a[:]).MarshalJSON()
}

// UnmarshalJSON tries to unmarshal an Address from it's JSON representation.
// This method does not perform validation of the result address.
func (a *Address) UnmarshalJSON(value []byte) error {
	b := B58Bytes{}
	err := b.UnmarshalJSON(value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Address from JSON")
	}
	if l := len(b); l != AddressSize {
		return errors.Errorf("incorrect size of an Address %d, expected %d", l, AddressSize)
	}
	copy(a[:], b)
	return nil
}

// NewAddressFromPublicKey produces an Address from given scheme and Public Key bytes.
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

// NewAddressFromString creates an Address from its string representation. This function checks that the address is valid.
func NewAddressFromString(s string) (Address, error) {
	var a Address
	b, err := base58.Decode(s)
	if err != nil {
		return a, errors.Wrap(err, "invalid Base58 string")
	}
	a, err = NewAddressFromBytes(b)
	if err != nil {
		return a, errors.Wrap(err, "failed to create an Address from Base58 string")
	}
	return a, nil
}

// NewAddressFromBytes creates an Address from the slice of bytes and checks that the result address is valid address.
func NewAddressFromBytes(b []byte) (Address, error) {
	var a Address
	if l := len(b); l < AddressSize {
		return a, errors.Errorf("insufficient array length %d, expected at least %d", l, AddressSize)
	}
	copy(a[:], b[:AddressSize])
	if ok, err := a.Valid(); !ok {
		return a, errors.Wrap(err, "invalid address")
	}
	return a, nil
}

// Valid checks that version and checksum of the Address are correct.
func (a *Address) Valid() (bool, error) {
	if a[0] != addressVersion {
		return false, errors.Errorf("unsupported address version %d", a[0])
	}
	hb := a[:headerSize+bodySize]
	ec, err := addressChecksum(hb)
	if err != nil {
		return false, errors.Wrap(err, "failed to calculate Address checksum")
	}
	ac := a[headerSize+bodySize:]
	if !bytes.Equal(ec, ac) {
		return false, errors.New("invalid Address checksum")
	}
	return true, nil
}

// Bytes converts the fixed-length byte array of the Address to a slice of bytes.
func (a Address) Bytes() []byte {
	return a[:]
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

// Alias represents the nickname tha could be attached to the Address.
type Alias struct {
	Version byte
	Scheme  byte
	Alias   string
}

// NewAliasFromString creates an Alias from its string representation. Function does not check that the result is a valid Alias.
// String representation of an Alias should have a following format: "alias:<scheme>:<alias>". Scheme should be represented with a one-byte ASCII symbol.
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
	a := Alias{Version: aliasVersion, Scheme: scheme[0], Alias: ps[2]}
	return &a, nil
}

// NewAliasFromBytes unmarshal an Alias from bytes and checks that it's valid.
func NewAliasFromBytes(b []byte) (*Alias, error) {
	var a Alias
	err := a.UnmarshalBinary(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new Alias from bytes")
	}
	return &a, nil
}

// String converts the Alias to its 3-part string representation.
func (a Alias) String() string {
	sb := new(strings.Builder)
	sb.WriteString(aliasPrefix)
	sb.WriteRune(':')
	sb.WriteByte(a.Scheme)
	sb.WriteRune(':')
	sb.WriteString(a.Alias)
	return sb.String()
}

// MarshalJSON is a custom JSON marshalling function.
func (a Alias) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(a.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads an Alias from JSON.
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

// MarshalBinary converts the Alias to the slice of bytes. Just calls Bytes().
func (a *Alias) MarshalBinary() ([]byte, error) {
	return a.Bytes(), nil
}

// Bytes converts the Alias to the slice of bytes.
func (a *Alias) Bytes() []byte {
	al := len(a.Alias)
	buf := make([]byte, aliasFixedSize+al)
	buf[0] = a.Version
	buf[1] = a.Scheme
	PutStringWithUInt16Len(buf[2:], a.Alias)
	return buf
}

// Reads an Alias from its bytes representation. This function does not validate the result.
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
	if l := len(data); l < al {
		return errors.Errorf("incorrect alias length: encoded length %d, actual %d", al, l)
	}
	a.Alias = string(data[:al])
	return nil
}

func NewAlias(scheme byte, alias string) *Alias {
	return &Alias{aliasVersion, scheme, alias}
}

// Valid validates the Alias checking it length, version and symbols.
func (a Alias) Valid() (bool, error) {
	if v := a.Version; v != aliasVersion {
		return false, errors.Errorf("%d is incorrect alias version, expected %d", v, aliasVersion)
	}
	if l := len(a.Alias); l < aliasMinLength || l > aliasMaxLength {
		return false, errors.Errorf("alias length should be between %d and %d", aliasMinLength, aliasMaxLength)
	}
	if !correctAlphabet(a.Alias) {
		return false, errors.Errorf("alias should contain only following characters: %s", aliasAlphabet)
	}
	return true, nil
}

func correctAlphabet(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && c != '_' && c != '@' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

// Recipient could be an Alias or an Address.
type Recipient struct {
	Address *Address
	Alias   *Alias
	len     int
}

// NewRecipientFromAddress creates the Recipient from given address.
func NewRecipientFromAddress(a Address) Recipient {
	return Recipient{Address: &a, len: AddressSize}
}

// NewRecipientFromAlias creates a Recipient with the given Alias inside.
func NewRecipientFromAlias(a Alias) Recipient {
	return Recipient{Alias: &a, len: aliasFixedSize + len(a.Alias)}
}

// Valid checks that either an Address or an Alias is set then checks the validity of the set field.
func (r Recipient) Valid() (bool, error) {
	switch {
	case r.Address != nil:
		return r.Address.Valid()
	case r.Alias != nil:
		return r.Alias.Valid()
	default:
		return false, errors.New("empty recipient")
	}
}

// MarshalJSON converts the Recipient to its JSON representation.
func (r Recipient) MarshalJSON() ([]byte, error) {
	if r.Alias != nil {
		return r.Alias.MarshalJSON()
	}
	return r.Address.MarshalJSON()
}

// UnmarshalJSON reads the Recipient from its JSON representation.
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

// MarshalBinary makes bytes of the Recipient.
func (r *Recipient) MarshalBinary() ([]byte, error) {
	if r.Alias != nil {
		return r.Alias.MarshalBinary()
	}
	return r.Address[:], nil
}

// UnmarshalBinary reads the Recipient from bytes. Validates the result.
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

// String gives the string representation of the Recipient.
func (r *Recipient) String() string {
	if r.Alias != nil {
		return r.Alias.String()
	}
	return r.Address.String()
}

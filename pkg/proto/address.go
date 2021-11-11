package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

const (
	AddressIDSize            = 20
	wavesAddressHeaderSize   = 2
	wavesAddressBodySize     = AddressIDSize
	wavesAddressChecksumSize = 4
	WavesAddressSize         = wavesAddressHeaderSize + wavesAddressBodySize + wavesAddressChecksumSize

	wavesAddressVersion byte = 0x01
	aliasVersion        byte = 0x02

	aliasFixedSize = 4
	AliasMinLength = 4
	AliasMaxLength = 30
	AliasAlphabet  = "-.0123456789@_abcdefghijklmnopqrstuvwxyz"
	AliasPrefix    = "alias"

	EthereumAddressSize = AddressIDSize

	MainNetScheme   byte = 'W'
	TestNetScheme   byte = 'T'
	StageNetScheme  byte = 'S'
	CustomNetScheme byte = 'E'
)

type AddressID [AddressIDSize]byte

func (a AddressID) Bytes() []byte {
	return a[:]
}

func (a AddressID) ToWavesAddress(scheme Scheme) WavesAddress {
	return mustRebuildAddress(scheme, a[:])
}

type Address interface {
	ID() AddressID
	Bytes() []byte
	String() string
}

// EthereumAddress is the first 20 bytes of Public Key's hash for the Waves address, or the 20 bytes of an Ethereum address.
type EthereumAddress [EthereumAddressSize]byte

// Bytes converts the fixed-length byte array of the EthereumAddress to a slice of bytes.
func (b EthereumAddress) Bytes() []byte {
	return b[:]
}

func (b EthereumAddress) ID() AddressID {
	var id AddressID
	copy(id[:], b[:])
	return id
}

func (b EthereumAddress) String() string {
	sb := new(strings.Builder)
	sb.WriteString("0x")
	sb.WriteString(hex.EncodeToString(b[:]))
	return sb.String()
}

// WavesAddress is the transformed Public Key with additional bytes of the version, a blockchain scheme and a checksum.
type WavesAddress [WavesAddressSize]byte

func (a WavesAddress) Body() []byte {
	return a[wavesAddressHeaderSize : wavesAddressHeaderSize+wavesAddressBodySize]
}

func (a WavesAddress) ID() AddressID {
	var id AddressID
	copy(id[:], a[wavesAddressHeaderSize:wavesAddressHeaderSize+wavesAddressBodySize])
	return id
}

// String produces the BASE58 string representation of the WavesAddress.
func (a WavesAddress) String() string {
	return base58.Encode(a[:])
}

// MarshalJSON is the custom JSON marshal function for the WavesAddress.
func (a WavesAddress) MarshalJSON() ([]byte, error) {
	return B58Bytes(a[:]).MarshalJSON()
}

// UnmarshalJSON tries to unmarshal an WavesAddress from it's JSON representation.
// This method does not perform validation of the result address.
func (a *WavesAddress) UnmarshalJSON(value []byte) error {
	b := B58Bytes{}
	err := b.UnmarshalJSON(value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal WavesAddress from JSON")
	}
	if l := len(b); l != WavesAddressSize {
		return errors.Errorf("incorrect size of an WavesAddress %d, expected %d", l, WavesAddressSize)
	}
	copy(a[:], b)
	return nil
}

// NewAddressFromPublicKey produces an WavesAddress from given scheme and Public Key bytes.
func NewAddressFromPublicKey(scheme byte, publicKey crypto.PublicKey) (WavesAddress, error) {
	var a WavesAddress
	a[0] = wavesAddressVersion
	a[1] = scheme
	h, err := crypto.SecureHash(publicKey[:])
	if err != nil {
		return a, errors.Wrap(err, "failed to produce Digest from PublicKey")
	}
	copy(a[wavesAddressHeaderSize:], h[:wavesAddressBodySize])
	cs, err := addressChecksum(a[:wavesAddressHeaderSize+wavesAddressBodySize])
	if err != nil {
		return a, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	copy(a[wavesAddressHeaderSize+wavesAddressBodySize:], cs)
	return a, nil
}

// NewAddressLikeFromAnyBytes produces an WavesAddress from given scheme and bytes.
func NewAddressLikeFromAnyBytes(scheme byte, b []byte) (WavesAddress, error) {
	var a WavesAddress
	a[0] = wavesAddressVersion
	a[1] = scheme
	h, err := crypto.SecureHash(b)
	if err != nil {
		return a, errors.Wrap(err, "failed to produce Digest from any bytes")
	}
	copy(a[wavesAddressHeaderSize:], h[:wavesAddressBodySize])
	cs, err := addressChecksum(a[:wavesAddressHeaderSize+wavesAddressBodySize])
	if err != nil {
		return a, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	copy(a[wavesAddressHeaderSize+wavesAddressBodySize:], cs)
	return a, nil
}

func MustAddressFromPublicKey(scheme byte, publicKey crypto.PublicKey) WavesAddress {
	rs, err := NewAddressFromPublicKey(scheme, publicKey)
	if err != nil {
		panic(err)
	}
	return rs
}

func RebuildAddress(scheme byte, body []byte) (WavesAddress, error) {
	if len(body) == 26 {
		return NewAddressFromBytes(body)
	}
	var a WavesAddress
	a[0] = wavesAddressVersion
	a[1] = scheme
	if l := len(body); l != wavesAddressBodySize {
		return WavesAddress{}, errors.Errorf("%d is unexpected address' body size", l)
	}
	copy(a[wavesAddressHeaderSize:], body[:wavesAddressBodySize])
	cs, err := addressChecksum(a[:wavesAddressHeaderSize+wavesAddressBodySize])
	if err != nil {
		return a, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	copy(a[wavesAddressHeaderSize+wavesAddressBodySize:], cs)
	return a, nil
}

func mustRebuildAddress(scheme byte, body []byte) WavesAddress {
	addr, err := RebuildAddress(scheme, body)
	if err != nil {
		panic(err.Error())
	}
	return addr
}

// NewAddressFromString creates an WavesAddress from its string representation. This function checks that the address is valid.
func NewAddressFromString(s string) (WavesAddress, error) {
	var a WavesAddress
	b, err := base58.Decode(s)
	if err != nil {
		return a, errors.Wrap(err, "invalid Base58 string")
	}
	a, err = NewAddressFromBytes(b)
	if err != nil {
		return a, errors.Wrap(err, "failed to create an WavesAddress from Base58 string")
	}
	return a, nil
}

func MustAddressFromString(s string) WavesAddress {
	addr, err := NewAddressFromString(s)
	if err != nil {
		panic(err)
	}
	return addr
}

// NewAddressFromBytes creates an WavesAddress from the slice of bytes and checks that the result address is valid address.
func NewAddressFromBytes(b []byte) (WavesAddress, error) {
	var a WavesAddress
	if l := len(b); l < WavesAddressSize {
		return a, errors.Errorf("insufficient array length %d, expected at least %d", l, WavesAddressSize)
	}
	copy(a[:], b[:WavesAddressSize])
	if ok, err := a.Valid(); !ok {
		return a, errors.Wrap(err, "invalid address")
	}
	return a, nil
}

// Valid checks that version and checksum of the WavesAddress are correct.
func (a *WavesAddress) Valid() (bool, error) {
	if a[0] != wavesAddressVersion {
		return false, errors.Errorf("unsupported address version %d", a[0])
	}
	hb := a[:wavesAddressHeaderSize+wavesAddressBodySize]
	ec, err := addressChecksum(hb)
	if err != nil {
		return false, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	ac := a[wavesAddressHeaderSize+wavesAddressBodySize:]
	if !bytes.Equal(ec, ac) {
		return false, errors.New("invalid WavesAddress checksum")
	}
	return true, nil
}

// Bytes converts the fixed-length byte array of the WavesAddress to a slice of bytes.
func (a WavesAddress) Bytes() []byte {
	return a[:]
}

func (a *WavesAddress) Eq(b WavesAddress) bool {
	return bytes.Equal(a.Bytes(), b.Bytes())
}

func addressChecksum(b []byte) ([]byte, error) {
	h, err := crypto.SecureHash(b)
	if err != nil {
		return nil, err
	}
	c := make([]byte, wavesAddressChecksumSize)
	copy(c, h[:wavesAddressChecksumSize])
	return c, nil
}

// Alias represents the nickname tha could be attached to the WavesAddress.
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
	if ps[0] != AliasPrefix {
		return nil, errors.Errorf("alias should start with prefix '%s'", AliasPrefix)
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

func (a Alias) BinarySize() int {
	return aliasFixedSize + len(a.Alias)
}

// String converts the Alias to its 3-part string representation.
func (a Alias) String() string {
	sb := new(strings.Builder)
	sb.WriteString(AliasPrefix)
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

func (a *Alias) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := a.Serialize(s)
	if err != nil {
		return 0, err
	}
	return s.N(), nil
}

func (a *Alias) Serialize(s *serializer.Serializer) error {
	err := s.Byte(a.Version)
	if err != nil {
		return err
	}
	err = s.Byte(a.Scheme)
	if err != nil {
		return err
	}
	err = s.StringWithUInt16Len(a.Alias)
	if err != nil {
		return err
	}
	return nil
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

// UnmarshalBinary reads an Alias from its bytes representation. This function does not validate the result.
func (a *Alias) UnmarshalBinary(data []byte) error {
	dl := len(data)
	if dl < aliasFixedSize+AliasMinLength {
		return errors.Errorf("incorrect alias length %d, should be at least %d bytes", dl, aliasFixedSize+AliasMinLength)
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
	if l := len(a.Alias); l < AliasMinLength || l > AliasMaxLength {
		return false, errs.NewTxValidationError(fmt.Sprintf("Alias '%s' length should be between %d and %d", a.Alias, AliasMinLength, AliasMaxLength))
	}
	if !correctAlphabet(a.Alias) {
		return false, errs.NewTxValidationError(fmt.Sprintf("Alias should contain only following characters: %s", AliasAlphabet))
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

// Recipient could be an Alias or an WavesAddress.
type Recipient struct {
	Address *WavesAddress
	Alias   *Alias
	len     int
}

// NewRecipientFromAddress creates the Recipient from given address.
func NewRecipientFromAddress(a WavesAddress) Recipient {
	return Recipient{Address: &a, len: WavesAddressSize}
}

// NewRecipientFromAlias creates a Recipient with the given Alias inside.
func NewRecipientFromAlias(a Alias) Recipient {
	return Recipient{Alias: &a, len: aliasFixedSize + len(a.Alias)}
}

func NewRecipientFromString(s string) (Recipient, error) {
	if strings.Contains(s, AliasPrefix) {
		a, err := NewAliasFromString(s)
		if err != nil {
			return Recipient{}, err
		}
		return NewRecipientFromAlias(*a), nil
	}
	a, err := NewAddressFromString(s)
	if err != nil {
		return Recipient{}, err
	}
	return NewRecipientFromAddress(a), nil
}

func (r Recipient) Eq(r2 Recipient) bool {
	res := r.len == r2.len
	if r.Address != nil && r2.Address != nil {
		res = res && (*r.Address == *r2.Address)
	} else {
		res = res && (r.Address == nil)
		res = res && (r2.Address == nil)
	}
	if r.Alias != nil && r2.Alias != nil {
		res = res && (*r.Alias == *r2.Alias)
	} else {
		res = res && (r.Alias == nil)
		res = res && (r2.Alias == nil)
	}
	return res
}

func (r Recipient) ToProtobuf() (*g.Recipient, error) {
	if r.Address == nil {
		return &g.Recipient{Recipient: &g.Recipient_Alias{Alias: r.Alias.Alias}}, nil
	}
	addrBody := r.Address.Body()
	return &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: addrBody}}, nil
}

// Valid checks that either an WavesAddress or an Alias is set then checks the validity of the set field.
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
	if strings.Contains(s, AliasPrefix) {
		var a Alias
		err := a.UnmarshalJSON(value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from JSON")
		}
		r.Alias = &a
		r.len = aliasFixedSize + len(a.Alias)
		return nil
	}
	var a WavesAddress
	err := a.UnmarshalJSON(value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Recipient from JSON")
	}
	r.Address = &a
	r.len = WavesAddressSize
	return nil
}

func (r *Recipient) BinarySize() int {
	return r.len
}

// MarshalBinary makes bytes of the Recipient.
func (r *Recipient) MarshalBinary() ([]byte, error) {
	if r.Alias != nil {
		return r.Alias.MarshalBinary()
	}
	return r.Address[:], nil
}

func (r *Recipient) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := r.Serialize(s)
	if err != nil {
		return 0, err
	}
	return s.N(), nil
}

func (r *Recipient) Serialize(s *serializer.Serializer) error {
	if r.Alias != nil {
		return r.Alias.Serialize(s)
	}
	err := s.Bytes(r.Address[:])
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalBinary reads the Recipient from bytes. Validates the result.
func (r *Recipient) UnmarshalBinary(data []byte) error {
	switch v := data[0]; v {
	case wavesAddressVersion:
		a, err := NewAddressFromBytes(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from bytes")
		}
		r.Address = &a
		r.len = WavesAddressSize
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

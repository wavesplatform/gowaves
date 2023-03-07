package proto

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"golang.org/x/crypto/sha3"
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

func (a AddressID) ToWavesAddress(scheme Scheme) (WavesAddress, error) {
	return newWavesAddress(scheme, a)
}

type Address interface {
	ID() AddressID
	Bytes() []byte
	String() string
	Equal(address Address) bool
	ToWavesAddress(scheme Scheme) (WavesAddress, error)
}

// EthereumAddress is the first 20 bytes of Public Key's hash for the Waves address, or the 20 bytes of an Ethereum address.
type EthereumAddress [EthereumAddressSize]byte

func NewEthereumAddressFromHexString(s string) (EthereumAddress, error) {
	b, err := DecodeFromHexString(s)
	if err != nil {
		return EthereumAddress{}, err
	}
	return NewEthereumAddressFromBytes(b)
}

func NewEthereumAddressFromBytes(b []byte) (EthereumAddress, error) {
	if len(b) != EthereumAddressSize {
		return EthereumAddress{},
			errors.Errorf("invalid EthereumAddress size: got %d, want %d", len(b), EthereumAddressSize)
	}
	var addr EthereumAddress
	copy(addr[:], b)
	return addr, nil
}

// BytesToEthereumAddress returns EthereumAddress with value b.
// If b is larger than len(h), b will be cropped from the left.
func BytesToEthereumAddress(b []byte) EthereumAddress {
	var a EthereumAddress
	a.setBytes(b)
	return a
}

// Bytes converts the fixed-length byte array of the EthereumAddress to a slice of bytes.
func (ea EthereumAddress) Bytes() []byte {
	return ea[:]
}

// Bytes converts the fixed-length byte array of the EthereumAddress to a slice of bytes.
// If *EthereumAddress == nil copy returns nil.
func (ea *EthereumAddress) tryToBytes() []byte {
	if ea == nil {
		return nil
	}
	return ea.Bytes()
}

// setBytes sets bytes to EthereumAddress with right side priority.
func (ea *EthereumAddress) setBytes(b []byte) {
	if len(b) > len(ea) {
		b = b[len(b)-EthereumAddressSize:]
	}
	copy(ea[EthereumAddressSize-len(b):], b)
}

func (ea EthereumAddress) ID() AddressID {
	var id AddressID
	copy(id[:], ea[:])
	return id
}

// Hash converts an address to a EthereumHash by left-padding it with zeros.
func (ea EthereumAddress) Hash() EthereumHash {
	return BytesToEthereumHash(ea[:])
}

func (ea EthereumAddress) Hex() string {
	return string(ea.checksumHex())
}

func (ea EthereumAddress) String() string {
	return ea.Hex()
}

func (ea EthereumAddress) Equal(address Address) bool {
	switch other := address.(type) {
	case EthereumAddress, *EthereumAddress:
		return bytes.Equal(ea.Bytes(), other.Bytes())
	case WavesAddress, *WavesAddress:
		return false
	default:
		panic(errors.Errorf("BUG, CREATE REPORT: unknown address type %T", address))
	}
}

func (ea EthereumAddress) ToWavesAddress(scheme Scheme) (WavesAddress, error) {
	return newWavesAddress(scheme, ea)
}

func (ea EthereumAddress) MarshalJSON() ([]byte, error) {
	hexString := ea.Hex()
	return []byte(fmt.Sprintf("\"%s\"", hexString)), nil
}

func (ea *EthereumAddress) UnmarshalJSON(bytes []byte) error {
	hexString, err := strconv.Unquote(string(bytes))
	if err != nil {
		return errors.Wrap(err, "quotes are required")
	}
	addr, err := NewEthereumAddressFromHexString(hexString)
	if err != nil {
		return err
	}
	*ea = addr
	return nil
}

func (ea *EthereumAddress) checksumHex() []byte {
	buf := []byte(EncodeToHexString(ea[:]))

	// compute checksum
	sha := sha3.NewLegacyKeccak256()
	// nickeskov: can't fail
	_, _ = sha.Write(buf[2:])
	hash := sha.Sum(nil)
	for i := 2; i < len(buf); i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if buf[i] > '9' && hashByte > 7 {
			buf[i] -= 32
		}
	}
	return buf[:]
}

// copy returns an exact copy of the provided EthereumAddress.
// If *EthereumAddress == nil copy returns nil.
func (ea *EthereumAddress) copy() *EthereumAddress {
	if ea == nil {
		return nil
	}
	cpy := *ea
	return &cpy
}

func (ea *EthereumAddress) unmarshalFromFastRLP(val *fastrlp.Value) error {
	if err := val.GetAddr(ea[:]); err != nil {
		return errors.Wrap(err, "failed to unmarshal EthereumAddress from fastRLP value")
	}
	return nil
}

func (ea *EthereumAddress) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	return arena.NewBytes(ea.Bytes())
}

// WavesAddress is the transformed Public Key with additional bytes of the version, a blockchain scheme and a checksum.
type WavesAddress [WavesAddressSize]byte

func (a WavesAddress) MarshalText() (text []byte, err error) {
	return []byte(a.String()), nil
}

func (a *WavesAddress) UnmarshalText(text []byte) error {
	addr, err := NewAddressFromString(string(text))
	if err != nil {
		return err
	}
	*a = addr
	return nil
}

func (a WavesAddress) Body() []byte {
	return a[wavesAddressHeaderSize : wavesAddressHeaderSize+wavesAddressBodySize]
}

func (a WavesAddress) Scheme() Scheme {
	return a[1]
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

func (a WavesAddress) Equal(address Address) bool {
	switch other := address.(type) {
	case WavesAddress, *WavesAddress:
		return bytes.Equal(a.Bytes(), other.Bytes())
	case EthereumAddress, *EthereumAddress:
		return false
	default:
		panic(errors.Errorf("BUG, CREATE REPORT: unknown address type %T", address))
	}
}

func (a WavesAddress) ToWavesAddress(_ Scheme) (WavesAddress, error) {
	return a, nil
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

func (a *WavesAddress) EthereumAddress() EthereumAddress {
	return EthereumAddress(a.ID())
}

// NewAddressFromPublicKey produces an WavesAddress from given scheme and Public Key bytes.
func NewAddressFromPublicKey(scheme byte, publicKey crypto.PublicKey) (WavesAddress, error) {
	h, err := crypto.SecureHash(publicKey[:])
	if err != nil {
		return WavesAddress{}, errors.Wrap(err, "failed to produce Digest from PublicKey")
	}
	return newAddressFromPublicKeyHash(scheme, h)
}

// newAddressFromPublicKeyHash produces an WavesAddress from given public key hash.
func newAddressFromPublicKeyHash(scheme Scheme, pubKeyHash crypto.Digest) (WavesAddress, error) {
	var body [wavesAddressBodySize]byte
	copy(body[:], pubKeyHash[:])
	return newWavesAddress(scheme, body)
}

// newWavesAddress produces an WavesAddress from given body (AddressID).
func newWavesAddress(scheme Scheme, body [wavesAddressBodySize]byte) (WavesAddress, error) {
	var addr WavesAddress
	addr[0] = wavesAddressVersion
	addr[1] = scheme
	copy(addr[wavesAddressHeaderSize:], body[:])
	checksum, err := addressChecksum(addr[:wavesAddressHeaderSize+wavesAddressBodySize])
	if err != nil {
		return addr, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	copy(addr[wavesAddressHeaderSize+wavesAddressBodySize:], checksum[:])
	return addr, nil
}

// NewAddressLikeFromAnyBytes produces an WavesAddress from given scheme and bytes.
func NewAddressLikeFromAnyBytes(scheme byte, b []byte) (WavesAddress, error) {
	h, err := crypto.SecureHash(b)
	if err != nil {
		return WavesAddress{}, errors.Wrap(err, "failed to produce Digest from any bytes")
	}
	return newAddressFromPublicKeyHash(scheme, h)
}

func MustAddressFromPublicKey(scheme byte, publicKey crypto.PublicKey) WavesAddress {
	rs, err := NewAddressFromPublicKey(scheme, publicKey)
	if err != nil {
		panic(err)
	}
	return rs
}

func RebuildAddress(scheme Scheme, bodyBytes []byte) (WavesAddress, error) {
	if len(bodyBytes) == WavesAddressSize {
		return NewAddressFromBytesChecked(scheme, bodyBytes)
	}
	if l := len(bodyBytes); l != wavesAddressBodySize {
		return WavesAddress{}, errors.Errorf("%d is unexpected address' body size", l)
	}
	var body [wavesAddressBodySize]byte
	copy(body[:], bodyBytes[:wavesAddressBodySize])
	return newWavesAddress(scheme, body)
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

// NewAddressFromBytes creates an WavesAddress from the slice of bytes and checks that the result address is a valid address.
func NewAddressFromBytes(b []byte) (WavesAddress, error) {
	var a WavesAddress
	if l := len(b); l < WavesAddressSize {
		return a, errors.Errorf("insufficient array length %d, expected at least %d", l, WavesAddressSize)
	}
	copy(a[:], b[:WavesAddressSize])
	if ok, err := a.validVersionAndChecksum(); !ok {
		return a, errors.Wrap(err, "invalid address")
	}
	return a, nil
}

// NewAddressFromBytesChecked creates an WavesAddress from the slice of bytes and checks that the result address
// is a valid address with valid scheme.
func NewAddressFromBytesChecked(scheme Scheme, b []byte) (WavesAddress, error) {
	var a WavesAddress
	if l := len(b); l < WavesAddressSize {
		return a, errors.Errorf("insufficient array length %d, expected at least %d", l, WavesAddressSize)
	}
	copy(a[:], b[:WavesAddressSize])
	if ok, err := a.Valid(scheme); !ok {
		return a, errors.Wrap(err, "invalid address")
	}
	return a, nil
}

// Valid checks that scheme, version and checksum of the WavesAddress are correct.
func (a *WavesAddress) Valid(scheme Scheme) (bool, error) {
	if s := a.Scheme(); s != scheme {
		return false, errors.Errorf("invalid scheme %q, expected %q", s, scheme)
	}
	return a.validVersionAndChecksum()
}

// validVersionAndChecksum checks that version and checksum of the WavesAddress are correct.
func (a *WavesAddress) validVersionAndChecksum() (bool, error) {
	if a[0] != wavesAddressVersion {
		return false, errors.Errorf("unsupported address version %d", a[0])
	}
	hb := a[:wavesAddressHeaderSize+wavesAddressBodySize]
	ec, err := addressChecksum(hb)
	if err != nil {
		return false, errors.Wrap(err, "failed to calculate WavesAddress checksum")
	}
	ac := a[wavesAddressHeaderSize+wavesAddressBodySize:]
	if !bytes.Equal(ec[:], ac) {
		return false, errors.New("invalid WavesAddress checksum")
	}
	return true, nil
}

// Bytes converts the fixed-length byte array of the WavesAddress to a slice of bytes.
func (a WavesAddress) Bytes() []byte {
	return a[:]
}

func addressChecksum(b []byte) (cs [wavesAddressChecksumSize]byte, err error) {
	h, err := crypto.SecureHash(b)
	if err != nil {
		return cs, err
	}
	copy(cs[:], h[:wavesAddressChecksumSize])
	return cs, nil
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
	if err := s.Byte(a.Version); err != nil {
		return err
	}
	if err := s.Byte(a.Scheme); err != nil {
		return err
	}
	if err := s.StringWithUInt16Len(a.Alias); err != nil {
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
func (a Alias) Valid(scheme Scheme) (bool, error) {
	if v := a.Version; v != aliasVersion {
		return false, errors.Errorf("%d is incorrect alias version, expected %d", v, aliasVersion)
	}
	if s := a.Scheme; s != scheme {
		return false, errs.NewTxValidationError(fmt.Sprintf("invalid scheme %q, expected %q", s, scheme))
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

type recipient interface {
	Alias() *Alias
	Address() *WavesAddress
	BinarySize() int
	Eq(r2 Recipient) bool
	EqAddr(addr WavesAddress) (bool, error)
	EqAlias(alias Alias) (bool, error)
	Serialize(s *serializer.Serializer) error
	Valid(scheme Scheme) (bool, error)
	ToProtobuf() (*g.Recipient, error)
	fmt.Stringer
	json.Marshaler
	encoding.BinaryMarshaler
}

type aliasRecipient struct {
	al Alias
}

func (r *aliasRecipient) Alias() *Alias {
	return &r.al
}

func (r *aliasRecipient) Address() *WavesAddress {
	return nil
}

func (r *aliasRecipient) Eq(r2 Recipient) bool {
	if al2 := r2.Alias(); al2 != nil {
		return r.al == *al2
	}
	return false
}

func (r *aliasRecipient) EqAddr(addr WavesAddress) (bool, error) {
	return false, errors.Errorf("failed to compare recipient '%s' with addr '%s'", r.String(), addr.String())
}

func (r *aliasRecipient) EqAlias(alias Alias) (bool, error) {
	return r.al == alias, nil
}

func (r *aliasRecipient) Serialize(s *serializer.Serializer) error {
	return r.al.Serialize(s)
}

func (r *aliasRecipient) BinarySize() int {
	return r.al.BinarySize()
}

func (r *aliasRecipient) Valid(scheme Scheme) (bool, error) {
	return r.al.Valid(scheme)
}

func (r *aliasRecipient) ToProtobuf() (*g.Recipient, error) {
	return &g.Recipient{Recipient: &g.Recipient_Alias{Alias: r.al.Alias}}, nil
}

func (r *aliasRecipient) String() string {
	return r.al.String()
}

func (r *aliasRecipient) MarshalJSON() ([]byte, error) {
	return r.al.MarshalJSON()
}

func (r *aliasRecipient) MarshalBinary() (data []byte, err error) {
	return r.al.MarshalBinary()
}

type wavesAddressRecipient struct {
	addr WavesAddress
}

func (r *wavesAddressRecipient) Alias() *Alias {
	return nil
}

func (r *wavesAddressRecipient) Address() *WavesAddress {
	return &r.addr
}

func (r *wavesAddressRecipient) BinarySize() int {
	return WavesAddressSize
}

func (r *wavesAddressRecipient) Eq(r2 Recipient) bool {
	if addr2 := r2.Address(); addr2 != nil {
		return bytes.Equal(r.addr[:], addr2[:])
	}
	return false
}

func (r *wavesAddressRecipient) EqAddr(addr WavesAddress) (bool, error) {
	return bytes.Equal(r.addr[:], addr[:]), nil
}

func (r *wavesAddressRecipient) EqAlias(alias Alias) (bool, error) {
	return false, errors.Errorf("failed to compare recipient '%s' with alias '%s'", r.String(), alias.String())
}

func (r *wavesAddressRecipient) Serialize(s *serializer.Serializer) error {
	return s.Bytes(r.addr[:])
}

func (r *wavesAddressRecipient) Valid(scheme Scheme) (bool, error) {
	return r.addr.Valid(scheme)
}

func (r *wavesAddressRecipient) String() string {
	return r.addr.String()
}

func (r *wavesAddressRecipient) ToProtobuf() (*g.Recipient, error) {
	return &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: r.addr.Body()}}, nil
}

func (r *wavesAddressRecipient) MarshalJSON() ([]byte, error) {
	return r.addr.MarshalJSON()
}

func (r *wavesAddressRecipient) MarshalBinary() (data []byte, err error) {
	return r.addr[:], nil
}

// incomparable is a zero-width, non-comparable type. Adding it to a struct
// makes that struct also non-comparable, and generally doesn't add
// any size (as long as it's first).
type incomparable [0]func()

// Recipient could be an Alias or an WavesAddress.
type Recipient struct {
	_     incomparable
	inner recipient
}

// NewRecipientFromAddress creates the Recipient from given address.
func NewRecipientFromAddress(a WavesAddress) Recipient {
	return Recipient{inner: &wavesAddressRecipient{a}}
}

// NewRecipientFromAlias creates a Recipient with the given Alias inside.
func NewRecipientFromAlias(a Alias) Recipient {
	return Recipient{inner: &aliasRecipient{a}}
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

func (r Recipient) Alias() *Alias {
	return r.inner.Alias()
}

func (r Recipient) Address() *WavesAddress {
	return r.inner.Address()
}

func (r *Recipient) BinarySize() int {
	return r.inner.BinarySize()
}

func (r Recipient) Eq(r2 Recipient) bool {
	return r.inner.Eq(r2)
}

func (r Recipient) EqAddr(addr WavesAddress) (bool, error) {
	return r.inner.EqAddr(addr)
}

func (r Recipient) EqAlias(alias Alias) (bool, error) {
	return r.inner.EqAlias(alias)
}

func (r Recipient) ToProtobuf() (*g.Recipient, error) {
	return r.inner.ToProtobuf()
}

// Valid checks that either an WavesAddress or an Alias is set then checks the validity of the set field.
func (r Recipient) Valid(scheme Scheme) (bool, error) {
	return r.inner.Valid(scheme)
}

// MarshalJSON converts the Recipient to its JSON representation.
func (r Recipient) MarshalJSON() ([]byte, error) {
	return r.inner.MarshalJSON()
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
		*r = NewRecipientFromAlias(a)
		return nil
	}
	var a WavesAddress
	err := a.UnmarshalJSON(value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Recipient from JSON")
	}
	*r = NewRecipientFromAddress(a)
	return nil
}

// MarshalBinary makes bytes of the Recipient.
func (r *Recipient) MarshalBinary() ([]byte, error) {
	return r.inner.MarshalBinary()
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
	return r.inner.Serialize(s)
}

// UnmarshalBinary reads the Recipient from bytes. Validates the result.
func (r *Recipient) UnmarshalBinary(data []byte) error {
	switch v := data[0]; v {
	case wavesAddressVersion:
		a, err := NewAddressFromBytes(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from bytes")
		}
		*r = NewRecipientFromAddress(a)
		return nil
	case aliasVersion:
		var a Alias
		err := a.UnmarshalBinary(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Recipient from bytes")
		}
		*r = NewRecipientFromAlias(a)
		return nil
	default:
		return errors.Errorf("unsupported Recipient version %d", v)
	}
}

// String gives the string representation of the Recipient.
func (r *Recipient) String() string {
	return r.inner.String()
}

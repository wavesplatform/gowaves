package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	//WavesAssetName is the default name for basic WAVES asset.
	WavesAssetName                           = "WAVES"
	quotedWavesAssetName                     = "\"" + WavesAssetName + "\""
	orderLen                                 = crypto.PublicKeySize + crypto.PublicKeySize + 1 + 1 + 1 + 8 + 8 + 8 + 8 + 8
	orderV2FixedBodyLen                      = 1 + orderLen
	orderV3FixedBodyLen                      = 1 + orderLen + 1
	orderV1MinLen                            = crypto.SignatureSize + orderLen
	orderV2MinLen                            = orderV2FixedBodyLen + proofsMinLen
	orderV3MinLen                            = orderV3FixedBodyLen + proofsMinLen
	jsonNull                                 = "null"
	integerArgumentLen                       = 1 + 8
	booleanArgumentLen                       = 1
	binaryArgumentMinLen                     = 1 + 4
	stringArgumentMinLen                     = 1 + 4
	listArgumentMinLen                       = 1 + 4
	PriceConstant                            = 100000000
	MaxOrderAmount                           = 100 * PriceConstant * PriceConstant
	MaxOrderTTL                              = uint64((30 * 24 * time.Hour) / time.Millisecond)
	MaxKeySize                               = 100
	MaxPBKeySize                             = 400
	MaxDataWithProofsBytes                   = 150 * 1024
	MaxDataWithProofsProtoBytes              = 165_890
	MaxDataWithProofsV6PayloadBytes          = 165_835 // (DataEntry.MaxPBKeySize + DataEntry.MaxValueSize) * 5
	maxDataEntryValueSize                    = 32767
	MaxDataEntriesScriptActionsSizeInBytesV1 = 5 * 1024
	MaxDataEntriesScriptActionsSizeInBytesV2 = 15 * 1024
	MaxScriptActionsV1                       = 10
	MaxScriptActionsV2                       = 30
	MaxDataEntryScriptActions                = 100
	MaxBalanceScriptActionsV3                = 100
	MaxAttachedPaymentsScriptActions         = 100
	MaxAssetScriptActionsV3                  = 30
	base64EncodingSizeLimit                  = 1024
	base64EncodingPrefix                     = "base64:"
)

type Timestamp = uint64
type Score = big.Int
type Scheme = byte

type Height = uint64

var jsonNullBytes = []byte(jsonNull)

type Bytes []byte

func (a Bytes) WriteTo(w io.Writer) (int64, error) {
	rs, err := w.Write(a)
	return int64(rs), err
}

// B58Bytes represents bytes as Base58 string in JSON
type B58Bytes []byte

// String represents underlying bytes as Base58 string
func (b B58Bytes) String() string {
	return base58.Encode(b)
}

// MarshalJSON writes B58Bytes Value as JSON string
func (b B58Bytes) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(b), nil
}

// UnmarshalJSON reads B58Bytes from JSON string
func (b *B58Bytes) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		*b = nil
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal B58Bytes from JSON")
	}
	if s == "" {
		*b = []byte{}
		return nil
	}
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode B58Bytes")
	}
	*b = v
	return nil
}

func (b B58Bytes) Bytes() []byte {
	return b
}

type B64Bytes []byte

func (b B64Bytes) String() string {
	return base64.StdEncoding.EncodeToString(b)
}

func (b B64Bytes) MarshalJSON() ([]byte, error) {
	return common.ToBase64JSON(b), nil
}

func (b *B64Bytes) UnmarshalJSON(value []byte) error {
	str := string(value)
	if str == jsonNull {
		return nil
	}
	s, err := strconv.Unquote(str)
	if err != nil {
		*b = nil
		return errors.Wrap(err, "failed to unmarshal B64Bytes from JSON")
	}
	if s == "" {
		*b = []byte{}
		return nil
	}
	v, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode B64Bytes")
	}
	*b = v
	return nil
}

func (b B64Bytes) Bytes() []byte {
	return b
}

type HexBytes []byte

// String represents underlying bytes as Hex string with 0x prefix
func (b HexBytes) String() string {
	return EncodeToHexString(b)
}

// MarshalJSON writes HexBytes Value as JSON string
func (b HexBytes) MarshalJSON() ([]byte, error) {
	s := b.String()
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes(), nil
}

// UnmarshalJSON reads HexBytes from JSON string
func (b *HexBytes) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		*b = nil
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal HexBytes from JSON")
	}
	if s == "" {
		*b = []byte{}
		return nil
	}
	v, err := DecodeFromHexString(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode HexBytes")
	}
	*b = v
	return nil
}

func (b HexBytes) Bytes() []byte {
	return b
}

type ByteVector []byte

// String represents underlying bytes as Base58 string or Base64 string with additional prefix.
func (v ByteVector) String() string {
	if len(v) < base64EncodingSizeLimit {
		return v.encodeBase58()
	}
	return v.encodeBase64()
}

func (v ByteVector) encodeBase58() string {
	return base58.Encode(v)
}

func (v ByteVector) encodeBase64() string {
	return base64EncodingPrefix + base64.StdEncoding.EncodeToString(v)
}

// MarshalJSON writes ByteVector Value as JSON string
func (v ByteVector) MarshalJSON() ([]byte, error) {
	s := v.String()
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes(), nil
}

// UnmarshalJSON reads ByteVector from JSON string
func (v *ByteVector) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		*v = nil
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ByteVector from JSON")
	}
	if s == "" {
		*v = []byte{}
		return nil
	}
	if strings.HasPrefix(s, base64EncodingPrefix) {
		s = strings.TrimPrefix(s, base64EncodingPrefix)
		err := v.decodeFromBase64String(s)
		if err != nil {
			return errors.Wrap(err, "failed to decode ByteVector from Base64 string")
		}
		return nil
	}
	err = v.decodeFromBase58String(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode ByteVector from Base58 string")
	}
	return nil
}

func (v *ByteVector) decodeFromBase64String(s string) error {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*v = b
	return nil
}

func (v *ByteVector) decodeFromBase58String(s string) error {
	b, err := base58.Decode(s)
	if err != nil {
		return err
	}
	*v = b
	return nil
}

func (v ByteVector) Bytes() []byte {
	return v
}

// OptionalAsset represents an optional asset identification
type OptionalAsset struct {
	Present bool
	ID      crypto.Digest
}

// NewOptionalAssetFromString creates an OptionalAsset structure from its string representation.
func NewOptionalAssetFromString(s string) (*OptionalAsset, error) {
	switch strings.ToUpper(s) {
	case WavesAssetName, "":
		return &OptionalAsset{Present: false}, nil
	default:
		d, err := crypto.NewDigestFromBase58(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create OptionalAsset from Base58 string")
		}
		return NewOptionalAssetFromDigest(d), nil
	}
}

// NewOptionalAssetFromBytes parses bytes as crypto.Digest and returns OptionalAsset.
func NewOptionalAssetFromBytes(b []byte) (*OptionalAsset, error) {
	d, err := crypto.NewDigestFromBytes(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OptionalAsset from bytes")
	}
	return NewOptionalAssetFromDigest(d), nil
}

func NewOptionalAsset(present bool, id crypto.Digest) OptionalAsset {
	return OptionalAsset{Present: present, ID: id}
}

func NewOptionalAssetFromDigest(d crypto.Digest) *OptionalAsset {
	return &OptionalAsset{Present: true, ID: d}
}

func NewOptionalAssetWaves() OptionalAsset {
	return OptionalAsset{Present: false}
}

// String method converts OptionalAsset to its text representation
func (a OptionalAsset) String() string {
	if a.Present {
		return a.ID.String()
	}
	return WavesAssetName
}

// MarshalJSON writes OptionalAsset as a JSON string Value
func (a OptionalAsset) MarshalJSON() ([]byte, error) {
	if a.Present {
		return a.ID.MarshalJSON()
	}
	return []byte(jsonNull), nil
}

// UnmarshalJSON reads OptionalAsset from a JSON string Value
func (a *OptionalAsset) UnmarshalJSON(value []byte) error {
	s := strings.ToUpper(string(value))
	switch s {
	case "NULL", quotedWavesAssetName:
		*a = NewOptionalAssetWaves()
	default:
		var d crypto.Digest
		err := d.UnmarshalJSON(value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal OptionalAsset")
		}
		*a = *NewOptionalAssetFromDigest(d)
	}
	return nil
}

func (a OptionalAsset) BinarySize() int {
	s := 1
	if a.Present {
		s += crypto.DigestSize
	}
	return s
}

// MarshalBinary marshals the optional asset to its binary representation.
func (a OptionalAsset) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	PutBool(buf, a.Present)
	if a.Present {
		copy(buf[1:], a.ID[:])
	}
	return buf, nil
}

// WriteTo writes its binary representation.
func (a OptionalAsset) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := a.Serialize(s)
	if err != nil {
		return 0, err
	}
	return s.N(), nil
}

// Serialize into binary representation.
func (a OptionalAsset) Serialize(s *serializer.Serializer) error {
	err := s.Bool(a.Present)
	if err != nil {
		return err
	}
	if a.Present {
		err = s.Bytes(a.ID[:])
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalBinary reads the OptionalAsset structure from its binary representation.
func (a *OptionalAsset) UnmarshalBinary(data []byte) error {
	var err error
	a.Present, err = Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OptionalAsset")
	}
	if a.Present {
		data = data[1:]
		if l := len(data); l < crypto.DigestSize {
			return errors.Errorf("not enough data for OptionalAsset value, expected %d, received %d", crypto.DigestSize, l)
		}
		copy(a.ID[:], data[:crypto.DigestSize])
	}
	return nil
}

func (a *OptionalAsset) ToID() []byte {
	if a.Present {
		return a.ID[:]
	}
	return nil
}

func (a *OptionalAsset) ToDigest() *crypto.Digest {
	if a.Present {
		return &a.ID
	}
	return nil
}

func (a OptionalAsset) Eq(b OptionalAsset) bool {
	return a.Present == b.Present && a.ID == b.ID
}

type Attachment []byte

func (a Attachment) Size() int {
	return len(a)
}

func (a Attachment) Bytes() ([]byte, error) {
	return a, nil
}

func (a Attachment) MarshalJSON() ([]byte, error) {
	return json.Marshal(base58.Encode(a))
}

func (a *Attachment) UnmarshalJSON(data []byte) error {
	*a = Attachment{}
	if len(data) == 0 {
		return nil
	}
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return errors.Wrap(err, "unmarshal")
	}
	if s == "" {
		return nil
	}
	rs, err := base58.Decode(s)
	if err != nil {
		return err
	}
	*a = rs
	return nil
}

func NewAttachmentFromBase58(s string) (Attachment, error) {
	return base58.Decode(s)
}

// OrderType an alias for byte that encodes the type of OrderV1 (BUY|SELL).
type OrderType byte

func (t OrderType) ToProtobuf() g.Order_Side {
	if t == Buy {
		return g.Order_BUY
	}
	return g.Order_SELL
}

// iota: reset
const (
	Buy OrderType = iota
	Sell
)

const (
	buyOrderName  = "buy"
	sellOrderName = "sell"
)

func (t OrderType) String() string {
	switch t {
	case Buy:
		return buyOrderName
	case Sell:
		return sellOrderName
	default:
		return fmt.Sprintf("BUG, CREATE REPORT: unknown order type (%d)", t)
	}
}

// MarshalJSON writes value of OrderType to JSON representation.
func (t OrderType) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(t.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads the OrderType value from JSON value.
func (t *OrderType) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderType from JSON")
	}
	switch strings.ToLower(s) {
	case buyOrderName:
		*t = Buy
	case sellOrderName:
		*t = Sell
	default:
		return errors.Errorf("incorrect OrderType '%s'", s)
	}
	return nil
}

// AssetPair is a pair of assets in ExchangeTransaction.
type AssetPair struct {
	AmountAsset OptionalAsset `json:"amountAsset"`
	PriceAsset  OptionalAsset `json:"priceAsset"`
}

func (p AssetPair) BinarySize() int {
	return p.AmountAsset.BinarySize() + p.PriceAsset.BinarySize()
}

func (p AssetPair) ToProtobuf() *g.AssetPair {
	return &g.AssetPair{AmountAssetId: p.AmountAsset.ToID(), PriceAssetId: p.PriceAsset.ToID()}
}

type OrderPriceMode byte

const (
	OrderPriceModeDefault OrderPriceMode = iota
	OrderPriceModeFixedDecimals
	OrderPriceModeAssetDecimals
)

func (m *OrderPriceMode) UnmarshalJSON(val []byte) error {
	switch quotedMode := string(val); quotedMode {
	case jsonNull, "\"default\"":
		*m = OrderPriceModeDefault
	case "\"fixedDecimals\"":
		*m = OrderPriceModeFixedDecimals
	case "\"assetDecimals\"":
		*m = OrderPriceModeAssetDecimals
	default:
		return errors.Errorf("invalid OrderPriceMode=%s", quotedMode)
	}
	return nil
}

func (m OrderPriceMode) MarshalJSON() ([]byte, error) {
	if !m.isValidOrderPriceValue() {
		return nil, errors.Errorf("invalid OrderPriceMode=%d", byte(m))
	}
	switch m {
	case OrderPriceModeDefault:
		return []byte(jsonNull), nil
	default:
		return []byte(fmt.Sprintf("\"%s\"", m.String())), nil
	}
}

func (m OrderPriceMode) String() string {
	switch m {
	case OrderPriceModeDefault:
		return "default"
	case OrderPriceModeFixedDecimals:
		return "fixedDecimals"
	case OrderPriceModeAssetDecimals:
		return "assetDecimals"
	default:
		return fmt.Sprintf("BUG, CREATE REPORT: invalid OrderPriceMode=%d", byte(m))
	}
}

func (m OrderPriceMode) upperSnakeCaseString() string {
	switch m {
	case OrderPriceModeDefault:
		return "DEFAULT"
	case OrderPriceModeFixedDecimals:
		return "FIXED_DECIMALS"
	case OrderPriceModeAssetDecimals:
		return "ASSET_DECIMALS"
	default:
		return fmt.Sprintf("BUG, CREATE REPORT: invalid OrderPriceMode=%d", byte(m))
	}
}

func (m *OrderPriceMode) FromProtobuf(gm g.Order_PriceMode) error {
	switch gm {
	case g.Order_DEFAULT:
		*m = OrderPriceModeDefault
	case g.Order_FIXED_DECIMALS:
		*m = OrderPriceModeFixedDecimals
	case g.Order_ASSET_DECIMALS:
		*m = OrderPriceModeAssetDecimals
	default:
		return errors.Errorf("invalid protobuf Order_PriceMode=%v", gm)
	}
	return nil
}

func (m OrderPriceMode) ToProtobuf() g.Order_PriceMode {
	switch m {
	case OrderPriceModeDefault:
		return g.Order_DEFAULT
	case OrderPriceModeFixedDecimals:
		return g.Order_FIXED_DECIMALS
	case OrderPriceModeAssetDecimals:
		return g.Order_ASSET_DECIMALS
	default:
		panic(fmt.Sprintf("BUG, CREATE REPORT: invalid OrderPriceMode=%d", byte(m)))
	}
}

func (m OrderPriceMode) isValidOrderPriceValue() bool {
	switch m {
	case OrderPriceModeDefault, OrderPriceModeFixedDecimals, OrderPriceModeAssetDecimals:
		return true
	default:
		return false
	}
}

func (m OrderPriceMode) Valid(orderVersion byte) (bool, error) {
	switch orderVersion {
	case 1, 2, 3:
		if m != OrderPriceModeDefault {
			return false, errors.Errorf("OrderV%d.PriceMode must be %q",
				orderVersion, OrderPriceModeDefault.String(),
			)
		}
	default:
		if !m.isValidOrderPriceValue() {
			return false, errors.Errorf("invalid OrderPriceMode = %d", byte(m))
		}
	}
	return true, nil
}

type Order interface {
	GetID() ([]byte, error)
	GetVersion() byte
	GetPriceMode() OrderPriceMode
	GetOrderType() OrderType
	GetMatcherPK() crypto.PublicKey
	GetAssetPair() AssetPair
	GetPrice() uint64
	GetExpiration() uint64
	Valid() (bool, error)
	GetAmount() uint64
	GetTimestamp() uint64
	GetMatcherFee() uint64
	GetMatcherFeeAsset() OptionalAsset
	GetSenderPKBytes() []byte
	GetSender(scheme Scheme) (Address, error)
	GenerateID(scheme Scheme) error
	GetProofs() (*ProofsV1, error)
	Verify(Scheme) (bool, error)
	ToProtobuf(Scheme) *g.Order
	ToProtobufSigned(Scheme) *g.Order
	BinarySize() int
}

func MarshalOrderBody(scheme Scheme, o Order) (data []byte, err error) {
	switch version := o.GetVersion(); version {
	case 1:
		o, ok := o.(*OrderV1)
		if !ok {
			return nil, errors.New("failed to cast an order version 1 to *OrderV1")
		}
		return o.BodyMarshalBinary()
	case 2:
		o, ok := o.(*OrderV2)
		if !ok {
			return nil, errors.New("failed to cast an order version 2 to *OrderV2")
		}
		return o.BodyMarshalBinary()
	case 3:
		o, ok := o.(*OrderV3)
		if !ok {
			return nil, errors.New("failed to cast an order version 3 to *OrderV3")
		}
		return o.BodyMarshalBinary()
	case 4:
		switch o := o.(type) {
		case *OrderV4:
			data, err = o.BodyMarshalBinary(scheme)
		case *EthereumOrderV4:
			data, err = o.BodyMarshalBinary(scheme)
		default:
			return nil, errors.New("failed to cast an order version 4 to *OrderV4 or *EthereumOrderV4")
		}
		return data, err
	default:
		return nil, errors.Errorf("invalid order version %d", version)
	}
}

type OrderBody struct {
	SenderPK   crypto.PublicKey `json:"senderPublicKey"`
	MatcherPK  crypto.PublicKey `json:"matcherPublicKey"`
	AssetPair  AssetPair        `json:"assetPair"`
	OrderType  OrderType        `json:"orderType"`
	Price      uint64           `json:"price"`
	Amount     uint64           `json:"amount"`
	Timestamp  uint64           `json:"timestamp"`
	Expiration uint64           `json:"expiration"`
	MatcherFee uint64           `json:"matcherFee"`
}

func (o OrderBody) BinarySize() int {
	return crypto.PublicKeySize*2 + 40 + o.AssetPair.BinarySize() + 1
}

func (o OrderBody) ToProtobuf(scheme Scheme) *g.Order {
	return &g.Order{
		ChainId:          int32(scheme),
		Sender:           &g.Order_SenderPublicKey{SenderPublicKey: o.SenderPK.Bytes()},
		MatcherPublicKey: o.MatcherPK.Bytes(),
		AssetPair:        o.AssetPair.ToProtobuf(),
		OrderSide:        o.OrderType.ToProtobuf(),
		Amount:           int64(o.Amount),
		Price:            int64(o.Price),
		Timestamp:        int64(o.Timestamp),
		Expiration:       int64(o.Expiration),
	}
}

func (o OrderBody) Valid() (bool, error) {
	if o.AssetPair.AmountAsset == o.AssetPair.PriceAsset {
		return false, errors.New("invalid asset pair")
	}
	if o.Price == 0 {
		return false, errors.New("price should be positive")
	}
	if !validJVMLong(o.Price) {
		return false, errors.New("price is too big")
	}
	if o.Amount == 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(o.Amount) {
		return false, errors.New("amount is too big")
	}
	if o.Amount > MaxOrderAmount {
		return false, errors.New("amount is larger than maximum allowed")
	}
	if o.MatcherFee == 0 {
		return false, errors.New("matcher's fee should be positive")
	}
	if !validJVMLong(o.MatcherFee) {
		return false, errors.New("matcher's fee is too big")
	}
	if o.MatcherFee > MaxOrderAmount {
		return false, errors.New("matcher's fee is larger than maximum allowed")
	}
	if o.Timestamp == 0 {
		return false, errors.New("timestamp should be positive")
	}
	if o.Expiration == 0 {
		return false, errors.New("expiration should be positive")
	}
	return true, nil
}

func (o OrderBody) GetSenderPKBytes() []byte {
	return o.SenderPK.Bytes()
}

func (o OrderBody) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, o.SenderPK)
}

func (o *OrderBody) SpendAsset() OptionalAsset {
	if o.OrderType == Buy {
		return o.AssetPair.PriceAsset
	}
	return o.AssetPair.AmountAsset
}

func (o *OrderBody) marshalBinary() ([]byte, error) {
	var p int
	aal := 0
	if o.AssetPair.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	pal := 0
	if o.AssetPair.PriceAsset.Present {
		pal += crypto.DigestSize
	}
	buf := make([]byte, orderLen+aal+pal)
	copy(buf[0:], o.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], o.MatcherPK[:])
	p += crypto.PublicKeySize
	aa, err := o.AssetPair.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal OrderBody to bytes")
	}
	copy(buf[p:], aa)
	p += 1 + aal
	pa, err := o.AssetPair.PriceAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal OrderBody to bytes")
	}
	copy(buf[p:], pa)
	p += 1 + pal
	buf[p] = byte(o.OrderType)
	p++
	binary.BigEndian.PutUint64(buf[p:], o.Price)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Timestamp)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.Expiration)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], o.MatcherFee)
	return buf, nil
}

func (o *OrderBody) Serialize(s *serializer.Serializer) error {
	err := s.Bytes(o.SenderPK[:])
	if err != nil {
		return err
	}
	err = s.Bytes(o.MatcherPK[:])
	if err != nil {
		return err
	}
	err = o.AssetPair.AmountAsset.Serialize(s)
	if err != nil {
		return errors.Wrapf(err, "failed marshal OrderBody to bytes")
	}
	err = o.AssetPair.PriceAsset.Serialize(s)
	if err != nil {
		return errors.Wrapf(err, "failed marshal OrderBody to bytes")
	}
	err = s.Byte(byte(o.OrderType))
	if err != nil {
		return err
	}
	err = s.Uint64(o.Price)
	if err != nil {
		return err
	}
	err = s.Uint64(o.Amount)
	if err != nil {
		return err
	}
	err = s.Uint64(o.Timestamp)
	if err != nil {
		return err
	}
	err = s.Uint64(o.Expiration)
	if err != nil {
		return err
	}
	return s.Uint64(o.MatcherFee)
}

func (o *OrderBody) UnmarshalBinary(data []byte) error {
	if l := len(data); l < orderLen {
		return errors.Errorf("not enough data for OrderBody, expected not less then %d, received %d", orderLen, l)
	}
	copy(o.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(o.MatcherPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	err := o.AssetPair.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderBody from bytes")
	}
	data = data[1:]
	if o.AssetPair.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = o.AssetPair.PriceAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderBody from bytes")
	}
	data = data[1:]
	if o.AssetPair.PriceAsset.Present {
		data = data[crypto.DigestSize:]
	}
	o.OrderType = OrderType(data[0])
	if o.OrderType > 1 {
		return errors.Errorf("incorrect order type %d", o.OrderType)
	}
	data = data[1:]
	o.Price = binary.BigEndian.Uint64(data)
	data = data[8:]
	o.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	o.Timestamp = binary.BigEndian.Uint64(data)
	data = data[8:]
	o.Expiration = binary.BigEndian.Uint64(data)
	data = data[8:]
	o.MatcherFee = binary.BigEndian.Uint64(data)
	return nil
}

// OrderV1 is an order created and signed by user. Two matched orders builds up an Exchange transaction.
type OrderV1 struct {
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	OrderBody
}

func (o OrderV1) BinarySize() int {
	return crypto.SignatureSize + o.OrderBody.BinarySize()
}

func (o OrderV1) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: nil, Amount: int64(o.MatcherFee)}
	res.PriceMode = o.GetPriceMode().ToProtobuf()
	res.Version = 1
	return res
}

func (o OrderV1) ToProtobufSigned(scheme Scheme) *g.Order {
	res := o.ToProtobuf(scheme)
	proofs := NewProofsFromSignature(o.Signature)
	res.Proofs = proofs.Bytes()
	return res
}

func (o OrderV1) GetID() ([]byte, error) {
	if o.ID != nil {
		return o.ID.Bytes(), nil
	}
	return nil, errors.New("no id for OrderV1")
}

func (o OrderV1) GetProofs() (*ProofsV1, error) {
	if o.Signature == nil {
		return nil, errors.New("not signed")
	}
	proofs := NewProofsFromSignature(o.Signature)
	return proofs, nil
}

func (o OrderV1) GetAmount() uint64 {
	return o.OrderBody.Amount
}

func (o OrderV1) GetTimestamp() uint64 {
	return o.Timestamp
}

func (o OrderV1) GetMatcherFee() uint64 {
	return o.MatcherFee
}

func (o OrderV1) GetMatcherFeeAsset() OptionalAsset {
	return OptionalAsset{}
}

// NewUnsignedOrderV1 creates the new unsigned order.
func NewUnsignedOrderV1(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) *OrderV1 {
	ob := OrderBody{
		SenderPK:  senderPK,
		MatcherPK: matcherPK,
		AssetPair: AssetPair{
			AmountAsset: amountAsset,
			PriceAsset:  priceAsset},
		OrderType:  orderType,
		Price:      price,
		Amount:     amount,
		Timestamp:  timestamp,
		Expiration: expiration,
		MatcherFee: matcherFee,
	}
	return &OrderV1{OrderBody: ob}
}

func (o *OrderV1) GetVersion() byte {
	return 1
}

func (o *OrderV1) GetPriceMode() OrderPriceMode {
	return OrderPriceModeDefault
}

func (o *OrderV1) GetOrderType() OrderType {
	return o.OrderType
}

func (o *OrderV1) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o *OrderV1) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o *OrderV1) GetPrice() uint64 {
	return o.Price
}

func (o *OrderV1) GetExpiration() uint64 {
	return o.Expiration
}

func (o OrderV1) BodyMarshalBinary() ([]byte, error) {
	return o.OrderBody.marshalBinary()
}

func (o OrderV1) BodySerialize(s *serializer.Serializer) error {
	return o.OrderBody.Serialize(s)
}

func (o *OrderV1) bodyUnmarshalBinary(data []byte) error {
	return o.OrderBody.UnmarshalBinary(data)
}

func (o *OrderV1) GenerateID(_ Scheme) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return err
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV1")
	}
	o.ID = &d
	return nil
}

// Sign adds a signature to the order.
func (o *OrderV1) Sign(_ Scheme, secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV1")
	}
	s, err := crypto.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV1")
	}
	o.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV1")
	}
	o.ID = &d
	return nil
}

// Verify checks that the order's signature is valid.
func (o *OrderV1) Verify(_ Scheme) (bool, error) {
	if o.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV1")
	}
	return crypto.Verify(o.SenderPK, *o.Signature, b), nil
}

// MarshalBinary writes order to its bytes representation.
func (o *OrderV1) MarshalBinary() ([]byte, error) {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV1 to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], o.Signature[:])
	return buf, nil
}

// Serialize order to its bytes representation.
func (o *OrderV1) Serialize(s *serializer.Serializer) error {
	err := o.BodySerialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal OrderV1 to bytes")
	}
	return s.Bytes(o.Signature[:])
}

// UnmarshalBinary reads an order from its binary representation.
func (o *OrderV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < orderV1MinLen {
		return errors.Errorf("not enough data for OrderV1, expected not less then %d, received %d", orderV1MinLen, l)
	}
	var bl int
	err := o.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV1")
	}
	bl += orderLen
	if o.AssetPair.AmountAsset.Present {
		bl += crypto.DigestSize
	}
	if o.AssetPair.PriceAsset.Present {
		bl += crypto.DigestSize
	}
	b := data[:bl]
	data = data[bl:]
	var s crypto.Signature
	copy(s[:], data[:crypto.SignatureSize])
	o.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV1 from bytes")
	}
	o.ID = &d
	return nil
}

// OrderV2 is an order created and signed by user. Two matched orders builds up an Exchange transaction. Version 2 with proofs.
type OrderV2 struct {
	Version byte           `json:"version"`
	ID      *crypto.Digest `json:"id,omitempty"`
	Proofs  *ProofsV1      `json:"proofs,omitempty"`
	OrderBody
}

func (o OrderV2) BinarySize() int {
	return 1 + o.Proofs.BinarySize() + o.OrderBody.BinarySize()
}

func (o OrderV2) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: nil, Amount: int64(o.MatcherFee)}
	res.PriceMode = o.GetPriceMode().ToProtobuf()
	res.Version = 2
	return res
}

func (o OrderV2) ToProtobufSigned(scheme Scheme) *g.Order {
	res := o.ToProtobuf(scheme)
	res.Proofs = o.Proofs.Bytes()
	return res
}

func (o OrderV2) GetID() ([]byte, error) {
	if o.ID != nil {
		return o.ID.Bytes(), nil
	}
	return nil, errors.New("no id for OrderV2")
}

func (o OrderV2) GetAmount() uint64 {
	return o.Amount
}

func (o OrderV2) GetTimestamp() uint64 {
	return o.Timestamp
}

func (o OrderV2) GetMatcherFee() uint64 {
	return o.MatcherFee
}

func (o OrderV2) GetMatcherFeeAsset() OptionalAsset {
	return OptionalAsset{}
}

func (o OrderV2) GetProofs() (*ProofsV1, error) {
	return o.Proofs, nil
}

// NewUnsignedOrderV2 creates the new unsigned order.
func NewUnsignedOrderV2(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) *OrderV2 {
	ob := OrderBody{
		SenderPK:  senderPK,
		MatcherPK: matcherPK,
		AssetPair: AssetPair{
			AmountAsset: amountAsset,
			PriceAsset:  priceAsset},
		OrderType:  orderType,
		Price:      price,
		Amount:     amount,
		Timestamp:  timestamp,
		Expiration: expiration,
		MatcherFee: matcherFee,
	}
	return &OrderV2{Version: 2, OrderBody: ob}
}

func (o *OrderV2) GetVersion() byte {
	return o.Version
}

func (o *OrderV2) GetPriceMode() OrderPriceMode {
	return OrderPriceModeDefault
}

func (o *OrderV2) GetOrderType() OrderType {
	return o.OrderType
}

func (o *OrderV2) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o *OrderV2) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o *OrderV2) GetPrice() uint64 {
	return o.Price
}

func (o *OrderV2) GetExpiration() uint64 {
	return o.Expiration
}

func (o *OrderV2) GenerateID(_ Scheme) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return err
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV2")
	}
	o.ID = &d
	return nil
}

func (o OrderV2) BodyMarshalBinary() ([]byte, error) {
	aal := 0
	if o.AssetPair.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	pal := 0
	if o.AssetPair.PriceAsset.Present {
		pal += crypto.DigestSize
	}
	buf := make([]byte, orderV2FixedBodyLen+aal+pal)
	buf[0] = o.Version
	b, err := o.OrderBody.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV2 to bytes")
	}
	copy(buf[1:], b)
	return buf, nil
}

func (o *OrderV2) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < orderV2FixedBodyLen {
		return errors.Errorf("not enough data for OrderV2, expected not less then %d, received %d", orderV2FixedBodyLen, l)
	}
	o.Version = data[0]
	if o.Version != 2 {
		return errors.Errorf("unexpected version %d for OrderV2, expected 2", o.Version)
	}
	var oo OrderBody
	err := oo.UnmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2 from bytes")
	}
	o.OrderBody = oo
	return nil
}

// Sign adds a signature to the order.
func (o *OrderV2) Sign(_ Scheme, secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV2")
	}
	if o.Proofs == nil {
		o.Proofs = NewProofs()
	}
	err = o.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV2")
	}
	d, err := crypto.FastHash(b)
	o.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV2")
	}
	return nil
}

// Verify checks that the order's signature is valid.
func (o *OrderV2) Verify(_ Scheme) (bool, error) {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV2")
	}
	return o.Proofs.Verify(o.SenderPK, b)
}

// MarshalBinary writes order to its bytes representation.
func (o *OrderV2) MarshalBinary() ([]byte, error) {
	bb, err := o.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV2 to bytes")
	}
	bl := len(bb)
	pfb, err := o.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV2 to bytes")
	}
	buf := make([]byte, bl+len(pfb))
	copy(buf, bb)
	copy(buf[bl:], pfb)
	return buf, nil
}

// UnmarshalBinary reads an order from its binary representation.
func (o *OrderV2) UnmarshalBinary(data []byte) error {
	if l := len(data); l < orderV2MinLen {
		return errors.Errorf("not enough data for OrderV2, expected not less then %d, received %d", orderV2MinLen, l)
	}
	var bl int
	err := o.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2")
	}
	bl += orderV2FixedBodyLen
	if o.AssetPair.AmountAsset.Present {
		bl += crypto.DigestSize
	}
	if o.AssetPair.PriceAsset.Present {
		bl += crypto.DigestSize
	}
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2 from bytes")
	}
	o.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2 from bytes")
	}
	o.ID = &id
	return nil
}

// OrderV3 is an order that supports matcher's fee in assets.
type OrderV3 struct {
	Version         byte           `json:"version"`
	ID              *crypto.Digest `json:"id,omitempty"`
	Proofs          *ProofsV1      `json:"proofs,omitempty"`
	MatcherFeeAsset OptionalAsset  `json:"matcherFeeAssetId"`
	OrderBody
}

func (o OrderV3) BinarySize() int {
	return 1 + o.Proofs.BinarySize() + o.MatcherFeeAsset.BinarySize() + o.OrderBody.BinarySize()
}

func (o OrderV3) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: o.MatcherFeeAsset.ToID(), Amount: int64(o.MatcherFee)}
	res.PriceMode = o.GetPriceMode().ToProtobuf()
	res.Version = 3
	return res
}

func (o OrderV3) ToProtobufSigned(scheme Scheme) *g.Order {
	res := o.ToProtobuf(scheme)
	res.Proofs = o.Proofs.Bytes()
	return res
}

func (o *OrderV3) GetID() ([]byte, error) {
	if o.ID != nil {
		return o.ID.Bytes(), nil
	}
	return nil, errors.New("no id for OrderV3")
}

func (o OrderV3) GetAmount() uint64 {
	return o.Amount
}

func (o OrderV3) GetTimestamp() uint64 {
	return o.Timestamp
}

func (o OrderV3) GetMatcherFee() uint64 {
	return o.MatcherFee
}

func (o OrderV3) GetMatcherFeeAsset() OptionalAsset {
	return o.MatcherFeeAsset
}

func (o OrderV3) GetProofs() (*ProofsV1, error) {
	return o.Proofs, nil
}

// NewUnsignedOrderV3 creates the new unsigned order.
func NewUnsignedOrderV3(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64, matcherFeeAsset OptionalAsset) *OrderV3 {
	ob := OrderBody{
		SenderPK:  senderPK,
		MatcherPK: matcherPK,
		AssetPair: AssetPair{
			AmountAsset: amountAsset,
			PriceAsset:  priceAsset},
		OrderType:  orderType,
		Price:      price,
		Amount:     amount,
		Timestamp:  timestamp,
		Expiration: expiration,
		MatcherFee: matcherFee,
	}
	return &OrderV3{Version: 3, MatcherFeeAsset: matcherFeeAsset, OrderBody: ob}
}

func (o *OrderV3) GetVersion() byte {
	return o.Version
}

func (o *OrderV3) GetPriceMode() OrderPriceMode {
	return OrderPriceModeDefault
}

func (o *OrderV3) GetOrderType() OrderType {
	return o.OrderType
}

func (o *OrderV3) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o *OrderV3) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o *OrderV3) GetPrice() uint64 {
	return o.Price
}

func (o *OrderV3) GetExpiration() uint64 {
	return o.Expiration
}

func (o *OrderV3) GenerateID(_ Scheme) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return err
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV3")
	}
	o.ID = &d
	return nil
}

func (o *OrderV3) BodyMarshalBinary() ([]byte, error) {
	aal := 0
	if o.AssetPair.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	pal := 0
	if o.AssetPair.PriceAsset.Present {
		pal += crypto.DigestSize
	}
	mal := 0
	if o.MatcherFeeAsset.Present {
		mal += crypto.DigestSize
	}
	buf := make([]byte, orderV3FixedBodyLen+aal+pal+mal)
	pos := 0
	buf[pos] = o.Version
	pos++
	b, err := o.OrderBody.marshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV3 to bytes")
	}
	copy(buf[pos:], b)
	pos += len(b)
	mfa, err := o.MatcherFeeAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal OrderV3 to bytes")
	}
	copy(buf[pos:], mfa)
	return buf, nil
}

func (o *OrderV3) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < orderV3FixedBodyLen {
		return errors.Errorf("not enough data for OrderV3, expected not less then %d, received %d", orderV3FixedBodyLen, l)
	}
	pos := 0
	o.Version = data[pos]
	pos++
	if o.Version != 3 {
		return errors.Errorf("unexpected version %d for OrderV3, expected 3", o.Version)
	}
	var oo OrderBody
	err := oo.UnmarshalBinary(data[pos:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV3 from bytes")
	}
	o.OrderBody = oo
	pos += orderLen
	if oo.AssetPair.AmountAsset.Present {
		pos += crypto.DigestSize
	}
	if oo.AssetPair.PriceAsset.Present {
		pos += crypto.DigestSize
	}
	err = o.MatcherFeeAsset.UnmarshalBinary(data[pos:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV3 from bytes")
	}
	return nil
}

// Sign adds a signature to the order.
func (o *OrderV3) Sign(_ Scheme, secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV3")
	}
	if o.Proofs == nil {
		o.Proofs = NewProofs()
	}
	err = o.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV3")
	}
	d, err := crypto.FastHash(b)
	o.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV3")
	}
	return nil
}

// Verify checks that the order's signature is valid.
func (o *OrderV3) Verify(_ Scheme) (bool, error) {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV3")
	}
	return o.Proofs.Verify(o.SenderPK, b)
}

// MarshalBinary writes order to its bytes representation.
func (o *OrderV3) MarshalBinary() ([]byte, error) {
	bb, err := o.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV3 to bytes")
	}
	bl := len(bb)
	pfb, err := o.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV3 to bytes")
	}
	buf := make([]byte, bl+len(pfb))
	copy(buf, bb)
	copy(buf[bl:], pfb)
	return buf, nil
}

// UnmarshalBinary reads an order from its binary representation.
func (o *OrderV3) UnmarshalBinary(data []byte) error {
	if l := len(data); l < orderV3MinLen {
		return errors.Errorf("not enough data for OrderV3, expected not less then %d, received %d", orderV3MinLen, l)
	}
	var bl int
	err := o.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV3")
	}
	bl += orderV3FixedBodyLen
	if o.AssetPair.AmountAsset.Present {
		bl += crypto.DigestSize
	}
	if o.AssetPair.PriceAsset.Present {
		bl += crypto.DigestSize
	}
	if o.MatcherFeeAsset.Present {
		bl += crypto.DigestSize
	}
	bb := data[:bl]
	data = data[bl:]
	var p ProofsV1
	err = p.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV3 from bytes")
	}
	o.Proofs = &p
	id, err := crypto.FastHash(bb)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV3 from bytes")
	}
	o.ID = &id
	return nil
}

// OrderV4 is for Protobuf.
type OrderV4 struct {
	Version         byte           `json:"version"`
	ID              *crypto.Digest `json:"id,omitempty"`
	Proofs          *ProofsV1      `json:"proofs,omitempty"`
	MatcherFeeAsset OptionalAsset  `json:"matcherFeeAssetId"`
	PriceMode       OrderPriceMode `json:"priceMode"`
	OrderBody
}

func (o OrderV4) BinarySize() int {
	// No binary format for OrderV4, return 0.
	return 0
}

func (o OrderV4) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: o.MatcherFeeAsset.ToID(), Amount: int64(o.MatcherFee)}
	res.PriceMode = o.PriceMode.ToProtobuf()
	res.Version = 4
	return res
}

func (o OrderV4) ToProtobufSigned(scheme Scheme) *g.Order {
	res := o.ToProtobuf(scheme)
	res.Proofs = o.Proofs.Bytes()
	return res
}

func (o *OrderV4) GetID() ([]byte, error) {
	if o.ID != nil {
		return o.ID.Bytes(), nil
	}
	return nil, errors.New("no id set")
}

func (o OrderV4) GetAmount() uint64 {
	return o.Amount
}

func (o OrderV4) GetTimestamp() uint64 {
	return o.Timestamp
}

func (o OrderV4) GetMatcherFee() uint64 {
	return o.MatcherFee
}

func (o OrderV4) GetMatcherFeeAsset() OptionalAsset {
	return o.MatcherFeeAsset
}

func (o OrderV4) GetProofs() (*ProofsV1, error) {
	return o.Proofs, nil
}

// NewUnsignedOrderV4 creates the new unsigned order.
func NewUnsignedOrderV4(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64, matcherFeeAsset OptionalAsset, priceMode OrderPriceMode) *OrderV4 {
	ob := OrderBody{
		SenderPK:  senderPK,
		MatcherPK: matcherPK,
		AssetPair: AssetPair{
			AmountAsset: amountAsset,
			PriceAsset:  priceAsset},
		OrderType:  orderType,
		Price:      price,
		Amount:     amount,
		Timestamp:  timestamp,
		Expiration: expiration,
		MatcherFee: matcherFee,
	}
	return &OrderV4{Version: 4, MatcherFeeAsset: matcherFeeAsset, PriceMode: priceMode, OrderBody: ob}
}

func (o *OrderV4) GetVersion() byte {
	return o.Version
}

func (o *OrderV4) GetPriceMode() OrderPriceMode {
	return o.PriceMode
}

func (o *OrderV4) GetOrderType() OrderType {
	return o.OrderType
}

func (o *OrderV4) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o *OrderV4) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o *OrderV4) GetPrice() uint64 {
	return o.Price
}

func (o *OrderV4) GetExpiration() uint64 {
	return o.Expiration
}

func (o *OrderV4) GenerateID(scheme Scheme) error {
	b, err := o.BodyMarshalBinary(scheme)
	if err != nil {
		return err
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV4")
	}
	o.ID = &d
	return nil
}

func (o *OrderV4) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	pbOrder := o.ToProtobuf(scheme)
	pbOrder.Proofs = nil
	return pbOrder.MarshalVTStrict()
}

// Sign adds a signature to the order.
func (o *OrderV4) Sign(scheme Scheme, secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary(scheme)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV4")
	}
	if o.Proofs == nil {
		o.Proofs = NewProofs()
	}
	err = o.Proofs.Sign(secretKey, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV4")
	}
	d, err := crypto.FastHash(b)
	o.ID = &d
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV4")
	}
	return nil
}

// Verify checks that the order's signature is valid.
func (o *OrderV4) Verify(scheme Scheme) (bool, error) {
	b, err := o.BodyMarshalBinary(scheme)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV4")
	}
	return o.Proofs.Verify(o.SenderPK, b)
}

func (o *OrderV4) Valid() (bool, error) {
	if ok, err := o.OrderBody.Valid(); !ok {
		return false, err
	}
	if ok, err := o.GetPriceMode().Valid(o.GetVersion()); !ok {
		return false, err
	}
	return true, nil
}

// NewUnsignedEthereumOrderV4 creates the new ethereum unsigned order.
func NewUnsignedEthereumOrderV4(senderPK *EthereumPublicKey, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64, matcherFeeAsset OptionalAsset, priceMode OrderPriceMode) *EthereumOrderV4 {
	orderV4 := NewUnsignedOrderV4(crypto.PublicKey{}, matcherPK, amountAsset, priceAsset, orderType, price, amount, timestamp, expiration, matcherFee, matcherFeeAsset, priceMode)
	return &EthereumOrderV4{
		SenderPK:        ethereumPublicKeyBase58Wrapper{inner: senderPK},
		Eip712Signature: EthereumSignature{},
		OrderV4:         *orderV4,
	}
}

type ethereumPublicKeyBase58Wrapper struct {
	inner *EthereumPublicKey
}

func (w *ethereumPublicKeyBase58Wrapper) MarshalJSON() ([]byte, error) {
	data := w.inner.SerializeXYCoordinates()
	return B58Bytes(data).MarshalJSON()
}

func (w *ethereumPublicKeyBase58Wrapper) UnmarshalJSON(bytes []byte) error {
	pkBytes := B58Bytes{}
	err := pkBytes.UnmarshalJSON(bytes)
	if err != nil {
		return err
	}
	inner := new(EthereumPublicKey)
	if err := inner.UnmarshalBinary(pkBytes); err != nil {
		return err
	}
	w.inner = inner
	return nil
}

type EthereumOrderV4 struct {
	SenderPK        ethereumPublicKeyBase58Wrapper `json:"senderPublicKey"`
	Eip712Signature EthereumSignature              `json:"eip712Signature,omitempty"`
	OrderV4
}

func (o *EthereumOrderV4) Valid() (bool, error) {
	if len(o.Proofs.Proofs) > 0 {
		// see isValid method in com/wavesplatform/transaction/assets/exchange/Order.scala
		return false, errors.New("eip712Signature excludes proofs")
	}
	return o.OrderV4.Valid()
}

func (o *EthereumOrderV4) GetSenderPKBytes() []byte {
	// 64 bytes of uncompressed ethereum public key
	return o.SenderPK.inner.SerializeXYCoordinates()
}

func (o *EthereumOrderV4) GetSender(_ Scheme) (Address, error) {
	return o.SenderPK.inner.EthereumAddress(), nil
}

func (o *EthereumOrderV4) GenerateID(scheme Scheme) error {
	b, err := o.BodyMarshalBinary(scheme)
	if err != nil {
		return err
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV4")
	}
	o.ID = &d
	return nil
}

func (o *EthereumOrderV4) GenerateSenderPK(scheme Scheme) error {
	hash, err := o.ethereumTypedDataHash(scheme)
	if err != nil {
		return errors.Wrap(err, "failed to generate typed data hash for EthereumOrderV4.SenderPK")
	}
	pk, err := recoverEthPubKeyForEthOrderV4(scheme, hash[:], o.Eip712Signature)
	if err != nil {
		return errors.Wrap(err, "failed to recover EthereumOrderV4.SenderPK")
	}
	o.SenderPK = ethereumPublicKeyBase58Wrapper{inner: pk}
	return nil
}

func recoverEthPubKeyForEthOrderV4(scheme Scheme, digest []byte, sig EthereumSignature) (*EthereumPublicKey, error) {
	v := sig.V()
	if v <= 28 {
		return sig.RecoverEthereumPublicKey(digest)
	}
	v = v - scheme*2 - 35 // according to the https://eips.ethereum.org/EIPS/eip-155
	sig.setV(v)
	return sig.RecoverEthereumPublicKey(digest)
}

func (o *EthereumOrderV4) Verify(scheme Scheme) (bool, error) {
	hash, err := o.ethereumTypedDataHash(scheme)
	if err != nil {
		return false, errors.Wrap(err, "failed to validate ethereum signature for EthereumOrderV4")
	}
	// we don't have to validate V param here because in ethereum it's used to chainID verification (mostly),
	// but we have own scheme validation in ethereumTypedDataHash
	_, r, s := o.Eip712Signature.AsVRS()
	return VerifyEthereumSignature(o.SenderPK.inner, r, s, hash[:]), nil
}

func (o *EthereumOrderV4) Sign(_ Scheme, _ crypto.SecretKey) error {
	return errors.Errorf("(%T) doesn't support Sign method", o)
}

func (o *EthereumOrderV4) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderV4.ToProtobuf(scheme)
	res.Sender = &g.Order_Eip712Signature{Eip712Signature: o.Eip712Signature.Bytes()}
	return res
}

func (o *EthereumOrderV4) ToProtobufSigned(scheme Scheme) *g.Order {
	res := o.ToProtobuf(scheme)
	return res
}

func (o *EthereumOrderV4) BodyMarshalBinary(scheme Scheme) ([]byte, error) {
	pbOrder := o.ToProtobuf(scheme)
	return MarshalToProtobufDeterministic(pbOrder)
}

// EthereumSign signs order and sets senderPK with provided *EthereumPrivateKey. This method is used only for test purposes
func (o *EthereumOrderV4) EthereumSign(scheme Scheme, sk *EthereumPrivateKey) (err error) {
	h, err := o.ethereumTypedDataHash(scheme)
	if err != nil {
		return errors.Wrap(err, "failed to sign ethereum order")
	}
	eip712SignatureBytes, err := crypto.ECDSASign(h[:], (*btcec.PrivateKey)(sk))
	if err != nil {
		return errors.Wrap(err, "failed to sign EthereumOrderV4 with 'ethereumSecretKey'")
	}
	eip712SignatureBytes[len(eip712SignatureBytes)-1] += 27 // Transform V signature value from 0/1 to 27/28 according to the yellow paper
	eip712Signature, err := NewEthereumSignatureFromBytes(eip712SignatureBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to convert '%x' bytes to EthereumSignature", eip712SignatureBytes)
	}
	o.Eip712Signature = eip712Signature
	o.SenderPK = ethereumPublicKeyBase58Wrapper{inner: sk.EthereumPublicKey()}
	err = o.GenerateID(scheme)
	if err != nil {
		return errors.Wrap(err, "failed generate ID for EthereumOrderV4")
	}
	return nil
}

func (o *EthereumOrderV4) ethereumTypedDataHash(scheme Scheme) (EthereumHash, error) {
	typedData := o.buildEthereumOrderV4TypedData(scheme)
	hash, err := typedData.Hash()
	if err != nil {
		return EthereumHash{}, errors.Wrap(err, "failed calculate ethereum typed data hash for EthereumOrderV4")
	}
	return hash, nil
}

func (o *EthereumOrderV4) buildEthereumOrderV4TypedData(scheme Scheme) ethereumTypedData {
	priceMode := o.PriceMode
	if priceMode == OrderPriceModeDefault {
		priceMode = OrderPriceModeFixedDecimals
	}
	message := ethereumTypedDataMessage{
		"version":           int32(o.Version),
		"matcherPublicKey":  o.MatcherPK.String(),
		"amountAsset":       o.AssetPair.AmountAsset.String(),
		"priceAsset":        o.AssetPair.PriceAsset.String(),
		"orderType":         strings.ToUpper(o.OrderType.String()),
		"amount":            int64(o.Amount),
		"price":             int64(o.Price),
		"timestamp":         int64(o.Timestamp),
		"expiration":        int64(o.Expiration),
		"matcherFee":        int64(o.MatcherFee),
		"matcherFeeAssetId": o.MatcherFeeAsset.String(),
		"priceMode":         priceMode.upperSnakeCaseString(),
	}

	var orderDomain = ethereumTypedData{
		Types: ethereumTypedDataTypes{
			"EIP712Domain": []ethereumTypedDataType{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"Order": []ethereumTypedDataType{
				{Name: "version", Type: "int32"},
				{Name: "matcherPublicKey", Type: "string"},
				{Name: "amountAsset", Type: "string"},
				{Name: "priceAsset", Type: "string"},
				{Name: "orderType", Type: "string"},
				{Name: "amount", Type: "int64"},
				{Name: "price", Type: "int64"},
				{Name: "timestamp", Type: "int64"},
				{Name: "expiration", Type: "int64"},
				{Name: "matcherFee", Type: "int64"},
				{Name: "matcherFeeAssetId", Type: "string"},
				{Name: "priceMode", Type: "string"},
			},
		},
		PrimaryType: "Order",
		Domain: ethereumTypedDataDomain{
			Name:    "Waves Order",
			Version: "1",
			ChainId: newHexOrDecimal256(int64(scheme)),
		},
		Message: message,
	}
	return orderDomain
}

const (
	proofsVersion  byte = 1
	proofsMinLen        = 1 + 2
	proofsMaxCount      = 8
	proofMaxSize        = 64
)

// ProofsV1 is a collection of proofs.
type ProofsV1 struct {
	Version byte
	Proofs  []B58Bytes
}

func NewProofs() *ProofsV1 {
	return &ProofsV1{Version: proofsVersion, Proofs: make([]B58Bytes, 0)}
}

func NewProofsFromSignature(sig *crypto.Signature) *ProofsV1 {
	return &ProofsV1{proofsVersion, []B58Bytes{sig.Bytes()}}
}

func (p ProofsV1) Bytes() [][]byte {
	res := make([][]byte, len(p.Proofs))
	for i, proof := range p.Proofs {
		res[i] = make([]byte, len(proof))
		copy(res[i], proof)
	}
	return res
}

// String gives a string representation of the proofs collection.
func (p ProofsV1) String() string {
	var sb strings.Builder
	sb.WriteRune('[')
	for i, e := range p.Proofs {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(e.String())
	}
	sb.WriteRune(']')
	return sb.String()
}

// MarshalJSON writes the proofs to JSON.
func (p ProofsV1) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Proofs)
}

// UnmarshalJSON reads the proofs from JSON.
func (p *ProofsV1) UnmarshalJSON(value []byte) error {
	var tmp []B58Bytes
	err := json.Unmarshal(value, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProofsV1 from JSON")
	}
	p.Version = proofsVersion
	p.Proofs = tmp
	if err := p.Valid(); err != nil {
		return errors.Wrap(err, "failed to unmarshal ProofsV1 from JSON")
	}
	return nil
}

// MarshalBinary writes the proofs to its binary form.
func (p *ProofsV1) MarshalBinary() ([]byte, error) {
	buf := make([]byte, p.BinarySize())
	pos := 0
	buf[pos] = proofsVersion
	pos++
	binary.BigEndian.PutUint16(buf[pos:], uint16(len(p.Proofs)))
	pos += 2
	for _, e := range p.Proofs {
		el := len(e)
		binary.BigEndian.PutUint16(buf[pos:], uint16(el))
		pos += 2
		copy(buf[pos:], e)
		pos += el
	}
	return buf, nil
}

// Serialize proofs to its binary form.
func (p *ProofsV1) Serialize(s *serializer.Serializer) error {
	err := s.Byte(proofsVersion)
	if err != nil {
		return err
	}
	err = s.Uint16(uint16(len(p.Proofs)))
	if err != nil {
		return err
	}
	for _, e := range p.Proofs {
		el := len(e)
		err = s.Uint16(uint16(el))
		if err != nil {
			return err
		}
		err = s.Bytes(e)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalBinary reads the proofs from its binary representation.
func (p *ProofsV1) UnmarshalBinary(data []byte) error {
	if l := len(data); l < proofsMinLen {
		return errors.Errorf("not enough data for ProofsV1 value, expected %d, received %d", proofsMinLen, l)
	}
	p.Version = data[0]
	if p.Version != proofsVersion {
		return errors.Errorf("unexpected ProofsV1 version %d, expected %d", p.Version, proofsVersion)
	}
	data = data[1:]
	n := int(binary.BigEndian.Uint16(data))
	if n > proofsMaxCount {
		return errors.Errorf("too many proofs in ProofsV1, expected no more than %d, received %d", proofsMaxCount, n)
	}
	data = data[2:]
	for i := 0; i < n; i++ {
		el := binary.BigEndian.Uint16(data)
		if el > proofMaxSize {
			return errors.Errorf("proof size %d bytes exceeds maximum allowed %d", el, proofMaxSize)
		}
		data = data[2:]
		pr := make([]byte, el)
		copy(pr, data[0:el])
		data = data[el:]
		p.Proofs = append(p.Proofs, pr)
	}
	return nil
}

// Sign creates a signature and stores it as a proof at first position.
func (p *ProofsV1) Sign(key crypto.SecretKey, data []byte) error {
	if len(p.Proofs) == 0 {
		s, err := crypto.Sign(key, data)
		if err != nil {
			return errors.Errorf("crypto.Sign(): %v", err)
		}
		p.Proofs = []B58Bytes{s[:]}
	} else {
		if len(p.Proofs[0]) != 0 {
			return errors.New("unable to overwrite non-empty proof at position 0")
		}
		s, err := crypto.Sign(key, data)
		if err != nil {
			return errors.Errorf("crypto.Sign(): %v", err)
		}
		p.Proofs[0] = s[:]
	}
	return nil
}

// Verify checks that the proof at first position is a valid signature.
func (p *ProofsV1) Verify(key crypto.PublicKey, data []byte) (bool, error) {
	sig, err := p.ExtractSignature()
	if err != nil {
		return false, errors.Wrap(err, "failed to extract signature from proofs")
	}
	return crypto.Verify(key, sig, data), nil
}

func (p *ProofsV1) BinarySize() int {
	pl := 0
	if p != nil {
		for _, e := range p.Proofs {
			pl += len(e) + 2
		}
	}
	return proofsMinLen + pl
}

func (p *ProofsV1) Valid() error {
	if v := p.Version; v != proofsVersion {
		return errors.Errorf("invalid proofs version %d", v)
	}
	if c := len(p.Proofs); c > proofsMaxCount {
		return errors.Errorf("invalid proofs count %d", c)
	}
	for i, proof := range p.Proofs {
		if s := len(proof); s > proofMaxSize {
			return errors.Errorf("proof #%d has invalid size %d", i, s)
		}
	}
	return nil
}

// IsSimpleSigned performs basics checks of signature correctness.
func (p *ProofsV1) IsSimpleSigned() bool {
	return len(p.Proofs) == 1 && len(p.Proofs[0]) == crypto.SignatureSize
}

// ExtractSignature tries to extract a signature from ProofsV1.Proofs slice.
func (p *ProofsV1) ExtractSignature() (crypto.Signature, error) {
	if !p.IsSimpleSigned() {
		return crypto.Signature{}, errors.Errorf("proofs are not a signature")
	}
	return crypto.NewSignatureFromBytes(p.Proofs[0])
}

func (p *ProofsV1) Len() int {
	return len(p.Proofs)
}

// DataValueType is an alias for byte that encodes the value type.
type DataValueType byte

// String translates ValueType value to human readable name.
func (vt DataValueType) String() string {
	switch vt {
	case DataInteger:
		return "integer"
	case DataBoolean:
		return "boolean"
	case DataBinary:
		return "binary"
	case DataString:
		return "string"
	case DataDelete:
		return "delete"
	default:
		return ""
	}
}

// Supported value types.
const (
	DataInteger DataValueType = iota
	DataBoolean
	DataBinary
	DataString
	DataDelete = DataValueType(0xff)
)

// DataEntry is a common interface of all types of data entries.
// The interface is used to store different types of data entries in one slice.
type DataEntry interface {
	GetKey() string
	SetKey(string)

	GetValueType() DataValueType
	MarshalValue() ([]byte, error)
	UnmarshalValue([]byte) error

	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	Valid(forbidEmptyKey, utf16KeyLen bool) error
	BinarySize() int
	PayloadSize() int

	ToProtobuf() *g.DataTransactionData_DataEntry
}

var bytesToDataEntry = map[DataValueType]reflect.Type{
	DataInteger: reflect.TypeOf(IntegerDataEntry{}),
	DataBoolean: reflect.TypeOf(BooleanDataEntry{}),
	DataString:  reflect.TypeOf(StringDataEntry{}),
	DataBinary:  reflect.TypeOf(BinaryDataEntry{}),
	DataDelete:  reflect.TypeOf(DeleteDataEntry{}),
}

func NewDataEntryFromValueBytes(valueBytes []byte) (DataEntry, error) {
	if len(valueBytes) < 1 {
		return nil, errors.New("invalid data size")
	}
	entryType, ok := bytesToDataEntry[DataValueType(valueBytes[0])]
	if !ok {
		return nil, errors.New("invalid data entry type")
	}
	entry, ok := reflect.New(entryType).Interface().(DataEntry)
	if !ok {
		panic("This entry type does not implement DataEntry interface")
	}
	if err := entry.UnmarshalValue(valueBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal entry")
	}
	return entry, nil
}

// IntegerDataEntry stores int64 value.
type IntegerDataEntry struct {
	Key   string
	Value int64
}

func (e IntegerDataEntry) ToProtobuf() *g.DataTransactionData_DataEntry {
	return &g.DataTransactionData_DataEntry{
		Key:   e.Key,
		Value: &g.DataTransactionData_DataEntry_IntValue{IntValue: e.Value},
	}
}

func (e IntegerDataEntry) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	if forbidEmptyKey && len(e.Key) == 0 {
		return errs.NewEmptyDataKey("empty entry key")
	}
	if utf16KeyLen {
		if len(utf16.Encode([]rune(e.Key))) > MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	} else {
		if len(e.Key) > MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}
	return nil
}

// GetKey returns the key of data entry.
func (e IntegerDataEntry) GetKey() string {
	return e.Key
}

// SetKey sets the key of data entry.
func (e *IntegerDataEntry) SetKey(key string) {
	e.Key = key
}

// GetValueType returns the value type of the entry.
func (e IntegerDataEntry) GetValueType() DataValueType {
	return DataInteger
}

func (e IntegerDataEntry) BinarySize() int {
	return 2 + len(e.Key) + 1 + 8
}

func (e IntegerDataEntry) PayloadSize() int {
	return len(e.Key) + 8 // 8 == sizeof(int64)
}

// MarshalValue marshals the integer data entry value in its bytes representation.
func (e IntegerDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+8)
	pos := 0
	buf[pos] = byte(DataInteger)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(e.Value))
	return buf, nil
}

// UnmarshalValue reads binary representation of integer data entry value to the structure.
func (e *IntegerDataEntry) UnmarshalValue(data []byte) error {
	const minLen = 1 + 8
	if l := len(data); l < minLen {
		return errors.Errorf("invalid length for IntegerDataEntry value, expected not less than %d, received %d", minLen, l)
	}
	if t := data[0]; t != byte(DataInteger) {
		return errors.Errorf("unexpected value type %d for IntegerDataEntry value, expected %d", t, DataInteger)
	}
	e.Value = int64(binary.BigEndian.Uint64(data[1:]))
	return nil
}

// MarshalBinary marshals the integer data entry in its bytes representation.
func (e IntegerDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	valueBytes, err := e.MarshalValue()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], valueBytes)
	return buf, nil
}

// UnmarshalBinary reads binary representation of integer data entry to the structure.
func (e *IntegerDataEntry) UnmarshalBinary(data []byte) error {
	const minLen = 2 + 1 + 8
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for IntegerDataEntry, expected not less than %d, received %d", minLen, l)
	}
	k, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal IntegerDataEntry from bytes")
	}
	e.Key = k
	kl := 2 + len(k)
	if err := e.UnmarshalValue(data[kl:]); err != nil {
		return err
	}
	return nil
}

// MarshalJSON writes a JSON representation of integer data entry.
func (e IntegerDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V int    `json:"value"`
	}{e.Key, e.GetValueType().String(), int(e.Value)})
}

// UnmarshalJSON reads an integer data entry from its JSON representation.
func (e *IntegerDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V int    `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize integer data entry from JSON")
	}
	e.Key = tmp.K
	e.Value = int64(tmp.V)
	return nil
}

// BooleanDataEntry represents a key-value pair that stores a bool value.
type BooleanDataEntry struct {
	Key   string
	Value bool
}

func (e BooleanDataEntry) ToProtobuf() *g.DataTransactionData_DataEntry {
	return &g.DataTransactionData_DataEntry{
		Key:   e.Key,
		Value: &g.DataTransactionData_DataEntry_BoolValue{BoolValue: e.Value},
	}
}

func (e BooleanDataEntry) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	if forbidEmptyKey && len(e.Key) == 0 {
		return errs.NewEmptyDataKey("empty entry key")
	}
	if utf16KeyLen {
		if len(utf16.Encode([]rune(e.Key))) > MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	} else {
		if len(e.Key) > MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}
	return nil
}

// GetKey returns the key of data entry.
func (e BooleanDataEntry) GetKey() string {
	return e.Key
}

// SetKey sets the key of data entry.
func (e *BooleanDataEntry) SetKey(key string) {
	e.Key = key
}

// GetValueType returns the data type (Boolean) of the entry.
func (e BooleanDataEntry) GetValueType() DataValueType {
	return DataBoolean
}

func (e BooleanDataEntry) BinarySize() int {
	return 2 + len(e.Key) + 1 + 1
}

func (e BooleanDataEntry) PayloadSize() int {
	return len(e.Key) + 1 // 1 == sizeof(bool)
}

// MarshalValue writes a byte representation of the boolean data entry value.
func (e BooleanDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+1)
	pos := 0
	buf[pos] = byte(DataBoolean)
	pos++
	PutBool(buf[pos:], e.Value)
	return buf, nil
}

// UnmarshalValue reads a byte representation of the data entry value.
func (e *BooleanDataEntry) UnmarshalValue(data []byte) error {
	const minLen = 1 + 1
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for BooleanDataEntry value, expected not less than %d, received %d", minLen, l)
	}
	if t := data[0]; t != byte(DataBoolean) {
		return errors.Errorf("unexpected value type %d for BooleanDataEntry, value expected %d", t, DataBoolean)
	}
	v, err := Bool(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BooleanDataEntry value from bytes")
	}
	e.Value = v
	return nil
}

// MarshalBinary writes a byte representation of the boolean data entry.
func (e BooleanDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	valueBytes, err := e.MarshalValue()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], valueBytes)
	return buf, nil
}

// UnmarshalBinary reads a byte representation of the data entry.
func (e *BooleanDataEntry) UnmarshalBinary(data []byte) error {
	const minLen = 2 + 1 + 1
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for BooleanDataEntry, expected not less than %d, received %d", minLen, l)
	}
	k, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BooleanDataEntry from bytes")
	}
	e.Key = k
	kl := 2 + len(k)
	if err := e.UnmarshalValue(data[kl:]); err != nil {
		return err
	}
	return nil
}

// MarshalJSON writes the data entry to a JSON representation.
func (e BooleanDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V bool   `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

// UnmarshalJSON reads the entry from its JSON representation.
func (e *BooleanDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V bool   `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize boolean data entry from JSON")
	}
	e.Key = tmp.K
	e.Value = tmp.V
	return nil
}

// BinaryDataEntry represents a key-value data entry that stores binary value.
type BinaryDataEntry struct {
	Key   string
	Value []byte
}

func (e BinaryDataEntry) ToProtobuf() *g.DataTransactionData_DataEntry {
	return &g.DataTransactionData_DataEntry{
		Key:   e.Key,
		Value: &g.DataTransactionData_DataEntry_BinaryValue{BinaryValue: e.Value},
	}
}

func (e BinaryDataEntry) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	if forbidEmptyKey && len(e.Key) == 0 {
		return errs.NewEmptyDataKey("empty entry key")
	}
	if utf16KeyLen {
		if len(utf16.Encode([]rune(e.Key))) > MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	} else {
		if len(e.Key) > MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}
	if len(e.Value) > maxDataEntryValueSize {
		return errs.NewTooBigArray("value is too large")
	}
	return nil
}

// GetKey returns the key of data entry.
func (e BinaryDataEntry) GetKey() string {
	return e.Key
}

// SetKey sets the key of data entry.
func (e *BinaryDataEntry) SetKey(key string) {
	e.Key = key
}

// GetValueType returns the type of value (Binary) stored in an entry.
func (e BinaryDataEntry) GetValueType() DataValueType {
	return DataBinary
}

func (e BinaryDataEntry) BinarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

func (e BinaryDataEntry) PayloadSize() int {
	return len(e.Key) + len(e.Value)
}

// MarshalValue writes an entry value to its byte representation.
func (e BinaryDataEntry) MarshalValue() ([]byte, error) {
	pos := 0
	buf := make([]byte, 1+2+len(e.Value))
	buf[pos] = byte(DataBinary)
	pos++
	if err := PutBytesWithUInt16Len(buf[pos:], e.Value); err != nil {
		return nil, errors.Wrap(err, "failed to marshal BinaryDataEntry value")
	}
	return buf, nil
}

// UnmarshalValue reads an entry value from a binary representation.
func (e *BinaryDataEntry) UnmarshalValue(data []byte) error {
	const minLen = 1 + 2
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for BinaryDataEntry value, expected not less than %d, received %d", minLen, l)
	}
	if t := data[0]; t != byte(DataBinary) {
		return errors.Errorf("unexpected value type %d for BinaryDataEntry value, expected %d", t, DataBinary)
	}
	v, err := BytesWithUInt16Len(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BinaryDataEntry value from bytes")
	}
	e.Value = v
	return nil
}

// MarshalBinary writes an entry to its byte representation.
func (e BinaryDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	valueBytes, err := e.MarshalValue()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], valueBytes)
	return buf, nil
}

// UnmarshalBinary reads an entry from a binary representation.
func (e *BinaryDataEntry) UnmarshalBinary(data []byte) error {
	const minLen = 2 + 1 + 2
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for BinaryDataEntry, expected not less than %d, received %d", minLen, l)
	}
	k, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BinaryDataEntry from bytes")
	}
	e.Key = k
	kl := 2 + len(k)
	if err := e.UnmarshalValue(data[kl:]); err != nil {
		return err
	}
	return nil
}

// MarshalJSON converts an entry to its JSON representation. Note that BASE64 is used to represent the binary value.
func (e BinaryDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V Script `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

// UnmarshalJSON converts JSON to a BinaryDataEntry structure. Value should be stored as BASE64 sting in JSON.
func (e *BinaryDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V Script `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize binary data entry from JSON")
	}
	e.Key = tmp.K
	e.Value = tmp.V
	return nil
}

// StringDataEntry structure is a key-value pair to store a string value.
type StringDataEntry struct {
	Key   string
	Value string
}

func (e StringDataEntry) ToProtobuf() *g.DataTransactionData_DataEntry {
	return &g.DataTransactionData_DataEntry{
		Key:   e.Key,
		Value: &g.DataTransactionData_DataEntry_StringValue{StringValue: e.Value},
	}
}

func (e StringDataEntry) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	if forbidEmptyKey && len(e.Key) == 0 {
		return errs.NewEmptyDataKey("empty entry key")
	}
	if utf16KeyLen {
		if len(utf16.Encode([]rune(e.Key))) > MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	} else {
		if len(e.Key) > MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}
	if len(e.Value) > maxDataEntryValueSize {
		return errs.NewTooBigArray("value is too large")
	}
	return nil
}

// GetKey returns the key of key-value pair.
func (e StringDataEntry) GetKey() string {
	return e.Key
}

// SetKey sets the key of data entry.
func (e *StringDataEntry) SetKey(key string) {
	e.Key = key
}

// GetValueType returns the type of value in key-value entry.
func (e StringDataEntry) GetValueType() DataValueType {
	return DataString
}

func (e StringDataEntry) BinarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

func (e StringDataEntry) PayloadSize() int {
	return len(e.Key) + len(e.Value)
}

// MarshalValue converts the data entry value to its byte representation.
func (e StringDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+2+len(e.Value))
	pos := 0
	buf[pos] = byte(DataString)
	pos++
	PutStringWithUInt16Len(buf[pos:], e.Value)
	return buf, nil
}

// UnmarshalValue reads StringDataEntry value from bytes.
func (e *StringDataEntry) UnmarshalValue(data []byte) error {
	const minLen = 1 + 2
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for StringDataEntry value, expected not less than %d, received %d", minLen, l)
	}
	if t := data[0]; t != byte(DataString) {
		return errors.Errorf("unexpected value type %d for StringDataEntry value, expected %d", t, DataString)
	}
	v, err := StringWithUInt16Len(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StringDataEntry value from bytes")
	}
	e.Value = v
	return nil
}

// MarshalBinary converts the data entry to its byte representation.
func (e StringDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	valueBytes, err := e.MarshalValue()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], valueBytes)
	return buf, nil
}

// UnmarshalBinary reads StringDataEntry structure from bytes.
func (e *StringDataEntry) UnmarshalBinary(data []byte) error {
	const minLen = 2 + 1 + 2
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for StringDataEntry, expected not less than %d, received %d", minLen, l)
	}
	k, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StringDataEntry from bytes")
	}
	e.Key = k
	kl := 2 + len(k)
	if err := e.UnmarshalValue(data[kl:]); err != nil {
		return err
	}
	return nil
}

// MarshalJSON writes the entry to its JSON representation.
func (e StringDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V string `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

// UnmarshalJSON reads the entry from JSON.
func (e *StringDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V string `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize string data entry from JSON")
	}
	e.Key = tmp.K
	e.Value = tmp.V
	return nil
}

// DeleteDataEntry structure stores the key that should be removed from state storage.
type DeleteDataEntry struct {
	Key string
}

func (e DeleteDataEntry) ToProtobuf() *g.DataTransactionData_DataEntry {
	return &g.DataTransactionData_DataEntry{
		Key:   e.Key,
		Value: nil,
	}
}

func (e DeleteDataEntry) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	if forbidEmptyKey && len(e.Key) == 0 {
		return errs.NewEmptyDataKey("empty entry key")
	}
	if utf16KeyLen {
		if len(utf16.Encode([]rune(e.Key))) > MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	} else {
		if len(e.Key) > MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}
	return nil
}

// GetKey returns the key of key-value pair.
func (e DeleteDataEntry) GetKey() string {
	return e.Key
}

// SetKey sets the key of data entry.
func (e *DeleteDataEntry) SetKey(key string) {
	e.Key = key
}

// GetValueType returns the type of value in key-value entry.
func (e DeleteDataEntry) GetValueType() DataValueType {
	return DataDelete
}

func (e DeleteDataEntry) BinarySize() int {
	return 2 + len(e.Key) + 1
}

func (e DeleteDataEntry) PayloadSize() int {
	return 0 // this entry doesn't have any payload
}

// MarshalValue converts the data entry value to its byte representation.
func (e DeleteDataEntry) MarshalValue() ([]byte, error) {
	return []byte{byte(DataDelete)}, nil
}

// UnmarshalValue checks DeleteDataEntry value type is set.
func (e *DeleteDataEntry) UnmarshalValue(data []byte) error {
	const minLen = 1
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for DeleteDataEntry value, expected not less than %d, received %d", minLen, l)
	}
	if t := data[0]; t != byte(DataDelete) {
		return errors.Errorf("unexpected value type %d for DeleteDataEntry value, expected %d", t, DataDelete)
	}
	return nil
}

// MarshalBinary converts the data entry to its byte representation.
func (e DeleteDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.BinarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	valueBytes, err := e.MarshalValue()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], valueBytes)
	return buf, nil
}

// UnmarshalBinary reads StringDataEntry structure from bytes.
func (e *DeleteDataEntry) UnmarshalBinary(data []byte) error {
	const minLen = 2 + 1
	if l := len(data); l < minLen {
		return errors.Errorf("invalid data length for DeleteDataEntry, expected not less than %d, received %d", minLen, l)
	}
	k, err := StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal DeleteDataEntry from bytes")
	}
	e.Key = k
	kl := 2 + len(k)
	if err := e.UnmarshalValue(data[kl:]); err != nil {
		return err
	}
	return nil
}

// MarshalJSON writes the entry to its JSON representation.
func (e DeleteDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string  `json:"key"`
		V *string `json:"value"`
	}{e.Key, nil})
}

// UnmarshalJSON reads the entry from JSON.
func (e *DeleteDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V string `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize string data entry from JSON")
	}
	e.Key = tmp.K
	return nil
}

// dataEntryType is the assistive structure used to get the type of DataEntry while unmarshal form JSON.
type dataEntryType struct {
	Type string `json:"type"`
}

func guessDataEntryType(dataEntryType dataEntryType) (DataEntry, error) {
	var r DataEntry
	switch dataEntryType.Type {
	case "integer":
		r = &IntegerDataEntry{}
	case "boolean":
		r = &BooleanDataEntry{}
	case "binary":
		r = &BinaryDataEntry{}
	case "string":
		r = &StringDataEntry{}
	case "":
		r = &DeleteDataEntry{}
	}
	if r == nil {
		return nil, errors.Errorf("unknown value type '%s' of DataEntry", dataEntryType.Type)
	}
	return r, nil
}

func NewDataEntryFromJSON(data []byte) (DataEntry, error) {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to unmarshal DataEntry from JSON") }

	var et dataEntryType
	if err := json.Unmarshal(data, &et); err != nil {
		return nil, wrapError(err)
	}
	entry, err := guessDataEntryType(et)
	if err != nil {
		return nil, wrapError(err)
	}
	if err := json.Unmarshal(data, entry); err != nil {
		return nil, wrapError(err)
	}
	return entry, nil
}

// DataEntries the slice of various entries of DataTransaction
type DataEntries []DataEntry

// PayloadSize returns summary payload size of all entries.
func (e DataEntries) PayloadSize() int {
	pl := 0
	for i := range e {
		pl += e[i].PayloadSize()
	}
	return pl
}

// BinarySize returns summary binary size of all entries.
func (e DataEntries) BinarySize() int {
	bs := 0
	for i := range e {
		bs += e[i].BinarySize()
	}
	return bs
}

// Valid calls DataEntry.Valid for each entry.
func (e DataEntries) Valid(forbidEmptyKey, utf16KeyLen bool) error {
	for i := range e {
		if err := e[i].Valid(forbidEmptyKey, utf16KeyLen); err != nil {
			return errors.Wrapf(err, "invalid entry %d", i)
		}
	}
	return nil
}

// UnmarshalJSON special method to unmarshal DataEntries from JSON with detection of real type of each entry.
func (e *DataEntries) UnmarshalJSON(data []byte) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to unmarshal DataEntries from JSON") }

	var ets []dataEntryType
	err := json.Unmarshal(data, &ets)
	if err != nil {
		return wrapError(err)
	}

	entries := make([]DataEntry, len(ets))
	for i, row := range ets {
		et, err := guessDataEntryType(row)
		if err != nil {
			return wrapError(err)
		}
		entries[i] = et
	}

	err = json.Unmarshal(data, &entries)
	if err != nil {
		return wrapError(err)
	}
	*e = entries
	return nil
}

const scriptPrefix = "base64:"

var scriptPrefixBytes = []byte(scriptPrefix)

type ScriptInfo struct {
	Version    int32
	Bytes      []byte
	Base64     string
	Complexity uint64
}

func (s *ScriptInfo) ToProtobuf() *pb.ScriptData {
	if s == nil {
		return &pb.ScriptData{}
	}
	return &pb.ScriptData{
		ScriptBytes: s.Bytes,
		ScriptText:  s.Base64,
		Complexity:  int64(s.Complexity),
	}
}

func VersionFromScriptBytes(scriptBytes []byte) (int32, error) {
	if len(scriptBytes) == 0 {
		// No script has 0 version.
		return 0, nil
	}
	version := int32(scriptBytes[0])
	if version == 0 {
		if len(scriptBytes) < 3 {
			return 0, errors.New("invalid data size")
		}
		version = int32(scriptBytes[2])
	}
	return version, nil
}

type ScriptBasicInfo struct {
	PK             crypto.PublicKey
	ScriptLen      uint32
	LibraryVersion ast.LibraryVersion
	HasVerifier    bool
	IsDApp         bool
}

type Script []byte

// IsEmpty checks that script bytes slice is nil or slice length equals zero
func (s Script) IsEmpty() bool {
	return len(s) == 0
}

// String gives a string representation of Script bytes, script bytes encoded as BASE64 with prefix
func (s Script) String() string {
	sb := strings.Builder{}
	sb.WriteString(scriptPrefix)
	sb.WriteString(base64.StdEncoding.EncodeToString(s))
	return sb.String()
}

// MarshalJSON writes Script as JSON
func (s Script) MarshalJSON() ([]byte, error) {
	if s == nil {
		return jsonNullBytes, nil
	}
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads Script from it's JSON representation
func (s *Script) UnmarshalJSON(value []byte) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to unmarshal Script from JSON") }
	if bytes.Equal(value, jsonNullBytes) {
		return nil
	}
	if len(value) < len(scriptPrefixBytes)+2 { // +2 for quotes
		return wrapError(errors.New("insufficient length"))
	}
	if value[0] != '"' || value[len(value)-1] != '"' {
		return wrapError(errors.New("no quotes"))
	}
	value = value[1 : len(value)-1]
	if !bytes.Equal(value[0:len(scriptPrefixBytes)], scriptPrefixBytes) {
		return wrapError(errors.New("no prefix"))
	}
	value = value[len(scriptPrefixBytes):]
	sb := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
	n, err := base64.StdEncoding.Decode(sb, value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Script form JSON")
	}
	*s = sb[:n]
	return nil
}

// ArgumentValueType is an alias for byte that encodes the value type.
type ArgumentValueType byte

// String translates ValueType value to human readable name.
func (vt ArgumentValueType) String() string {
	switch vt {
	case ArgumentInteger:
		return "integer"
	case ArgumentBoolean:
		return "boolean"
	case ArgumentBinary:
		return "binary"
	case ArgumentString:
		return "string"
	case ArgumentList:
		return "list"
	default:
		return ""
	}
}

const (
	ArgumentInteger    = ArgumentValueType(0)  // E_LONG
	ArgumentBinary     = ArgumentValueType(1)  // E_BYTES
	ArgumentString     = ArgumentValueType(2)  // E_STRING
	ArgumentBoolean    = ArgumentValueType(99) // Nonexistent RIDE type is used
	ArgumentValueTrue  = ArgumentValueType(6)  // E_TRUE
	ArgumentValueFalse = ArgumentValueType(7)  // E_FALSE
	ArgumentList       = ArgumentValueType(11) // E_LIST
)

type Argument interface {
	GetValueType() ArgumentValueType
	MarshalBinary() ([]byte, error)
	BinarySize() int
	Serialize(*serializer.Serializer) error
}

// ArgumentType is the assistive structure used to get the type of DataEntry while unmarshal form JSON.
type ArgumentType struct {
	Type string `json:"type"`
}

func guessArgumentType(argumentType ArgumentType) (Argument, error) {
	var r Argument
	switch strings.ToLower(argumentType.Type) {
	case "integer", "int":
		r = &IntegerArgument{}
	case "boolean":
		r = &BooleanArgument{}
	case "binary", "bytevector":
		r = &BinaryArgument{}
	case "string":
		r = &StringArgument{}
	case "list", "array":
		r = &ListArgument{}
	}
	if r == nil {
		return nil, errors.Errorf("unknown value type '%s' of Argument", argumentType.Type)
	}
	return r, nil
}

type Arguments []Argument

// Append adds an argument to the Arguments list.
func (a *Arguments) Append(arg Argument) {
	*a = append(*a, arg)
}

// UnmarshalJSON custom JSON deserialization method.
func (a *Arguments) UnmarshalJSON(data []byte) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to unmarshal Arguments from JSON") }

	var ats []ArgumentType
	err := json.Unmarshal(data, &ats)
	if err != nil {
		return wrapError(err)
	}

	arguments := make([]Argument, len(ats))
	for i, row := range ats {
		arg, err := guessArgumentType(row)
		if err != nil {
			return wrapError(err)
		}
		arguments[i] = arg
	}

	err = json.Unmarshal(data, &arguments)
	if err != nil {
		return wrapError(err)
	}
	*a = arguments
	return nil
}

func (a Arguments) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	p := 0
	binary.BigEndian.PutUint32(buf, uint32(len(a)))
	p += 4
	for _, arg := range a {
		b, err := arg.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal Arguments to bytes")
		}
		copy(buf[p:], b)
		p += len(b)
	}
	return buf, nil
}

func (a Arguments) Serialize(s *serializer.Serializer) error {
	err := s.Uint32(uint32(len(a)))
	if err != nil {
		return err
	}
	for _, arg := range a {
		err := arg.Serialize(s)
		if err != nil {
			return errors.Wrap(err, "failed to marshal Arguments to bytes")
		}
	}
	return nil
}

func (a *Arguments) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 4 {
		return errors.Errorf("%d is not enough bytes for Arguments", l)
	}
	n := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	for i := 0; i < int(n); i++ {
		var arg Argument
		var err error
		switch ArgumentValueType(data[0]) {
		case ArgumentInteger:
			var ia IntegerArgument
			err = ia.UnmarshalBinary(data)
			arg = &ia
		case ArgumentValueTrue, ArgumentValueFalse:
			var ba BooleanArgument
			err = ba.UnmarshalBinary(data)
			arg = &ba
		case ArgumentBinary:
			var ba BinaryArgument
			err = ba.UnmarshalBinary(data)
			arg = &ba
		case ArgumentString:
			var sa StringArgument
			err = sa.UnmarshalBinary(data)
			arg = &sa
		case ArgumentList:
			var aa ListArgument
			err = aa.UnmarshalBinary(data)
			arg = &aa
		default:
			return errors.Errorf("unsupported argument type %d", data[0])
		}
		if err != nil {
			return errors.Wrap(err, "failed unmarshal Arguments from bytes")
		}
		a.Append(arg)
		data = data[arg.BinarySize():]
	}
	return nil
}

func (a Arguments) BinarySize() int {
	r := 4
	for _, arg := range a {
		r += arg.BinarySize()
	}
	return r
}

type IntegerArgument struct {
	Value int64
}

func NewIntegerArgument(i int64) *IntegerArgument {
	return &IntegerArgument{i}
}

// GetValueType returns the value type of the entry.
func (a IntegerArgument) GetValueType() ArgumentValueType {
	return ArgumentInteger
}

func (a IntegerArgument) BinarySize() int {
	return integerArgumentLen
}

// MarshalBinary marshals the integer argument in its bytes representation.
func (a IntegerArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	pos := 0
	buf[pos] = byte(ArgumentInteger)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(a.Value))
	return buf, nil
}

// Serialize the integer argument in its bytes representation.
func (a IntegerArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentInteger))
	if err != nil {
		return err
	}
	return s.Uint64(uint64(a.Value))
}

// UnmarshalBinary reads binary representation of integer argument to the structure.
func (a *IntegerArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < integerArgumentLen {
		return errors.Errorf("invalid data length for IntegerArgument, expected not less than %d, received %d", integerArgumentLen, l)
	}
	if t := data[0]; t != byte(ArgumentInteger) {
		return errors.Errorf("unexpected value type %d for IntegerArgument, expected %d", t, ArgumentInteger)
	}
	a.Value = int64(binary.BigEndian.Uint64(data[1:]))
	return nil
}

// MarshalJSON writes a JSON representation of integer argument.
func (a IntegerArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V int    `json:"value"`
	}{a.GetValueType().String(), int(a.Value)})
}

// UnmarshalJSON reads an integer argument from its JSON representation.
func (a *IntegerArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string `json:"type"`
		V int    `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize integer argument from JSON")
	}
	a.Value = int64(tmp.V)
	return nil
}

// BooleanArgument represents a key-value pair that stores a bool value.
type BooleanArgument struct {
	Value bool
}

// GetValueType returns the data type (Boolean) of the argument.
func (a BooleanArgument) GetValueType() ArgumentValueType {
	return ArgumentBoolean
}

func (a BooleanArgument) BinarySize() int {
	return booleanArgumentLen
}

// MarshalBinary writes a byte representation of the boolean data entry.
func (a BooleanArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	if a.Value {
		buf[0] = byte(ArgumentValueTrue)
	} else {
		buf[0] = byte(ArgumentValueFalse)
	}
	return buf, nil
}

// Serialize argument to its byte representation.
func (a BooleanArgument) Serialize(s *serializer.Serializer) error {
	buf := byte(0)
	if a.Value {
		buf = byte(ArgumentValueTrue)
	} else {
		buf = byte(ArgumentValueFalse)
	}
	return s.Byte(buf)
}

// UnmarshalBinary reads a byte representation of the data entry.
func (a *BooleanArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < booleanArgumentLen {
		return errors.Errorf("invalid data length for BooleanArgument, expected not less than %d, received %d", booleanArgumentLen, l)
	}
	switch data[0] {
	case byte(ArgumentValueTrue):
		a.Value = true
	case byte(ArgumentValueFalse):
		a.Value = false
	default:
		return errors.Errorf("unexpected value (%d) for BooleanArgument", data[0])
	}
	return nil
}

// MarshalJSON writes the argument to a JSON representation.
func (a BooleanArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V bool   `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

// UnmarshalJSON reads the entry from its JSON representation.
func (a *BooleanArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string `json:"type"`
		V bool   `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize boolean argument from JSON")
	}
	a.Value = tmp.V
	return nil
}

// BinaryArgument represents an argument that stores binary value.
type BinaryArgument struct {
	Value []byte
}

// GetValueType returns the type of value (Binary) stored in an argument.
func (a BinaryArgument) GetValueType() ArgumentValueType {
	return ArgumentBinary
}

func (a BinaryArgument) BinarySize() int {
	return binaryArgumentMinLen + len(a.Value)
}

// MarshalBinary writes an argument to its byte representation.
func (a BinaryArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	pos := 0
	buf[pos] = byte(ArgumentBinary)
	pos++
	if err := PutBytesWithUInt32Len(buf[pos:], a.Value); err != nil {
		return nil, errors.Wrap(err, "failed to marshal BinaryArgument")
	}
	return buf, nil
}

// Serialize argument to its byte representation.
func (a BinaryArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentBinary))
	if err != nil {
		return err
	}
	return s.BytesWithUInt32Len(a.Value)
}

// UnmarshalBinary reads an argument from a binary representation.
func (a *BinaryArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < binaryArgumentMinLen {
		return errors.Errorf("invalid data length for BinaryArgument, expected not less than %d, received %d", binaryArgumentMinLen, l)
	}
	if t := data[0]; t != byte(ArgumentBinary) {
		return errors.Errorf("unexpected value type %d for BinaryArgument, expected %d", t, ArgumentBinary)
	}
	v, err := BytesWithUInt32Len(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BinaryArgument from bytes")
	}
	a.Value = v
	return nil
}

// MarshalJSON converts an argument to its JSON representation. Note that BASE64 is used to represent the binary value.
func (a BinaryArgument) MarshalJSON() ([]byte, error) {
	// TODO: support marshal BinaryArgument to JSON with `ByteVector` type field
	return json.Marshal(&struct {
		T string     `json:"type"`
		V ByteVector `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

// UnmarshalJSON converts JSON to a BinaryArgument structure. Value should be stored as BASE64 sting in JSON.
func (a *BinaryArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string     `json:"type"`
		V ByteVector `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize binary data entry from JSON")
	}
	a.Value = tmp.V
	return nil
}

// StringArgument structure is an argument that store a string value.
type StringArgument struct {
	Value string
}

func NewStringArgument(s string) *StringArgument {
	return &StringArgument{s}
}

// GetValueType returns the type of value of the argument.
func (a StringArgument) GetValueType() ArgumentValueType {
	return ArgumentString
}

func (a StringArgument) BinarySize() int {
	return stringArgumentMinLen + len(a.Value)
}

// MarshalBinary converts the argument to its byte representation.
func (a StringArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	pos := 0
	buf[pos] = byte(ArgumentString)
	pos++
	PutStringWithUInt32Len(buf[pos:], a.Value)
	return buf, nil
}

// Serialize argument to its byte representation.
func (a StringArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentString))
	if err != nil {
		return err
	}
	return s.StringWithUInt32Len(a.Value)
}

// UnmarshalBinary reads an StringArgument structure from bytes.
func (a *StringArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < stringArgumentMinLen {
		return errors.Errorf("invalid data length for StringArgument, expected not less than %d, received %d", stringArgumentMinLen, l)
	}
	if t := data[0]; t != byte(ArgumentString) {
		return errors.Errorf("unexpected value type %d for StringArgument, expected %d", t, ArgumentString)
	}
	v, err := StringWithUInt32Len(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StringArgument from bytes")
	}
	a.Value = v
	return nil
}

// MarshalJSON writes the entry to its JSON representation.
func (a StringArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V string `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

// UnmarshalJSON reads the entry from JSON.
func (a *StringArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string `json:"type"`
		V string `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize string data entry from JSON")
	}
	a.Value = tmp.V
	return nil
}

type ListArgument struct {
	Items Arguments
}

// GetValueType returns the type of value of the argument.
func (a ListArgument) GetValueType() ArgumentValueType {
	return ArgumentList
}

func (a ListArgument) BinarySize() int {
	return 1 + a.Items.BinarySize()
}

// MarshalBinary converts the argument to its byte representation.
func (a ListArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.BinarySize())
	pos := 0
	buf[pos] = byte(ArgumentList)
	pos++
	b, err := a.Items.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(buf[pos:], b)
	return buf, nil
}

// Serialize argument to its byte representation.
func (a ListArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentList))
	if err != nil {
		return err
	}
	return a.Items.Serialize(s)
}

// UnmarshalBinary reads an StringArgument structure from bytes.
func (a *ListArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < listArgumentMinLen {
		return errors.Errorf("invalid data length for ListArgument, expected not less than %d, received %d", listArgumentMinLen, l)
	}
	if t := data[0]; t != byte(ArgumentList) {
		return errors.Errorf("unexpected value type %d for ListArgument, expected %d", t, ArgumentList)
	}
	data = data[1:]
	args := new(Arguments)
	err := args.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	a.Items = *args
	return nil
}

// MarshalJSON writes the entry to its JSON representation.
func (a ListArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string     `json:"type"`
		V []Argument `json:"value"`
	}{a.GetValueType().String(), a.Items})
}

// UnmarshalJSON reads the entry from JSON.
func (a *ListArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string    `json:"type"`
		V Arguments `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize string data entry from JSON")
	}
	a.Items = tmp.V
	return nil
}

// FunctionCall structure represents the description of function called in the InvokeScript transaction.
type FunctionCall struct {
	Default   bool
	Name      string
	Arguments Arguments
}

const (
	tokenFunctionCall = 9
	tokenUserFunction = 1
)

func (c FunctionCall) Serialize(s *serializer.Serializer) error {
	if c.Default {
		return s.Byte(0)
	}
	err := s.Bytes([]byte{1, tokenFunctionCall, tokenUserFunction})
	if err != nil {
		return err
	}
	err = s.StringWithUInt32Len(c.Name)
	if err != nil {
		return err
	}
	err = c.Arguments.Serialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to serialize FunctionCall to bytes")
	}
	return nil
}

func (c FunctionCall) MarshalBinary() ([]byte, error) {
	if c.Default {
		return []byte{0}, nil
	}
	buf := make([]byte, c.BinarySize())
	buf[0] = 1
	buf[1] = tokenFunctionCall
	buf[2] = tokenUserFunction
	PutStringWithUInt32Len(buf[3:], c.Name)
	ab, err := c.Arguments.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal FunctionCall to bytes")
	}
	copy(buf[3+4+len(c.Name):], ab)
	return buf, nil
}

func (c *FunctionCall) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 1 {
		return errors.Errorf("%d is not enough bytes for FunctionCall", l)
	}
	if data[0] == 0 {
		c.Default = true
		return nil
	}
	data = data[1:]
	if l := len(data); l < 1+1+4 {
		return errors.Errorf("%d is not enough bytes of FunctionCall", l)
	}
	if data[0] != tokenFunctionCall {
		return errors.Errorf("unexpected expression type %d, expected E_FUNCALL", data[0])
	}
	if data[1] != tokenUserFunction {
		return errors.Errorf("unexpected function type %d, expected a user function", data[1])
	}
	var err error
	data = data[2:]
	c.Name, err = StringWithUInt32Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal FunctionCall from bytes")
	}
	data = data[4+len(c.Name):]
	args := Arguments{}
	err = args.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal FunctionCall from bytes")
	}
	c.Arguments = args
	return nil
}

// MarshalJSON writes the entry to its JSON representation.
func (c FunctionCall) MarshalJSON() ([]byte, error) {
	if c.Default {
		return []byte("null"), nil
	}
	tmp := struct {
		Name      string    `json:"function"`
		Arguments Arguments `json:"args"`
	}{c.Name, c.Arguments}
	return json.Marshal(tmp)
}

// UnmarshalJSON reads the entry from JSON.
func (c *FunctionCall) UnmarshalJSON(value []byte) error {
	str := string(value)
	if str == "null" || str == "{}" {
		c.Default = true
		return nil
	}
	var tmp struct {
		Name      string    `json:"function"`
		Arguments Arguments `json:"args"`
	}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize function call from JSON")
	}
	c.Default = false
	c.Name = tmp.Name
	c.Arguments = tmp.Arguments
	return nil
}

func (c FunctionCall) BinarySize() int {
	if c.Default {
		return 1
	}
	return 1 + 1 + 1 + 4 + len(c.Name) + c.Arguments.BinarySize()
}

type FullScriptTransfer struct {
	Amount    uint64
	Asset     OptionalAsset
	Recipient Recipient
	Sender    WavesAddress
	SenderPK  crypto.PublicKey
	Timestamp uint64
	ID        *crypto.Digest
}

func NewFullScriptTransfer(action *TransferScriptAction, sender WavesAddress, senderPK crypto.PublicKey, txID *crypto.Digest, timestamp uint64) (*FullScriptTransfer, error) {
	return &FullScriptTransfer{
		Amount:    uint64(action.Amount),
		Asset:     action.Asset,
		Recipient: action.Recipient,
		Sender:    sender,
		SenderPK:  senderPK,
		Timestamp: timestamp,
		ID:        txID,
	}, nil
}

func NewFullScriptTransferFromPaymentAction(action *AttachedPaymentScriptAction, sender WavesAddress, senderPK crypto.PublicKey, txID *crypto.Digest, timestamp uint64) (*FullScriptTransfer, error) {
	return &FullScriptTransfer{
		Amount:    uint64(action.Amount),
		Asset:     action.Asset,
		Recipient: action.Recipient,
		Sender:    sender,
		SenderPK:  senderPK,
		Timestamp: timestamp,
		ID:        txID,
	}, nil
}

// ScriptPayment part of InvokeScriptTransaction that describes attached payments that comes in with invoke.
type ScriptPayment struct {
	Amount uint64        `json:"amount"`
	Asset  OptionalAsset `json:"assetId"`
}

func (p ScriptPayment) MarshalBinary() ([]byte, error) {
	size := p.BinarySize()
	buf := make([]byte, size)
	pos := 0
	binary.BigEndian.PutUint16(buf[pos:], uint16(size-2))
	pos += 2
	binary.BigEndian.PutUint64(buf[pos:], p.Amount)
	pos += 8
	ab, err := p.Asset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize ScriptPayment to bytes")
	}
	copy(buf[pos:], ab)
	return buf, nil
}

func (p ScriptPayment) Serialize(s *serializer.Serializer) error {
	size := p.BinarySize()
	err := s.Uint16(uint16(size - 2))
	if err != nil {
		return err
	}
	err = s.Uint64(p.Amount)
	if err != nil {
		return err
	}
	err = p.Asset.Serialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to serialize ScriptPayment to bytes")
	}
	return nil
}

func (p *ScriptPayment) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 2 {
		return errors.Errorf("%d is not enough bytes for ScriptPayment", l)
	}
	size := int(binary.BigEndian.Uint16(data[:2]))
	if l := len(data[2:]); l < size {
		return errors.Errorf("%d is not enough bytes for ScriptPayment", l)
	}
	p.Amount = binary.BigEndian.Uint64(data[2:10])
	var a OptionalAsset
	err := a.UnmarshalBinary(data[10:])
	if err != nil {
		return errors.Wrap(err, "failed to deserialize ScriptPayment from bytes")
	}
	p.Asset = a
	return nil
}

func (p *ScriptPayment) BinarySize() int {
	return 2 + 8 + p.Asset.BinarySize()
}

// ScriptPayments list of payments attached to InvokeScriptTransaction.
type ScriptPayments []ScriptPayment

func (sps *ScriptPayments) Append(sp ScriptPayment) {
	*sps = append(*sps, sp)
}

func (sps ScriptPayments) MarshalBinary() ([]byte, error) {
	buf := make([]byte, sps.BinarySize())
	p := 0
	binary.BigEndian.PutUint16(buf[p:], uint16(len(sps)))
	p += 2
	for _, sp := range sps {
		b, err := sp.MarshalBinary()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal ScriptPayments to bytes")
		}
		copy(buf[p:], b)
		p += len(b)
	}
	return buf, nil
}

func (sps ScriptPayments) Serialize(s *serializer.Serializer) error {
	err := s.Uint16(uint16(len(sps)))
	if err != nil {
		return err
	}
	for _, sp := range sps {
		err := sp.Serialize(s)
		if err != nil {
			return errors.Wrap(err, "failed to marshal ScriptPayments to bytes")
		}
	}
	return nil
}

func (sps *ScriptPayments) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 2 {
		return errors.Errorf("%d is not enough bytes for ScriptPayments", l)
	}
	n := binary.BigEndian.Uint16(data)
	data = data[2:]
	for i := 0; i < int(n); i++ {
		var sp ScriptPayment
		err := sp.UnmarshalBinary(data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal ScriptPayments from bytes")
		}
		sps.Append(sp)
		data = data[sp.BinarySize():]
	}
	return nil
}

func (sps ScriptPayments) BinarySize() int {
	s := 2
	for _, p := range sps {
		s += p.BinarySize()
	}
	return s
}

type FullWavesBalance struct {
	Regular    uint64
	Generating uint64
	Available  uint64
	Effective  uint64
	LeaseIn    uint64
	LeaseOut   uint64
}

func (b *FullWavesBalance) ToProtobuf() *pb.BalanceResponse_WavesBalances {
	return &pb.BalanceResponse_WavesBalances{
		Regular:    int64(b.Regular),
		Generating: int64(b.Generating),
		Available:  int64(b.Available),
		Effective:  int64(b.Effective),
		LeaseIn:    int64(b.LeaseIn),
		LeaseOut:   int64(b.LeaseOut),
	}
}

type StateHash struct {
	BlockID BlockID
	SumHash crypto.Digest
	FieldsHashes
}

type FieldsHashes struct {
	DataEntryHash     crypto.Digest
	AccountScriptHash crypto.Digest
	AssetScriptHash   crypto.Digest
	LeaseStatusHash   crypto.Digest
	SponsorshipHash   crypto.Digest
	AliasesHash       crypto.Digest
	WavesBalanceHash  crypto.Digest
	AssetBalanceHash  crypto.Digest
	LeaseBalanceHash  crypto.Digest
}

type fieldsHashesJS struct {
	DataEntryHash     DigestWrapped `json:"dataEntryHash"`
	AccountScriptHash DigestWrapped `json:"accountScriptHash"`
	AssetScriptHash   DigestWrapped `json:"assetScriptHash"`
	LeaseStatusHash   DigestWrapped `json:"leaseStatusHash"`
	SponsorshipHash   DigestWrapped `json:"sponsorshipHash"`
	AliasesHash       DigestWrapped `json:"aliasHash"`
	WavesBalanceHash  DigestWrapped `json:"wavesBalanceHash"`
	AssetBalanceHash  DigestWrapped `json:"assetBalanceHash"`
	LeaseBalanceHash  DigestWrapped `json:"leaseBalanceHash"`
}

func (s FieldsHashes) MarshalJSON() ([]byte, error) {
	return json.Marshal(fieldsHashesJS{
		DigestWrapped(s.DataEntryHash),
		DigestWrapped(s.AccountScriptHash),
		DigestWrapped(s.AssetScriptHash),
		DigestWrapped(s.LeaseStatusHash),
		DigestWrapped(s.SponsorshipHash),
		DigestWrapped(s.AliasesHash),
		DigestWrapped(s.WavesBalanceHash),
		DigestWrapped(s.AssetBalanceHash),
		DigestWrapped(s.LeaseBalanceHash),
	})
}

func (s *FieldsHashes) UnmarshalJSON(value []byte) error {
	var sh fieldsHashesJS
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	return nil
}

func (s *StateHash) GenerateSumHash(prevSumHash []byte) error {
	h, err := crypto.NewFastHash()
	if err != nil {
		return err
	}
	if _, err := h.Write(prevSumHash); err != nil {
		return err
	}
	if _, err := h.Write(s.WavesBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AssetBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.DataEntryHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AccountScriptHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AssetScriptHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.LeaseBalanceHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.LeaseStatusHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.SponsorshipHash[:]); err != nil {
		return err
	}
	if _, err := h.Write(s.AliasesHash[:]); err != nil {
		return err
	}
	h.Sum(s.SumHash[:0])
	return nil
}

func (s *StateHash) MarshalBinary() []byte {
	idBytes := s.BlockID.Bytes()
	res := make([]byte, 1+len(idBytes)+crypto.DigestSize*10)
	res[0] = byte(len(idBytes))
	pos := 1
	copy(res[pos:pos+len(idBytes)], idBytes)
	pos += len(idBytes)
	copy(res[pos:pos+crypto.DigestSize], s.SumHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.DataEntryHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AccountScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetScriptHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseStatusHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.SponsorshipHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AliasesHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.WavesBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.AssetBalanceHash[:])
	pos += crypto.DigestSize
	copy(res[pos:pos+crypto.DigestSize], s.LeaseBalanceHash[:])
	return res
}

func (s *StateHash) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return errors.New("invalid data size")
	}
	idBytesLen := int(data[0])
	correctSize := 1 + idBytesLen + crypto.DigestSize*10
	if len(data) != correctSize {
		return errors.New("invalid data size")
	}
	var err error
	pos := 1
	s.BlockID, err = NewBlockIDFromBytes(data[pos : pos+idBytesLen])
	if err != nil {
		return err
	}
	pos += idBytesLen
	copy(s.SumHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.DataEntryHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AccountScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetScriptHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseStatusHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.SponsorshipHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AliasesHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.WavesBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.AssetBalanceHash[:], data[pos:pos+crypto.DigestSize])
	pos += crypto.DigestSize
	copy(s.LeaseBalanceHash[:], data[pos:pos+crypto.DigestSize])
	return nil
}

// DigestWrapped is required for state hashes API.
// The quickest way to use Hex for hashes in JSON in this particular case.
type DigestWrapped crypto.Digest

func (d DigestWrapped) MarshalJSON() ([]byte, error) {
	s := hex.EncodeToString(d[:])
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (d *DigestWrapped) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(b) != crypto.DigestSize {
		return errors.New("bad size")
	}
	copy(d[:], b[:crypto.DigestSize])
	return nil
}

type stateHashJS struct {
	BlockID BlockID       `json:"blockId"`
	SumHash DigestWrapped `json:"stateHash"`
	fieldsHashesJS
}

func (s StateHash) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toStateHashJS())
}

func (s *StateHash) UnmarshalJSON(value []byte) error {
	var sh stateHashJS
	if err := json.Unmarshal(value, &sh); err != nil {
		return err
	}
	s.BlockID = sh.BlockID
	s.SumHash = crypto.Digest(sh.SumHash)
	s.DataEntryHash = crypto.Digest(sh.DataEntryHash)
	s.AccountScriptHash = crypto.Digest(sh.AccountScriptHash)
	s.AssetScriptHash = crypto.Digest(sh.AssetScriptHash)
	s.LeaseStatusHash = crypto.Digest(sh.LeaseStatusHash)
	s.SponsorshipHash = crypto.Digest(sh.SponsorshipHash)
	s.AliasesHash = crypto.Digest(sh.AliasesHash)
	s.WavesBalanceHash = crypto.Digest(sh.WavesBalanceHash)
	s.AssetBalanceHash = crypto.Digest(sh.AssetBalanceHash)
	s.LeaseBalanceHash = crypto.Digest(sh.LeaseBalanceHash)
	return nil
}

func (s *StateHash) toStateHashJS() stateHashJS {
	return stateHashJS{
		s.BlockID,
		DigestWrapped(s.SumHash),
		fieldsHashesJS{
			DigestWrapped(s.DataEntryHash),
			DigestWrapped(s.AccountScriptHash),
			DigestWrapped(s.AssetScriptHash),
			DigestWrapped(s.LeaseStatusHash),
			DigestWrapped(s.SponsorshipHash),
			DigestWrapped(s.AliasesHash),
			DigestWrapped(s.WavesBalanceHash),
			DigestWrapped(s.AssetBalanceHash),
			DigestWrapped(s.LeaseBalanceHash),
		},
	}
}

type StateHashDebug struct {
	stateHashJS
	Height  uint64 `json:"height,omitempty"`
	Version string `json:"version,omitempty"`
}

func NewStateHashJSDebug(s StateHash, h uint64, v string) StateHashDebug {
	return StateHashDebug{s.toStateHashJS(), h, v}
}

func (s StateHashDebug) GetStateHash() *StateHash {
	sh := &StateHash{
		BlockID: s.BlockID,
		SumHash: crypto.Digest(s.SumHash),
		FieldsHashes: FieldsHashes{
			crypto.Digest(s.DataEntryHash),
			crypto.Digest(s.AccountScriptHash),
			crypto.Digest(s.AssetScriptHash),
			crypto.Digest(s.LeaseStatusHash),
			crypto.Digest(s.SponsorshipHash),
			crypto.Digest(s.AliasesHash),
			crypto.Digest(s.WavesBalanceHash),
			crypto.Digest(s.AssetBalanceHash),
			crypto.Digest(s.LeaseBalanceHash),
		},
	}
	return sh
}

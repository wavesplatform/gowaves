package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

const (
	//WavesAssetName is the default name for basic WAVES asset.
	WavesAssetName       = "WAVES"
	quotedWavesAssetName = "\"" + WavesAssetName + "\""
	orderLen             = crypto.PublicKeySize + crypto.PublicKeySize + 1 + 1 + 1 + 8 + 8 + 8 + 8 + 8
	orderV2FixedBodyLen  = 1 + orderLen
	orderV3FixedBodyLen  = 1 + orderLen + 1
	orderV1MinLen        = crypto.SignatureSize + orderLen
	orderV2MinLen        = orderV2FixedBodyLen + proofsMinLen
	orderV3MinLen        = orderV3FixedBodyLen + proofsMinLen
	jsonNull             = "null"
	integerArgumentLen   = 1 + 8
	booleanArgumentLen   = 1
	binaryArgumentMinLen = 1 + 4
	stringArgumentMinLen = 1 + 4
	PriceConstant        = 100000000
	MaxOrderAmount       = 100 * PriceConstant * PriceConstant
	MaxOrderTTL          = uint64((30 * 24 * time.Hour) / time.Millisecond)
	maxKeySize           = 100
	maxValueSize         = 32767

	maxInvokeTransfers           = 10
	maxInvokeWrites              = 100
	maxInvokeWriteKeySizeInBytes = 100
	maxWriteSetSizeInBytes       = 5 * 1024
)

type Timestamp = uint64
type Score = big.Int
type Scheme = byte
type Height = uint64

var jsonNullBytes = []byte{0x6e, 0x75, 0x6c, 0x6c}

type Bytes []byte

func (a Bytes) WriteTo(w io.Writer) (int64, error) {
	rs, err := w.Write(a)
	return int64(rs), err
}

type WrapWriteTo struct {
	buf *bytes.Buffer
}

func (a WrapWriteTo) Read(b []byte) (int, error) {
	return a.buf.Read(b)
}

// B58Bytes represents bytes as Base58 string in JSON
type B58Bytes []byte

// String represents underlying bytes as Base58 string
func (b B58Bytes) String() string {
	return base58.Encode(b)
}

// MarshalJSON writes B58Bytes Value as JSON string
func (b B58Bytes) MarshalJSON() ([]byte, error) {
	s := base58.Encode(b)
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return []byte(sb.String()), nil
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
		*b = B58Bytes([]byte{})
		return nil
	}
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode B58Bytes")
	}
	*b = B58Bytes(v)
	return nil
}

func (b B58Bytes) Bytes() []byte {
	return b
}

// OptionalAsset represents an optional asset identification
type OptionalAsset struct {
	Present bool
	ID      crypto.Digest
}

//NewOptionalAssetFromString creates an OptionalAsset structure from its string representation.
func NewOptionalAssetFromString(s string) (*OptionalAsset, error) {
	switch strings.ToUpper(s) {
	case WavesAssetName, "":
		return &OptionalAsset{Present: false}, nil
	default:
		a, err := crypto.NewDigestFromBase58(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create OptionalAsset from Base58 string")
		}
		return &OptionalAsset{Present: true, ID: a}, nil
	}
}

func NewOptionalAssetFromBytes(b []byte) (*OptionalAsset, error) {
	if len(b) == 0 {
		return &OptionalAsset{}, nil
	}

	a, err := crypto.NewDigestFromBytes(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OptionalAsset from Base58 string")
	}
	return &OptionalAsset{Present: true, ID: a}, nil
}

func NewOptionalAssetFromDigest(d crypto.Digest) *OptionalAsset {
	return &OptionalAsset{Present: true, ID: d}
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
		*a = OptionalAsset{Present: false}
	default:
		var d crypto.Digest
		err := d.UnmarshalJSON(value)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal OptionalAsset")
		}
		*a = OptionalAsset{Present: true, ID: d}
	}
	return nil
}

func (a OptionalAsset) binarySize() int {
	s := 1
	if a.Present {
		s += crypto.DigestSize
	}
	return s
}

//MarshalBinary marshals the optional asset to its binary representation.
func (a OptionalAsset) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	PutBool(buf, a.Present)
	if a.Present {
		copy(buf[1:], a.ID[:])
	}
	return buf, nil
}

//WriteTo writes its binary representation.
func (a OptionalAsset) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := a.Serialize(s)
	if err != nil {
		return 0, err
	}
	return s.N(), nil
}

//Serialize into binary representation.
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

//UnmarshalBinary reads the OptionalAsset structure from its binary representation.
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

//Attachment represents the additional data stored in Transfer and MassTransfer transactions.
type Attachment string

// NewAttachmentFromBase58 creates an Attachment structure from its base58 string representation.
func NewAttachmentFromBase58(s string) (Attachment, error) {
	v, err := base58.Decode(s)
	if err != nil {
		return "", err
	}
	return Attachment(v), nil
}

// String returns Attachment's string representation
func (a Attachment) String() string {
	return string(a)
}

func (a Attachment) Bytes() []byte {
	return []byte(a)
}

// MarshalJSON writes Attachment as a JSON string Value
func (a Attachment) MarshalJSON() ([]byte, error) {
	b := []byte(a)
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(base58.Encode(b))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads Attachment from a JSON string Value
func (a *Attachment) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Attachment from JSON")
	}

	if s == "" {
		*a = Attachment("")
		return nil
	}

	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode Attachment from JSON Value")
	}
	*a = Attachment(string(v))
	return nil
}

//OrderType an alias for byte that encodes the type of OrderV1 (BUY|SELL).
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

func (o OrderType) String() string {
	if o == 0 {
		return buyOrderName
	}
	return sellOrderName
}

//MarshalJSON writes value of OrderType to JSON representation.
func (o OrderType) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(o.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

//UnmarshalJSON reads the OrderType value from JSON value.
func (o *OrderType) UnmarshalJSON(value []byte) error {
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
		*o = Buy
	case sellOrderName:
		*o = Sell
	default:
		return errors.Errorf("incorrect OrderType '%s'", s)
	}
	return nil
}

//AssetPair is a pair of assets in ExchangeTransaction.
type AssetPair struct {
	AmountAsset OptionalAsset `json:"amountAsset"`
	PriceAsset  OptionalAsset `json:"priceAsset"`
}

func (p AssetPair) ToProtobuf() *g.AssetPair {
	return &g.AssetPair{AmountAssetId: p.AmountAsset.ToID(), PriceAssetId: p.PriceAsset.ToID()}
}

type OrderVersion struct {
	Version byte `json:"version"`
}

type Order interface {
	GetID() ([]byte, error)
	GetVersion() byte
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
	GetSenderPK() crypto.PublicKey
	BodyMarshalBinary() ([]byte, error)
	GetProofs() (*ProofsV1, error)
	Verify(crypto.PublicKey) (bool, error)
	ToProtobuf(scheme Scheme) *g.Order
}

func OrderToOrderBody(o Order) (OrderBody, error) {
	switch o.GetVersion() {
	case 1:
		o, ok := o.(*OrderV1)
		if !ok {
			return OrderBody{}, errors.New("failed to cast an order version 1 to *OrderV1")
		}
		return o.OrderBody, nil
	case 2:
		o, ok := o.(*OrderV2)
		if !ok {
			return OrderBody{}, errors.New("failed to cast an order version 2 to *OrderV2")
		}
		return o.OrderBody, nil
	case 3:
		o, ok := o.(*OrderV3)
		if !ok {
			return OrderBody{}, errors.New("failed to cast an order version 3 to *OrderV3")
		}
		return o.OrderBody, nil
	default:
		return OrderBody{}, errors.New("invalid order version")
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

func (o OrderBody) ToProtobuf(scheme Scheme) *g.Order {
	return &g.Order{
		ChainId:          int32(scheme),
		SenderPublicKey:  o.SenderPK.Bytes(),
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
	s, err := o.SpendAmount(o.Amount, o.Price)
	if err != nil {
		return false, err
	}
	if s == 0 {
		return false, errors.New("spend amount should be positive")
	}
	if !o.SpendAsset().Present && !validJVMLong(s+o.MatcherFee) {
		return false, errors.New("sum of spend asset amount and matcher fee overflows JVM long")
	}
	r, err := o.ReceiveAmount(o.Amount, o.Price)
	if err != nil {
		return false, err
	}
	if r == 0 {
		return false, errors.New("receive amount should be positive")
	}
	if o.Timestamp == 0 {
		return false, errors.New("timestamp should be positive")
	}
	if o.Expiration == 0 {
		return false, errors.New("expiration should be positive")
	}
	return true, nil
}

func (o OrderBody) GetSenderPK() crypto.PublicKey {
	return o.SenderPK
}

func (o *OrderBody) SpendAmount(matchAmount, matchPrice uint64) (uint64, error) {
	if o.OrderType == Sell {
		return matchAmount, nil
	}
	return otherAmount(matchAmount, matchPrice, "spend")
}

func (o *OrderBody) ReceiveAmount(matchAmount, matchPrice uint64) (uint64, error) {
	if o.OrderType == Buy {
		return matchAmount, nil
	}
	return otherAmount(matchAmount, matchPrice, "receive")
}

var (
	bigPriceConstant = big.NewInt(PriceConstant)
)

func otherAmount(amount, price uint64, name string) (uint64, error) {
	a := big.NewInt(0).SetUint64(amount)
	p := big.NewInt(0).SetUint64(price)
	r := big.NewInt(0).Mul(a, p)
	r = big.NewInt(0).Div(r, bigPriceConstant)
	if !r.IsUint64() {
		return 0, errors.Errorf("%s amount is too large", name)
	}
	return r.Uint64(), nil
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

func (o *OrderBody) unmarshalBinary(data []byte) error {
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

//OrderV1 is an order created and signed by user. Two matched orders builds up an Exchange transaction.
type OrderV1 struct {
	ID        *crypto.Digest    `json:"id,omitempty"`
	Signature *crypto.Signature `json:"signature,omitempty"`
	OrderBody
}

func (o OrderV1) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: nil, Amount: int64(o.MatcherFee)}
	res.Version = 1
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

//NewUnsignedOrderV1 creates the new unsigned order.
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
	return o.OrderBody.unmarshalBinary(data)
}

//Sign adds a signature to the order.
func (o *OrderV1) Sign(secretKey crypto.SecretKey) error {
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

//Verify checks that the order's signature is valid.
func (o *OrderV1) Verify(publicKey crypto.PublicKey) (bool, error) {
	if o.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV1")
	}
	return crypto.Verify(publicKey, *o.Signature, b), nil
}

//MarshalBinary writes order to its bytes representation.
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

//Serialize order to its bytes representation.
func (o *OrderV1) Serialize(s *serializer.Serializer) error {
	err := o.BodySerialize(s)
	if err != nil {
		return errors.Wrap(err, "failed to marshal OrderV1 to bytes")
	}
	return s.Bytes(o.Signature[:])
}

//UnmarshalBinary reads an order from its binary representation.
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

//OrderV2 is an order created and signed by user. Two matched orders builds up an Exchange transaction. Version 2 with proofs.
type OrderV2 struct {
	Version byte           `json:"version"`
	ID      *crypto.Digest `json:"id,omitempty"`
	Proofs  *ProofsV1      `json:"proofs,omitempty"`
	OrderBody
}

func (o OrderV2) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: nil, Amount: int64(o.MatcherFee)}
	res.Version = 2
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

//NewUnsignedOrderV2 creates the new unsigned order.
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
	err := oo.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2 from bytes")
	}
	o.OrderBody = oo
	return nil
}

//Sign adds a signature to the order.
func (o *OrderV2) Sign(secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV2")
	}
	if o.Proofs == nil {
		o.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = o.Proofs.Sign(0, secretKey, b)
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

//Verify checks that the order's signature is valid.
func (o *OrderV2) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV2")
	}
	return o.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes order to its bytes representation.
func (o *OrderV2) MarshalBinary() ([]byte, error) {
	bb, err := o.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV2 to bytes")
	}
	bl := len(bb)
	pb, err := o.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV2 to bytes")
	}
	buf := make([]byte, bl+len(pb))
	copy(buf, bb)
	copy(buf[bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads an order from its binary representation.
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

func (o OrderV3) ToProtobuf(scheme Scheme) *g.Order {
	res := o.OrderBody.ToProtobuf(scheme)
	res.MatcherFee = &g.Amount{AssetId: o.MatcherFeeAsset.ToID(), Amount: int64(o.MatcherFee)}
	res.Version = 3
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

//NewUnsignedOrderV3 creates the new unsigned order.
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
	err := oo.unmarshalBinary(data[pos:])
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

//Sign adds a signature to the order.
func (o *OrderV3) Sign(secretKey crypto.SecretKey) error {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV3")
	}
	if o.Proofs == nil {
		o.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = o.Proofs.Sign(0, secretKey, b)
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

//Verify checks that the order's signature is valid.
func (o *OrderV3) Verify(publicKey crypto.PublicKey) (bool, error) {
	b, err := o.BodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV3")
	}
	return o.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes order to its bytes representation.
func (o *OrderV3) MarshalBinary() ([]byte, error) {
	bb, err := o.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV3 to bytes")
	}
	bl := len(bb)
	pb, err := o.Proofs.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV3 to bytes")
	}
	buf := make([]byte, bl+len(pb))
	copy(buf, bb)
	copy(buf[bl:], pb)
	return buf, nil
}

//UnmarshalBinary reads an order from its binary representation.
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

const (
	proofsVersion  byte = 1
	proofsMinLen        = 1 + 2
	proofsMaxCount      = 8
	proofMaxSize        = 64
)

//ProofsV1 is a collection of proofs.
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

//String gives a string representation of the proofs collection.
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

//MarshalJSON writes the proofs to JSON.
func (p ProofsV1) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Proofs)
}

//UnmarshalJSON reads the proofs from JSON.
func (p *ProofsV1) UnmarshalJSON(value []byte) error {
	var tmp []B58Bytes
	err := json.Unmarshal(value, &tmp)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal ProofsV1 from JSON")
	}
	p.Version = proofsVersion
	p.Proofs = tmp
	return nil
}

//MarshalBinary writes the proofs to its binary form.
func (p *ProofsV1) MarshalBinary() ([]byte, error) {
	buf := make([]byte, p.binarySize())
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

//Serialize proofs to its binary form.
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

//UnmarshalBinary reads the proofs from its binary representation.
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

//Sign creates a signature and stores it as a proof at given position.
func (p *ProofsV1) Sign(pos int, key crypto.SecretKey, data []byte) error {
	if pos < 0 || pos > proofsMaxCount {
		return errors.Errorf("failed to create proof at position %d, allowed positions from 0 to %d", pos, proofsMaxCount-1)
	}
	if len(p.Proofs)-1 < pos {
		s, err := crypto.Sign(key, data)
		if err != nil {
			return errors.Errorf("crypto.Sign(): %v", err)
		}
		p.Proofs = append(p.Proofs[:pos], append([]B58Bytes{s[:]}, p.Proofs[pos:]...)...)
	} else {
		pr := p.Proofs[pos]
		if len(pr) > 0 {
			return errors.Errorf("unable to overwrite non-empty proof at position %d", pos)
		}
		s, err := crypto.Sign(key, data)
		if err != nil {
			return errors.Errorf("crypto.Sign(): %v", err)
		}
		copy(pr[:], s[:])
	}
	return nil
}

//Verify checks that the proof at given position is a valid signature.
func (p *ProofsV1) Verify(pos int, key crypto.PublicKey, data []byte) (bool, error) {
	if len(p.Proofs) <= pos {
		return false, errors.Errorf("no proof at position %d", pos)
	}
	var sig crypto.Signature
	sb := p.Proofs[pos]
	if l := len(sb); l != crypto.SignatureSize {
		return false, errors.Errorf("unexpected signature size %d, expected %d", l, crypto.SignatureSize)
	}
	copy(sig[:], sb)
	return crypto.Verify(key, sig, data), nil
}

func (p *ProofsV1) binarySize() int {
	pl := 0
	if p != nil {
		for _, e := range p.Proofs {
			pl += len(e) + 2
		}
	}
	return proofsMinLen + pl
}

// ValueType is an alias for byte that encodes the value type.
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
	default:
		return ""
	}
}

//Supported value types.
const (
	DataInteger DataValueType = iota
	DataBoolean
	DataBinary
	DataString
)

// ValueType is an alias for byte that encodes the value type.
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
	default:
		return ""
	}
}

const (
	ArgumentInteger ArgumentValueType = iota
	ArgumentBinary
	ArgumentString
	ArgumentBoolean
)

//Special values to represent Boolean value
const (
	BooleanTrue  = 6
	BooleanFalse = 7
)

//DataEntry is a common interface of all types of data entries.
//The interface is used to store different types of data entries in one slice.
type DataEntry interface {
	GetKey() string
	SetKey(string)

	GetValueType() DataValueType
	MarshalValue() ([]byte, error)
	UnmarshalValue([]byte) error

	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	Valid() (bool, error)
	binarySize() int

	ToProtobuf() *g.DataTransactionData_DataEntry
}

var bytesToDataEntry = map[DataValueType]reflect.Type{
	DataInteger: reflect.TypeOf(IntegerDataEntry{}),
	DataBoolean: reflect.TypeOf(BooleanDataEntry{}),
	DataString:  reflect.TypeOf(StringDataEntry{}),
	DataBinary:  reflect.TypeOf(BinaryDataEntry{}),
}

func NewDataEntryFromBytes(data []byte) (DataEntry, error) {
	if len(data) < 1 {
		return nil, errors.New("invalid data size")
	}
	valueType, err := extractValueType(data)
	if err != nil {
		return nil, errors.New("failed to extract type of data entry")
	}
	entryType, ok := bytesToDataEntry[valueType]
	if !ok {
		return nil, errors.New("bad value type byte")
	}
	entry, ok := reflect.New(entryType).Interface().(DataEntry)
	if !ok {
		panic("This entry type does not implement DataEntry interface")
	}
	if err := entry.UnmarshalBinary(data); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal entry")
	}
	return entry, nil
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

//IntegerDataEntry stores int64 value.
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

func (e IntegerDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(utf16.Encode([]rune(e.Key))) > maxKeySize {
		return false, errors.New("key is too large")
	}
	return true, nil
}

//GetKey returns the key of data entry.
func (e IntegerDataEntry) GetKey() string {
	return e.Key
}

//SetKey sets the key of data entry.
func (e *IntegerDataEntry) SetKey(key string) {
	e.Key = key
}

//GetValueType returns the value type of the entry.
func (e IntegerDataEntry) GetValueType() DataValueType {
	return DataInteger
}

func (e IntegerDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 8
}

//MarshalValue marshals the integer data entry value in its bytes representation.
func (e IntegerDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+8)
	pos := 0
	buf[pos] = byte(DataInteger)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(e.Value))
	return buf, nil
}

//UnmarshalBinary reads binary representation of integer data entry value to the structure.
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

//MarshalBinary marshals the integer data entry in its bytes representation.
func (e IntegerDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
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

//UnmarshalBinary reads binary representation of integer data entry to the structure.
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

//MarshalJSON writes a JSON representation of integer data entry.
func (e IntegerDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V int    `json:"value"`
	}{e.Key, e.GetValueType().String(), int(e.Value)})
}

//UnmarshalJSON reads an integer data entry from its JSON representation.
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

//BooleanDataEntry represents a key-value pair that stores a bool value.
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

func (e BooleanDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(utf16.Encode([]rune(e.Key))) > maxKeySize {
		return false, errors.New("key is too large")
	}
	return true, nil
}

//GetKey returns the key of data entry.
func (e BooleanDataEntry) GetKey() string {
	return e.Key
}

//SetKey sets the key of data entry.
func (e *BooleanDataEntry) SetKey(key string) {
	e.Key = key
}

//GetValueType returns the data type (Boolean) of the entry.
func (e BooleanDataEntry) GetValueType() DataValueType {
	return DataBoolean
}

func (e BooleanDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 1
}

//MarshalValue writes a byte representation of the boolean data entry value.
func (e BooleanDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+1)
	pos := 0
	buf[pos] = byte(DataBoolean)
	pos++
	PutBool(buf[pos:], e.Value)
	return buf, nil
}

//UnmarshalValue reads a byte representation of the data entry value.
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

//MarshalBinary writes a byte representation of the boolean data entry.
func (e BooleanDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
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

//UnmarshalBinary reads a byte representation of the data entry.
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

//MarshalJSON writes the data entry to a JSON representation.
func (e BooleanDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V bool   `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

//UnmarshalJSON reads the entry from its JSON representation.
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

//BinaryDataEntry represents a key-value data entry that stores binary value.
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

func (e BinaryDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(utf16.Encode([]rune(e.Key))) > maxKeySize {
		return false, errors.New("key is too large")
	}
	if len(e.Value) > maxValueSize {
		return false, errors.New("value is too large")
	}
	return true, nil
}

//GetKey returns the key of data entry.
func (e BinaryDataEntry) GetKey() string {
	return e.Key
}

//SetKey sets the key of data entry.
func (e *BinaryDataEntry) SetKey(key string) {
	e.Key = key
}

//GetValueType returns the type of value (Binary) stored in an entry.
func (e BinaryDataEntry) GetValueType() DataValueType {
	return DataBinary
}

func (e BinaryDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

//MarshalValue writes an entry value to its byte representation.
func (e BinaryDataEntry) MarshalValue() ([]byte, error) {
	pos := 0
	buf := make([]byte, 1+2+len(e.Value))
	buf[pos] = byte(DataBinary)
	pos++
	PutBytesWithUInt16Len(buf[pos:], e.Value)
	return buf, nil
}

//UnmarshalValue reads an entry value from a binary representation.
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

//MarshalBinary writes an entry to its byte representation.
func (e BinaryDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
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

//UnmarshalBinary reads an entry from a binary representation.
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

//MarshalJSON converts an entry to its JSON representation. Note that BASE64 is used to represent the binary value.
func (e BinaryDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V Script `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

//UnmarshalJSON converts JSON to a BinaryDataEntry structure. Value should be stored as BASE64 sting in JSON.
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

//StringDataEntry structure is a key-value pair to store a string value.
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

func (e StringDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(utf16.Encode([]rune(e.Key))) > maxKeySize {
		return false, errors.New("key is too large")
	}
	if len(e.Value) > maxValueSize {
		return false, errors.New("value is too large")
	}
	return true, nil
}

//GetKey returns the key of key-value pair.
func (e StringDataEntry) GetKey() string {
	return e.Key
}

//SetKey sets the key of data entry.
func (e *StringDataEntry) SetKey(key string) {
	e.Key = key
}

//GetValueType returns the type of value in key-value entry.
func (e StringDataEntry) GetValueType() DataValueType {
	return DataString
}

func (e StringDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

//MarshalValue converts the data entry value to its byte representation.
func (e StringDataEntry) MarshalValue() ([]byte, error) {
	buf := make([]byte, 1+2+len(e.Value))
	pos := 0
	buf[pos] = byte(DataString)
	pos++
	PutStringWithUInt16Len(buf[pos:], e.Value)
	return buf, nil
}

//UnmarshalValue reads StringDataEntry value from bytes.
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

//MarshalBinary converts the data entry to its byte representation.
func (e StringDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
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

//UnmarshalBinary reads StringDataEntry structure from bytes.
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

//MarshalJSON writes the entry to its JSON representation.
func (e StringDataEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		K string `json:"key"`
		T string `json:"type"`
		V string `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

//UnmarshalJSON reads the entry from JSON.
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

//DataEntryType is the assistive structure used to get the type of DataEntry while unmarshal form JSON.
type DataEntryType struct {
	Type string `json:"type"`
}

func guessDataEntryType(dataEntryType DataEntryType) (DataEntry, error) {
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
	}
	if r == nil {
		return nil, errors.Errorf("unknown value type '%s' of DataEntry", dataEntryType.Type)
	}
	return r, nil
}

// DataEntries the slice of various entries of DataTransaction
type DataEntries []DataEntry

// UnmarshalJSOL special method to unmarshal DataEntries from JSON with detection of real type of each entry.
func (e *DataEntries) UnmarshalJSON(data []byte) error {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to unmarshal DataEntries from JSON") }

	var ets []DataEntryType
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
	Bytes      []byte
	Base64     string
	Complexity uint64
}

func (s ScriptInfo) ToProtobuf() *g.ScriptData {
	return &g.ScriptData{
		ScriptBytes: s.Bytes,
		ScriptText:  s.Base64,
		Complexity:  int64(s.Complexity),
	}
}

type Script []byte

func (s Script) ToProtobuf() *g.Script {
	return &g.Script{
		Bytes: s,
	}
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
	if value[0] != '"' || value[len(value)-1] != '"' {
		return wrapError(errors.New("no quotes"))
	}
	value = value[1 : len(value)-1]
	if !bytes.Equal(value[0:7], scriptPrefixBytes) {
		return wrapError(errors.New("no prefix"))
	}
	value = value[7:]
	sb := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
	n, err := base64.StdEncoding.Decode(sb, value)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Script form JSON")
	}
	*s = Script(sb[:n])
	return nil
}

type Argument interface {
	GetValueType() ArgumentValueType
	MarshalBinary() ([]byte, error)
	binarySize() int
	Serialize(*serializer.Serializer) error
}

//DataEntryType is the assistive structure used to get the type of DataEntry while unmarshal form JSON.
type ArgumentType struct {
	Type string `json:"type"`
}

func guessArgumentType(argumentType ArgumentType) (Argument, error) {
	var r Argument
	switch argumentType.Type {
	case "integer":
		r = &IntegerArgument{}
	case "boolean":
		r = &BooleanArgument{}
	case "binary":
		r = &BinaryArgument{}
	case "string":
		r = &StringArgument{}
	}
	if r == nil {
		return nil, errors.Errorf("unknown value type '%s' of Argument", argumentType.Type)
	}
	return r, nil
}

type Arguments []Argument

//Append adds an argument to the Arguments list.
func (a *Arguments) Append(arg Argument) {
	*a = append(*a, arg)
}

//UnmarshalJSON custom JSON deserialization method.
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
	buf := make([]byte, a.binarySize())
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
		case BooleanTrue, BooleanFalse:
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
		default:
			return errors.Errorf("unsupported argument type %d", data[0])
		}
		if err != nil {
			return errors.Wrap(err, "failed unmarshal Arguments from bytes")
		}
		a.Append(arg)
		data = data[arg.binarySize():]
	}
	return nil
}

func (a Arguments) binarySize() int {
	r := 4
	for _, arg := range a {
		r += arg.binarySize()
	}
	return r
}

type IntegerArgument struct {
	Value int64
}

func NewIntegerArgument(i int64) *IntegerArgument {
	return &IntegerArgument{i}
}

//GetValueType returns the value type of the entry.
func (a IntegerArgument) GetValueType() ArgumentValueType {
	return ArgumentInteger
}

func (a IntegerArgument) binarySize() int {
	return integerArgumentLen
}

//MarshalBinary marshals the integer argument in its bytes representation.
func (a IntegerArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(ArgumentInteger)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(a.Value))
	return buf, nil
}

//Serialize the integer argument in its bytes representation.
func (a IntegerArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentInteger))
	if err != nil {
		return err
	}
	return s.Uint64(uint64(a.Value))
}

//UnmarshalBinary reads binary representation of integer argument to the structure.
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

//MarshalJSON writes a JSON representation of integer argument.
func (a IntegerArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V int    `json:"value"`
	}{a.GetValueType().String(), int(a.Value)})
}

//UnmarshalJSON reads an integer argument from its JSON representation.
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

//BooleanArgument represents a key-value pair that stores a bool value.
type BooleanArgument struct {
	Value bool
}

func NewBooleanArgument(b bool) *BooleanArgument {
	return &BooleanArgument{b}
}

//GetValueType returns the data type (Boolean) of the argument.
func (a BooleanArgument) GetValueType() ArgumentValueType {
	return ArgumentBoolean
}

func (a BooleanArgument) binarySize() int {
	return booleanArgumentLen
}

//MarshalBinary writes a byte representation of the boolean data entry.
func (a BooleanArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	if a.Value {
		buf[0] = BooleanTrue
	} else {
		buf[0] = BooleanFalse
	}
	return buf, nil
}

//Serialize argument to its byte representation.
func (a BooleanArgument) Serialize(s *serializer.Serializer) error {
	buf := byte(0)
	if a.Value {
		buf = BooleanTrue
	} else {
		buf = BooleanFalse
	}
	return s.Byte(buf)
}

//UnmarshalBinary reads a byte representation of the data entry.
func (a *BooleanArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < booleanArgumentLen {
		return errors.Errorf("invalid data length for BooleanArgument, expected not less than %d, received %d", booleanArgumentLen, l)
	}
	switch data[0] {
	case BooleanTrue:
		a.Value = true
	case BooleanFalse:
		a.Value = false
	default:
		return errors.Errorf("unexpected value (%d) for BooleanArgument", data[0])
	}
	return nil
}

//MarshalJSON writes the argument to a JSON representation.
func (a BooleanArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V bool   `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

//UnmarshalJSON reads the entry from its JSON representation.
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

//BinaryArgument represents an argument that stores binary value.
type BinaryArgument struct {
	Value []byte
}

func NewBinaryArgument(b []byte) *BinaryArgument {
	return &BinaryArgument{b}
}

//GetValueType returns the type of value (Binary) stored in an argument.
func (a BinaryArgument) GetValueType() ArgumentValueType {
	return ArgumentBinary
}

func (a BinaryArgument) binarySize() int {
	return binaryArgumentMinLen + len(a.Value)
}

//MarshalBinary writes an argument to its byte representation.
func (a BinaryArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(ArgumentBinary)
	pos++
	PutBytesWithUInt32Len(buf[pos:], a.Value)
	return buf, nil
}

//Serialize argument to its byte representation.
func (a BinaryArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentBinary))
	if err != nil {
		return err
	}
	return s.BytesWithUInt32Len(a.Value)
}

//UnmarshalBinary reads an argument from a binary representation.
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

//MarshalJSON converts an argument to its JSON representation. Note that BASE64 is used to represent the binary value.
func (a BinaryArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V Script `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

//UnmarshalJSON converts JSON to a BinaryArgument structure. Value should be stored as BASE64 sting in JSON.
func (a *BinaryArgument) UnmarshalJSON(value []byte) error {
	tmp := struct {
		T string `json:"type"`
		V Script `json:"value"`
	}{}
	if err := json.Unmarshal(value, &tmp); err != nil {
		return errors.Wrap(err, "failed to deserialize binary data entry from JSON")
	}
	a.Value = tmp.V
	return nil
}

//StringArgument structure is an argument that store a string value.
type StringArgument struct {
	Value string
}

func NewStringArgument(s string) *StringArgument {
	return &StringArgument{s}
}

//GetValueType returns the type of value of the argument.
func (a StringArgument) GetValueType() ArgumentValueType {
	return ArgumentString
}

func (a StringArgument) binarySize() int {
	return stringArgumentMinLen + len(a.Value)
}

//MarshalBinary converts the argument to its byte representation.
func (a StringArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(ArgumentString)
	pos++
	PutStringWithUInt32Len(buf[pos:], a.Value)
	return buf, nil
}

//Serialize argument to its byte representation.
func (a StringArgument) Serialize(s *serializer.Serializer) error {
	err := s.Byte(byte(ArgumentString))
	if err != nil {
		return err
	}
	return s.StringWithUInt32Len(a.Value)
}

//UnmarshalBinary reads an StringArgument structure from bytes.
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

//MarshalJSON writes the entry to its JSON representation.
func (a StringArgument) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		T string `json:"type"`
		V string `json:"value"`
	}{a.GetValueType().String(), a.Value})
}

//UnmarshalJSON reads the entry from JSON.
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

// FunctionCall structure represents the description of function called in the InvokeScript transaction.
type FunctionCall struct {
	Default   bool
	Name      string
	Arguments Arguments
}

func (c FunctionCall) Serialize(s *serializer.Serializer) error {
	if c.Default {
		return s.Byte(0)
	}
	err := s.Bytes([]byte{1, reader.E_FUNCALL, reader.FH_USER})
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
	buf := make([]byte, c.binarySize())
	buf[0] = 1
	buf[1] = reader.E_FUNCALL
	buf[2] = reader.FH_USER
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
	if data[0] != reader.E_FUNCALL {
		return errors.Errorf("unexpected expression type %d, expected E_FUNCALL", data[0])
	}
	if data[1] != reader.FH_USER {
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

//MarshalJSON writes the entry to its JSON representation.
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

//UnmarshalJSON reads the entry from JSON.
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

func (c FunctionCall) binarySize() int {
	if c.Default {
		return 1
	}
	return 1 + 1 + 1 + 4 + len(c.Name) + c.Arguments.binarySize()
}

type ScriptResult struct {
	Transfers TransferSet
	Writes    WriteSet
}

func (sr *ScriptResult) MarshalWithAddresses() ([]byte, error) {
	transfersBytes, err := sr.Transfers.MarshalWithAddresses()
	if err != nil {
		return nil, err
	}
	writesBytes, err := sr.Writes.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, len(transfersBytes)+len(writesBytes)+8)
	pos := 0
	transfersSize := uint32(len(transfersBytes))
	binary.BigEndian.PutUint32(res[pos:], transfersSize)
	pos += 4
	copy(res[pos:], transfersBytes)
	pos += len(transfersBytes)
	writesSize := uint32(len(writesBytes))
	binary.BigEndian.PutUint32(res[pos:], writesSize)
	pos += 4
	copy(res[pos:], writesBytes)
	return res, nil
}

func (sr *ScriptResult) UnmarshalWithAddresses(data []byte) error {
	pos := 4
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	transfersSize := binary.BigEndian.Uint32(data[:pos])
	pos += int(transfersSize)
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	var ts TransferSet
	if err := ts.UnmarshalWithAddresses(data[4:pos]); err != nil {
		return err
	}
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	writesSize := binary.BigEndian.Uint32(data[pos:])
	pos += 4
	if len(data) < pos {
		return errors.New("invalid data size")
	}
	var ws WriteSet
	if err := ws.UnmarshalBinary(data[pos:]); err != nil {
		return err
	}
	pos += int(writesSize)
	if pos != len(data) {
		return errors.New("invalid data size")
	}
	sr.Transfers = ts
	sr.Writes = ws
	return nil
}

func (sr *ScriptResult) Valid() error {
	if err := sr.Transfers.Valid(); err != nil {
		return err
	}
	if err := sr.Writes.Valid(); err != nil {
		return err
	}
	return nil
}

func (sr *ScriptResult) ToProtobuf() (*g.InvokeScriptResult, error) {
	transfers, err := sr.Transfers.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return &g.InvokeScriptResult{
		Data:      sr.Writes.ToProtobuf(),
		Transfers: transfers,
	}, nil
}

type TransferSet []ScriptResultTransfer

func (ts *TransferSet) binarySize() int {
	totalSize := 0
	for _, tr := range *ts {
		totalSize += tr.binarySize()
	}
	return totalSize
}

func (ts *TransferSet) MarshalWithAddresses() ([]byte, error) {
	res := make([]byte, ts.binarySize())
	pos := 0
	for _, tr := range *ts {
		trBytes, err := tr.MarshalWithAddress()
		if err != nil {
			return nil, err
		}
		if pos+len(trBytes) > len(res) {
			return nil, errors.New("invalid data size")
		}
		copy(res[pos:], trBytes)
		pos += len(trBytes)
	}
	return res, nil
}

func (ts *TransferSet) UnmarshalWithAddresses(data []byte) error {
	pos := 0
	for pos < len(data) {
		var tr ScriptResultTransfer
		if err := tr.UnmarshalWithAddress(data[pos:]); err != nil {
			return err
		}
		pos += tr.binarySize()
		*ts = append(*ts, tr)
	}
	return nil
}

func (ts *TransferSet) Valid() error {
	if len(*ts) > maxInvokeTransfers {
		return errors.Errorf("transfer set of size %d is greater than allowed maximum of %d\n", len(*ts), maxInvokeTransfers)
	}
	for _, tr := range *ts {
		if tr.Amount < 0 {
			return errors.New("transfer amount is < 0")
		}
	}
	return nil
}

func (ts *TransferSet) ToProtobuf() ([]*g.InvokeScriptResult_Payment, error) {
	res := make([]*g.InvokeScriptResult_Payment, len(*ts))
	var err error
	for i, tr := range *ts {
		res[i], err = tr.ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

type WriteSet []DataEntry

func (ws *WriteSet) binarySize() int {
	totalSize := 0
	for _, entry := range *ws {
		totalSize += entry.binarySize()
	}
	return totalSize
}

func (ws *WriteSet) MarshalBinary() ([]byte, error) {
	res := make([]byte, ws.binarySize())
	pos := 0
	for _, entry := range *ws {
		entryBytes, err := entry.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if pos+len(entryBytes) > len(res) {
			return nil, errors.New("invalid data size")
		}
		copy(res[pos:], entryBytes)
		pos += len(entryBytes)
	}
	return res, nil
}

func (ws *WriteSet) UnmarshalBinary(data []byte) error {
	pos := 0
	for pos < len(data) {
		entry, err := NewDataEntryFromBytes(data[pos:])
		if err != nil {
			return err
		}
		pos += entry.binarySize()
		*ws = append(*ws, entry)
	}
	return nil
}

func (ws *WriteSet) Valid() error {
	if len(*ws) > maxInvokeWrites {
		return errors.Errorf("write set of size %d is greater than allowed maximum of %d\n", len(*ws), maxInvokeWrites)
	}
	totalSize := 0
	for _, entry := range *ws {
		if len(utf16.Encode([]rune(entry.GetKey()))) > maxInvokeWriteKeySizeInBytes {
			return errors.New("key is too large")
		}
		totalSize += entry.binarySize()
	}
	if totalSize > maxWriteSetSizeInBytes {
		return errors.Errorf("total write set size %d is greater than maximum %d\n", totalSize, maxWriteSetSizeInBytes)
	}
	return nil
}

func (ws *WriteSet) ToProtobuf() []*g.DataTransactionData_DataEntry {
	res := make([]*g.DataTransactionData_DataEntry, len(*ws))
	for i, entry := range *ws {
		res[i] = entry.ToProtobuf()
	}
	return res
}

type FullScriptTransfer struct {
	ScriptResultTransfer
	Sender    Address
	Timestamp uint64
	ID        *crypto.Digest
}

func NewFullScriptTransfer(scheme byte, tr *ScriptResultTransfer, tx *InvokeScriptV1) (*FullScriptTransfer, error) {
	return &FullScriptTransfer{
		ScriptResultTransfer: *tr,
		Sender:               *tx.ScriptRecipient.Address,
		Timestamp:            tx.Timestamp,
		ID:                   tx.ID,
	}, nil
}

type ScriptResultTransfer struct {
	Recipient Recipient
	Amount    int64
	Asset     OptionalAsset
}

func (tr *ScriptResultTransfer) binarySize() int {
	return AddressSize + 8 + tr.Asset.binarySize()
}

func (tr *ScriptResultTransfer) MarshalWithAddress() ([]byte, error) {
	if tr.Recipient.Address == nil {
		return nil, errors.New("can't marshal Recipient with no address set")
	}
	recipientBytes := tr.Recipient.Address.Bytes()
	if len(recipientBytes) != AddressSize {
		return nil, errors.New("invalid address size")
	}
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, uint64(tr.Amount))
	assetBytes, err := tr.Asset.MarshalBinary()
	if err != nil {
		return nil, err
	}
	res := make([]byte, tr.binarySize())
	copy(res, amountBytes)
	copy(res[len(amountBytes):], assetBytes)
	copy(res[len(amountBytes)+len(assetBytes):], recipientBytes)
	return res, nil
}

func (tr *ScriptResultTransfer) UnmarshalWithAddress(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid data size")
	}
	tr.Amount = int64(binary.BigEndian.Uint64(data[:8]))
	var asset OptionalAsset
	if err := asset.UnmarshalBinary(data[8:]); err != nil {
		return err
	}
	tr.Asset = asset
	pos := 8 + asset.binarySize()
	addr, err := NewAddressFromBytes(data[pos:])
	if err != nil {
		return err
	}
	tr.Recipient = NewRecipientFromAddress(addr)
	return nil
}

func (tr *ScriptResultTransfer) ToProtobuf() (*g.InvokeScriptResult_Payment, error) {
	if tr.Recipient.Address == nil {
		return nil, errors.New("script transfer has alias recipient, protobuf needs address")
	}
	addrBody, err := tr.Recipient.Address.Body()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get address body")
	}
	return &g.InvokeScriptResult_Payment{
		Amount:  &g.Amount{AssetId: tr.Asset.ToID(), Amount: tr.Amount},
		Address: addrBody,
	}, nil
}

type ScriptPayment struct {
	Amount uint64        `json:"amount"`
	Asset  OptionalAsset `json:"assetId"`
}

func (p ScriptPayment) MarshalBinary() ([]byte, error) {
	size := p.binarySize()
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
	size := p.binarySize()
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

func (p *ScriptPayment) binarySize() int {
	return 2 + 8 + p.Asset.binarySize()
}

type ScriptPayments []ScriptPayment

func (sps *ScriptPayments) Append(sp ScriptPayment) {
	*sps = append(*sps, sp)
}

func (sps ScriptPayments) MarshalBinary() ([]byte, error) {
	buf := make([]byte, sps.binarySize())
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
		data = data[sp.binarySize():]
	}
	return nil
}

func (sps ScriptPayments) binarySize() int {
	s := 2
	for _, p := range sps {
		s += p.binarySize()
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

func (b *FullWavesBalance) ToProtobuf() *g.BalanceResponse_WavesBalances {
	return &g.BalanceResponse_WavesBalances{
		Regular:    int64(b.Regular),
		Generating: int64(b.Generating),
		Available:  int64(b.Available),
		Effective:  int64(b.Effective),
		LeaseIn:    int64(b.LeaseIn),
		LeaseOut:   int64(b.LeaseOut),
	}
}

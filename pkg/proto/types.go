package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

const (
	//WavesAssetName is the default name for basic WAVES asset.
	WavesAssetName       = "WAVES"
	quotedWavesAssetName = "\"" + WavesAssetName + "\""
	orderLen             = crypto.PublicKeySize + crypto.PublicKeySize + 1 + 1 + 1 + 8 + 8 + 8 + 8 + 8
	orderV2FixedBodyLen  = 1 + orderLen
	orderV1MinLen        = crypto.SignatureSize + orderLen
	orderV2MinLen        = orderV2FixedBodyLen + proofsMinLen
	jsonNull             = "null"
	integerArgumentLen   = 1 + 8
	booleanArgumentLen   = 1 + 1
	binaryArgumentMinLen = 1 + 4
	stringArgumentMinLen = 1 + 4
	PriceConstant        = 100000000
	MaxOrderAmount       = 100 * PriceConstant * PriceConstant
	MaxOrderTTL          = uint64((30 * 24 * time.Hour) / time.Millisecond)
	maxKeySize           = 100
	maxValueSize         = 32767
)

type Timestamp = uint64
type Schema = byte
type Height = uint64

var jsonNullBytes = []byte{0x6e, 0x75, 0x6c, 0x6c}

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

func NewOptionalAssetFromDigest(d crypto.Digest) (*OptionalAsset, error) {
	return &OptionalAsset{Present: true, ID: d}, nil
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
	copy(buf[1:], a.ID[:])
	return buf, nil
}

//WriteTo writes its binary representation.
func (a OptionalAsset) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	err := s.Bool(a.Present)
	if err != nil {
		return 0, err
	}
	if a.Present {
		err = s.Bytes(a.ID[:])
		if err != nil {
			return 0, err
		}
	}
	return s.N(), nil
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

type OrderVersion struct {
	Version byte `json:"version"`
}

type Order interface {
	GetVersion() byte
	GetOrderType() OrderType
	GetMatcherPK() crypto.PublicKey
	GetAssetPair() AssetPair
	GetPrice() uint64
	GetExpiration() uint64
	Valid() (bool, error)
}

func OrderToOrderBody(o Order) (OrderBody, error) {
	switch o.GetVersion() {
	case 1:
		o, ok := o.(OrderV1)
		if !ok {
			return OrderBody{}, errors.New("failed to cast an order version 1 to OrderV1")
		}
		return o.OrderBody, nil
	case 2:
		o, ok := o.(OrderV2)
		if !ok {
			return OrderBody{}, errors.New("failed to cast an order version 2 to OrderV2")
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

func (o OrderBody) Valid() (bool, error) {
	if o.AssetPair.AmountAsset == o.AssetPair.PriceAsset {
		return false, errors.New("invalid asset pair")
	}
	if o.Price <= 0 {
		return false, errors.New("price should be positive")
	}
	if !validJVMLong(o.Price) {
		return false, errors.New("price is too big")
	}
	if o.Amount <= 0 {
		return false, errors.New("amount should be positive")
	}
	if !validJVMLong(o.Amount) {
		return false, errors.New("amount is too big")
	}
	if o.Amount > MaxOrderAmount {
		return false, errors.New("amount is larger than maximum allowed")
	}
	if o.MatcherFee <= 0 {
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
	if s <= 0 {
		return false, errors.New("spend amount should be positive")
	}
	if !o.SpendAsset().Present && !validJVMLong(s+o.MatcherFee) {
		return false, errors.New("sum of spend asset amount and matcher fee overflows JVM long")
	}
	r, err := o.ReceiveAmount(o.Amount, o.Price)
	if err != nil {
		return false, err
	}
	if r <= 0 {
		return false, errors.New("receive amount should be positive")
	}
	if o.Timestamp <= 0 {
		return false, errors.New("timestamp should be positive")
	}
	if o.Expiration <= 0 {
		return false, errors.New("expiration should be positive")
	}
	return true, nil
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

func (o *OrderBody) unmarshalBinary(data []byte) error {
	if l := len(data); l < orderLen {
		return errors.Errorf("not enough data for OrderBody, expected not less then %d, received %d", orderLen, l)
	}
	copy(o.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(o.MatcherPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	err = o.AssetPair.AmountAsset.UnmarshalBinary(data)
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

func (o OrderV1) GetVersion() byte {
	return 1
}

func (o OrderV1) GetOrderType() OrderType {
	return o.OrderType
}

func (o OrderV1) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o OrderV1) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o OrderV1) GetPrice() uint64 {
	return o.Price
}

func (o OrderV1) GetExpiration() uint64 {
	return o.Expiration
}

func (o *OrderV1) bodyMarshalBinary() ([]byte, error) {
	return o.OrderBody.marshalBinary()
}

func (o *OrderV1) bodyUnmarshalBinary(data []byte) error {
	return o.OrderBody.unmarshalBinary(data)
}

//Sign adds a signature to the order.
func (o *OrderV1) Sign(secretKey crypto.SecretKey) error {
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign OrderV1")
	}
	s := crypto.Sign(secretKey, b)
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
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV1")
	}
	return crypto.Verify(publicKey, *o.Signature, b), nil
}

//MarshalBinary writes order to its bytes representation.
func (o *OrderV1) MarshalBinary() ([]byte, error) {
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal OrderV1 to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], o.Signature[:])
	return buf, nil
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

func (o OrderV2) GetVersion() byte {
	return o.Version
}

func (o OrderV2) GetOrderType() OrderType {
	return o.OrderType
}

func (o OrderV2) GetMatcherPK() crypto.PublicKey {
	return o.MatcherPK
}

func (o OrderV2) GetAssetPair() AssetPair {
	return o.AssetPair
}

func (o OrderV2) GetPrice() uint64 {
	return o.Price
}

func (o OrderV2) GetExpiration() uint64 {
	return o.Expiration
}

func (o *OrderV2) bodyMarshalBinary() ([]byte, error) {
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
		return nil, errors.Wrap(err, "failed to marshal OrderV1 to bytes")
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
	b, err := o.bodyMarshalBinary()
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
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of OrderV2")
	}
	return o.Proofs.Verify(0, publicKey, b)
}

//MarshalBinary writes order to its bytes representation.
func (o *OrderV2) MarshalBinary() ([]byte, error) {
	bb, err := o.bodyMarshalBinary()
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
		s := crypto.Sign(key, data)
		p.Proofs = append(p.Proofs[:pos], append([]B58Bytes{s[:]}, p.Proofs[pos:]...)...)
	} else {
		pr := p.Proofs[pos]
		if len(pr) > 0 {
			return errors.Errorf("unable to overwrite non-empty proof at position %d", pos)
		}
		s := crypto.Sign(key, data)
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
type ValueType byte

// String translates ValueType value to human readable name.
func (vt ValueType) String() string {
	switch vt {
	case Integer:
		return "integer"
	case Boolean:
		return "boolean"
	case Binary:
		return "binary"
	case String:
		return "string"
	default:
		return ""
	}
}

//Supported value types.
const (
	Integer ValueType = iota
	Boolean
	Binary
	String
)

//DataEntry is a common interface of all types of data entries.
//The interface is used to store different types of data entries in one slice.
type DataEntry interface {
	GetKey() string
	GetValueType() ValueType
	MarshalBinary() ([]byte, error)
	Valid() (bool, error)
	binarySize() int
}

//IntegerDataEntry stores int64 value.
type IntegerDataEntry struct {
	Key   string
	Value int64
}

func (e IntegerDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(e.Key) > maxKeySize {
		return false, errors.New("key is too large")
	}
	return true, nil
}

//GetKey returns the key of data entry.
func (e IntegerDataEntry) GetKey() string {
	return e.Key
}

//GetValueType returns the value type of the entry.
func (e IntegerDataEntry) GetValueType() ValueType {
	return Integer
}

func (e IntegerDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 8
}

//MarshalBinary marshals the integer data entry in its bytes representation.
func (e IntegerDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	buf[pos] = byte(Integer)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(e.Value))
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
	if t := data[kl]; t != byte(Integer) {
		return errors.Errorf("unexpected value type %d for IntegerDataEntry, expected %d", t, Integer)
	}
	e.Value = int64(binary.BigEndian.Uint64(data[kl+1:]))
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

func (e BooleanDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(e.Key) > maxKeySize {
		return false, errors.New("key is too large")
	}
	return true, nil
}

//GetKey returns the key of data entry.
func (e BooleanDataEntry) GetKey() string {
	return e.Key
}

//GetValueType returns the data type (Boolean) of the entry.
func (e BooleanDataEntry) GetValueType() ValueType {
	return Boolean
}

func (e BooleanDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 1
}

//MarshalBinary writes a byte representation of the boolean data entry.
func (e BooleanDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	buf[pos] = byte(Boolean)
	pos++
	PutBool(buf[pos:], e.Value)
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
	if t := data[kl]; t != byte(Boolean) {
		return errors.Errorf("unexpected value type %d for BooleanDataEntry, expected %d", t, Boolean)
	}
	v, err := Bool(data[kl+1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BooleanDataEntry from bytes")
	}
	e.Value = v
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

func (e BinaryDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(e.Key) > maxKeySize {
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

//GetValueType returns the type of value (Binary) stored in an entry.
func (e BinaryDataEntry) GetValueType() ValueType {
	return Binary
}

func (e BinaryDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

//MarshalBinary writes an entry to its byte representation.
func (e BinaryDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	buf[pos] = byte(Binary)
	pos++
	PutBytesWithUInt16Len(buf[pos:], e.Value)
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
	if t := data[kl]; t != byte(Binary) {
		return errors.Errorf("unexpected value type %d for BinaryDataEntry, expected %d", t, Binary)
	}
	v, err := BytesWithUInt16Len(data[kl+1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BinaryDataEntry from bytes")
	}
	e.Value = v
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

func (e StringDataEntry) Valid() (bool, error) {
	if len(e.Key) == 0 {
		return false, errors.New("empty entry key")
	}
	if len(e.Key) > maxKeySize {
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

//GetValueType returns the type of value in key-value entry.
func (e StringDataEntry) GetValueType() ValueType {
	return String
}

func (e StringDataEntry) binarySize() int {
	return 2 + len(e.Key) + 1 + 2 + len(e.Value)
}

//MarshalBinary converts the data entry to its byte representation.
func (e StringDataEntry) MarshalBinary() ([]byte, error) {
	buf := make([]byte, e.binarySize())
	pos := 0
	PutStringWithUInt16Len(buf[pos:], e.Key)
	pos += 2 + len(e.Key)
	buf[pos] = byte(String)
	pos++
	PutStringWithUInt16Len(buf[pos:], e.Value)
	return buf, nil
}

//UnmarshalBinary reads an StringDataEntry structure from bytes.
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
	if t := data[kl]; t != byte(String) {
		return errors.Errorf("unexpected value type %d for StringDataEntry, expected %d", t, String)
	}
	v, err := StringWithUInt16Len(data[kl+1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StringDataEntry from bytes")
	}
	e.Value = v
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

type Script []byte

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
	GetValueType() ValueType
	MarshalBinary() ([]byte, error)
	binarySize() int
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

func (a *Arguments) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 4 {
		return errors.Errorf("%d is not enough bytes for Arguments", l)
	}
	n := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	for i := 0; i < int(n); i++ {
		var arg Argument
		var err error
		switch ValueType(data[0]) {
		case Integer:
			var ia IntegerArgument
			err = ia.UnmarshalBinary(data)
			arg = &ia
		case Boolean:
			var ba BooleanArgument
			err = ba.UnmarshalBinary(data)
			arg = &ba
		case Binary:
			var ba BinaryArgument
			err = ba.UnmarshalBinary(data)
			arg = &ba
		case String:
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

//GetValueType returns the value type of the entry.
func (a IntegerArgument) GetValueType() ValueType {
	return Integer
}

func (a IntegerArgument) binarySize() int {
	return integerArgumentLen
}

//MarshalBinary marshals the integer argument in its bytes representation.
func (a IntegerArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(Integer)
	pos++
	binary.BigEndian.PutUint64(buf[pos:], uint64(a.Value))
	return buf, nil
}

//UnmarshalBinary reads binary representation of integer argument to the structure.
func (a *IntegerArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < integerArgumentLen {
		return errors.Errorf("invalid data length for IntegerArgument, expected not less than %d, received %d", integerArgumentLen, l)
	}
	if t := data[0]; t != byte(Integer) {
		return errors.Errorf("unexpected value type %d for IntegerArgument, expected %d", t, Integer)
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

//GetValueType returns the data type (Boolean) of the argument.
func (a BooleanArgument) GetValueType() ValueType {
	return Boolean
}

func (a BooleanArgument) binarySize() int {
	return booleanArgumentLen
}

//MarshalBinary writes a byte representation of the boolean data entry.
func (a BooleanArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(Boolean)
	pos++
	PutBool(buf[pos:], a.Value)
	return buf, nil
}

//UnmarshalBinary reads a byte representation of the data entry.
func (a *BooleanArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < booleanArgumentLen {
		return errors.Errorf("invalid data length for BooleanArgument, expected not less than %d, received %d", booleanArgumentLen, l)
	}
	if t := data[0]; t != byte(Boolean) {
		return errors.Errorf("unexpected value type %d for BooleanArgument, expected %d", t, Boolean)
	}
	v, err := Bool(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal BooleanArgument from bytes")
	}
	a.Value = v
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

//GetValueType returns the type of value (Binary) stored in an argument.
func (a BinaryArgument) GetValueType() ValueType {
	return Binary
}

func (a BinaryArgument) binarySize() int {
	return binaryArgumentMinLen + len(a.Value)
}

//MarshalBinary writes an argument to its byte representation.
func (a BinaryArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(Binary)
	pos++
	PutBytesWithUInt32Len(buf[pos:], a.Value)
	return buf, nil
}

//UnmarshalBinary reads an argument from a binary representation.
func (a *BinaryArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < binaryArgumentMinLen {
		return errors.Errorf("invalid data length for BinaryArgument, expected not less than %d, received %d", binaryArgumentMinLen, l)
	}
	if t := data[0]; t != byte(Binary) {
		return errors.Errorf("unexpected value type %d for BinaryArgument, expected %d", t, Binary)
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

//GetValueType returns the type of value of the argument.
func (a StringArgument) GetValueType() ValueType {
	return String
}

func (a StringArgument) binarySize() int {
	return stringArgumentMinLen + len(a.Value)
}

//MarshalBinary converts the argument to its byte representation.
func (a StringArgument) MarshalBinary() ([]byte, error) {
	buf := make([]byte, a.binarySize())
	pos := 0
	buf[pos] = byte(String)
	pos++
	PutStringWithUInt32Len(buf[pos:], a.Value)
	return buf, nil
}

//UnmarshalBinary reads an StringArgument structure from bytes.
func (a *StringArgument) UnmarshalBinary(data []byte) error {
	if l := len(data); l < stringArgumentMinLen {
		return errors.Errorf("invalid data length for StringArgument, expected not less than %d, received %d", stringArgumentMinLen, l)
	}
	if t := data[0]; t != byte(String) {
		return errors.Errorf("unexpected value type %d for StringArgument, expected %d", t, String)
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
	Name      string    `json:"function"`
	Arguments Arguments `json:"args"`
}

func (c FunctionCall) MarshalBinary() ([]byte, error) {
	buf := make([]byte, c.binarySize())
	buf[0] = reader.E_FUNCALL
	buf[1] = reader.FH_USER
	PutStringWithUInt32Len(buf[2:], c.Name)
	ab, err := c.Arguments.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal FunctionCall to bytes")
	}
	copy(buf[2+4+len(c.Name):], ab)
	return buf, nil
}

func (c *FunctionCall) UnmarshalBinary(data []byte) error {
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

func (c FunctionCall) binarySize() int {
	return 1 + 1 + 4 + len(c.Name) + c.Arguments.binarySize()
}

type ScriptPayment struct {
	Amount uint64        `json:"amount"`
	Asset  OptionalAsset `json:"assetId"`
}

func (p ScriptPayment) MarshalBinary() ([]byte, error) {
	buf := make([]byte, p.binarySize())
	ab, err := p.Asset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize ScriptPayment to bytes")
	}
	binary.BigEndian.PutUint64(buf, p.Amount)
	copy(buf[8:], ab)
	return buf, nil
}

func (p *ScriptPayment) UnmarshalBinary(data []byte) error {
	if l := len(data); l < 8+1 {
		return errors.Errorf("%d is not enough bytes for ScriptPayment", l)
	}
	p.Amount = binary.BigEndian.Uint64(data[:8])
	var a OptionalAsset
	err := a.UnmarshalBinary(data[8:])
	if err != nil {
		return errors.Wrap(err, "failed to deserialize ScriptPayment from bytes")
	}
	p.Asset = a
	return nil
}

func (p *ScriptPayment) binarySize() int {
	return p.Asset.binarySize() + 8
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

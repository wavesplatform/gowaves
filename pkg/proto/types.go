package proto

import (
	"encoding/binary"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"strconv"
	"strings"
)

const (
	WavesAssetName       = "WAVES"
	QuotedWavesAssetName = "\"" + WavesAssetName + "\""

	orderFixedBodyLen = crypto.PublicKeySize + crypto.PublicKeySize + 1 + 1 + 1 + 8 + 8 + 8 + 8 + 8
	orderMinLen       = crypto.SignatureSize + orderFixedBodyLen
)

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
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		errors.Wrap(err, "failed to unmarshal B58Bytes from JSON")
	}
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode B58Bytes")
	}
	*b = B58Bytes(v)
	return nil
}

// OptionalAsset represents an optional asset identification
type OptionalAsset struct {
	Present bool
	ID      crypto.Digest
}

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
	return []byte("null"), nil
}

// UnmarshalJSON reads OptionalAsset from a JSON string Value
func (a *OptionalAsset) UnmarshalJSON(value []byte) error {
	s := strings.ToUpper(string(value))
	switch s {
	case "NULL", QuotedWavesAssetName:
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

func (a *OptionalAsset) MarshalBinary() ([]byte, error) {
	s := 1
	if a.Present {
		s += crypto.DigestSize
	}
	buf := make([]byte, s)
	PutBool(buf, a.Present)
	copy(buf[1:], a.ID[:])
	return buf, nil
}

func (a *OptionalAsset) UnmarshalBinary(data []byte) error {
	var err error
	a.Present, err = Bool(data)
	if err != nil {
		errors.Wrap(err, "failed to unmarshal OptionalAsset")
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

type Attachment string

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
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Attachment from JSON")
	}
	v, err := base58.Decode(s)
	if err != nil {
		return errors.Wrap(err, "failed to decode Attachment from JSON Value")
	}
	*a = Attachment(string(v))
	return nil
}

type Script struct {
	Version byte
	Body    []byte
}

type OptionalScript struct {
	MaybeScript *Script
}

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
	} else {
		return sellOrderName
	}
}

func (o OrderType) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(o.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (o *OrderType) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderType from JSON")
	}
	if l := strings.ToLower(s); l == buyOrderName {
		*o = Buy
	} else if l == sellOrderName {
		*o = Sell
	} else {
		return errors.Errorf("incorrect OrderType '%s'", s)
	}
	return nil
}

type AssetPair struct {
	AmountAsset OptionalAsset `json:"amountAsset"`
	PriceAsset  OptionalAsset `json:"priceAsset"`
}

type Order struct {
	ID         *crypto.Digest    `json:"id,omitempty"`
	Signature  *crypto.Signature `json:"signature,omitempty"`
	SenderPK   crypto.PublicKey  `json:"senderPublicKey"`
	MatcherPK  crypto.PublicKey  `json:"matcherPublicKey"`
	AssetPair  AssetPair         `json:"assetPair"`
	OrderType  OrderType         `json:"orderType"`
	Price      uint64            `json:"price"`
	Amount     uint64            `json:"amount"`
	Timestamp  uint64            `json:"timestamp"`
	Expiration uint64            `json:"expiration"`
	MatcherFee uint64            `json:"matcherFee"`
}

func NewUnsignedOrder(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) (*Order, error) {
	if price <= 0 {
		return nil, errors.New("price should be positive")
	}
	if amount <= 0 {
		return nil, errors.New("amount should be positive")
	}
	if matcherFee <= 0 {
		return nil, errors.New("matcher's fee should be positive")
	}
	//TODO: Add expiration validation
	return &Order{SenderPK: senderPK, MatcherPK: matcherPK, AssetPair: AssetPair{AmountAsset: amountAsset, PriceAsset: priceAsset}, OrderType: orderType, Price: price, Amount: amount, Timestamp: timestamp, Expiration: expiration, MatcherFee: matcherFee}, nil
}

func (o *Order) bodyMarshalBinary() ([]byte, error) {
	var p int
	aal := 0
	if o.AssetPair.AmountAsset.Present {
		aal += crypto.DigestSize
	}
	pal := 0
	if o.AssetPair.PriceAsset.Present {
		pal += crypto.DigestSize
	}
	buf := make([]byte, orderFixedBodyLen+aal+pal)
	copy(buf[0:], o.SenderPK[:])
	p += crypto.PublicKeySize
	copy(buf[p:], o.MatcherPK[:])
	p += crypto.PublicKeySize
	aa, err := o.AssetPair.AmountAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal Order to bytes")
	}
	copy(buf[p:], aa)
	p += 1 + aal
	pa, err := o.AssetPair.PriceAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal Order to bytes")
	}
	copy(buf[p:], pa)
	p += 1 + pal
	buf[p] = byte(o.OrderType)
	p += 1
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

func (o *Order) bodyUnmarshalBinary(data []byte) error {
	if l := len(data); l < orderFixedBodyLen {
		return errors.Errorf("not enough data for Order, expected not less then %d, received %d", orderFixedBodyLen, l)
	}
	copy(o.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(o.MatcherPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	err = o.AssetPair.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Order from bytes")
	}
	data = data[1:]
	if o.AssetPair.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = o.AssetPair.PriceAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Order from bytes")
	}
	data = data[1:]
	if o.AssetPair.PriceAsset.Present {
		data = data[crypto.DigestSize:]
	}
	o.OrderType = OrderType(data[0])
	if o.OrderType > 1 {
		return errors.Errorf("incorrect Order type %d", o.OrderType)
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

func (o *Order) Sign(secretKey crypto.SecretKey) error {
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to sign Order")
	}
	s := crypto.Sign(secretKey, b)
	o.Signature = &s
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign Order")
	}
	o.ID = &d
	return nil
}

func (o *Order) Verify(publicKey crypto.PublicKey) (bool, error) {
	if o.Signature == nil {
		return false, errors.New("empty signature")
	}
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "failed to verify signature of Order")
	}
	return crypto.Verify(publicKey, *o.Signature, b), nil
}

func (o *Order) MarshalBinary() ([]byte, error) {
	b, err := o.bodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Order to bytes")
	}
	bl := len(b)
	buf := make([]byte, bl+crypto.SignatureSize)
	copy(buf[0:], b)
	copy(buf[bl:], o.Signature[:])
	return buf, nil
}

func (o *Order) UnmarshalBinary(data []byte) error {
	if l := len(data); l < orderMinLen {
		return errors.Errorf("not enough data for Order, expected not less then %d, received %d", orderMinLen, l)
	}
	var bl int
	err := o.bodyUnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Order")
	}
	bl += orderFixedBodyLen
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
		return errors.Wrap(err, "failed to unmarshal TransferV1 transaction")
	}
	o.ID = &d
	return nil
}

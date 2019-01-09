package proto

import (
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
	if s == jsonNull {
		return nil
	}
	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal B58Bytes from JSON")
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

//MarshalBinary marshals the optional asset to its binary representation.
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

//UnmarshalBinary reads the OptionalAsset structure from its binary representation.
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

type Order interface {
	GetVersion() byte
	GetOrderType() OrderType
	GetMatcherPK() crypto.PublicKey
}

type order struct {
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

func newOrder(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) (*order, error) {
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
	return &order{SenderPK: senderPK, MatcherPK: matcherPK, AssetPair: AssetPair{AmountAsset: amountAsset, PriceAsset: priceAsset}, OrderType: orderType, Price: price, Amount: amount, Timestamp: timestamp, Expiration: expiration, MatcherFee: matcherFee}, nil
}

func (o *order) marshalBinary() ([]byte, error) {
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
		return nil, errors.Wrapf(err, "failed marshal order to bytes")
	}
	copy(buf[p:], aa)
	p += 1 + aal
	pa, err := o.AssetPair.PriceAsset.MarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshal order to bytes")
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

func (o *order) unmarshalBinary(data []byte) error {
	if l := len(data); l < orderLen {
		return errors.Errorf("not enough data for order, expected not less then %d, received %d", orderLen, l)
	}
	copy(o.SenderPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(o.MatcherPK[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	var err error
	err = o.AssetPair.AmountAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal order from bytes")
	}
	data = data[1:]
	if o.AssetPair.AmountAsset.Present {
		data = data[crypto.DigestSize:]
	}
	err = o.AssetPair.PriceAsset.UnmarshalBinary(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal order from bytes")
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
	order
}

//NewUnsignedOrderV1 creates the new unsigned order.
func NewUnsignedOrderV1(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) (*OrderV1, error) {
	o, err := newOrder(senderPK, matcherPK, amountAsset, priceAsset, orderType, price, amount, timestamp, expiration, matcherFee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OrderV1")
	}
	return &OrderV1{order: *o}, nil
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

func (o *OrderV1) bodyMarshalBinary() ([]byte, error) {
	return o.order.marshalBinary()
}

func (o *OrderV1) bodyUnmarshalBinary(data []byte) error {
	return o.order.unmarshalBinary(data)
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
	order
}

//NewUnsignedOrderV2 creates the new unsigned order.
func NewUnsignedOrderV2(senderPK, matcherPK crypto.PublicKey, amountAsset, priceAsset OptionalAsset, orderType OrderType, price, amount, timestamp, expiration, matcherFee uint64) (*OrderV2, error) {
	o, err := newOrder(senderPK, matcherPK, amountAsset, priceAsset, orderType, price, amount, timestamp, expiration, matcherFee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OrderV2")
	}
	return &OrderV2{Version: 2, order: *o}, nil
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
	b, err := o.order.marshalBinary()
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
	var oo order
	err := oo.unmarshalBinary(data[1:])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal OrderV2 from bytes")
	}
	o.order = oo
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
	p.Proofs = tmp
	return nil
}

//MarshalBinary writes the proofs to its binary form.
func (p *ProofsV1) MarshalBinary() ([]byte, error) {
	pl := 0
	for _, e := range p.Proofs {
		pl += len(e) + 2
	}
	buf := make([]byte, proofsMinLen+pl)
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
	binarySize() int
}

//IntegerDataEntry stores int64 value.
type IntegerDataEntry struct {
	Key   string
	Value int64
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
		V []byte `json:"value"`
	}{e.Key, e.GetValueType().String(), e.Value})
}

//UnmarshalJSON converts JSON to a BinaryDataEntry structure. Value should be stored as BASE64 sting in JSON.
func (e *BinaryDataEntry) UnmarshalJSON(value []byte) error {
	tmp := struct {
		K string `json:"key"`
		T string `json:"type"`
		V []byte `json:"value"`
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

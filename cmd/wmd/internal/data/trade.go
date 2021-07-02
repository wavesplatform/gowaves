package data

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	TradeSize = 1 + 3*crypto.DigestSize + 3*crypto.PublicKeySize + 8 + 8 + 8
)

type Trade struct {
	AmountAsset   crypto.Digest
	PriceAsset    crypto.Digest
	TransactionID crypto.Digest
	OrderType     proto.OrderType
	Buyer         proto.WavesAddress
	Seller        proto.WavesAddress
	Matcher       proto.WavesAddress
	Price         uint64
	Amount        uint64
	Timestamp     uint64
}

func NewTradeFromExchangeWithSig(scheme byte, tx *proto.ExchangeWithSig) (Trade, error) {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to convert ExchangeWithSig to Trade") }
	orderType := 1
	if tx.Order1.Timestamp > tx.Order2.Timestamp {
		orderType = 0
	}
	b, err := proto.NewAddressFromPublicKey(scheme, tx.Order1.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	s, err := proto.NewAddressFromPublicKey(scheme, tx.Order2.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	m, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	return Trade{
		AmountAsset:   tx.Order1.AssetPair.AmountAsset.ID,
		PriceAsset:    tx.Order1.AssetPair.PriceAsset.ID,
		TransactionID: *tx.ID,
		OrderType:     proto.OrderType(orderType),
		Buyer:         b,
		Seller:        s,
		Matcher:       m,
		Price:         tx.Price,
		Amount:        tx.Amount,
		Timestamp:     tx.Timestamp,
	}, nil
}

func NewTradeFromExchangeWithProofs(scheme byte, tx *proto.ExchangeWithProofs) (Trade, error) {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to convert ExchangeWithProofs to Trade") }
	var buyTS, sellTS uint64
	var buyer, seller, matcher proto.WavesAddress
	var err error
	var amountAsset, priceAsset crypto.Digest

	bo, err := tx.GetBuyOrder()
	if err != nil {
		return Trade{}, wrapError(err)
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return Trade{}, wrapError(err)
	}
	ap, pk, buyTS, err := extractOrderParameters(bo)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	buyer, err = proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return Trade{}, wrapError(err)
	}

	ap, pk, sellTS, err = extractOrderParameters(so)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	seller, err = proto.NewAddressFromPublicKey(scheme, pk)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	amountAsset = ap.AmountAsset.ID
	priceAsset = ap.PriceAsset.ID
	orderType := 1
	if buyTS > sellTS {
		orderType = 0
	}
	matcher, err = proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	return Trade{
		AmountAsset:   amountAsset,
		PriceAsset:    priceAsset,
		TransactionID: *tx.ID,
		OrderType:     proto.OrderType(orderType),
		Buyer:         buyer,
		Seller:        seller,
		Matcher:       matcher,
		Price:         tx.Price,
		Amount:        tx.Amount,
		Timestamp:     tx.Timestamp,
	}, nil
}

func (t *Trade) MarshalBinary() ([]byte, error) {
	buf := make([]byte, TradeSize)
	p := 0
	copy(buf[p:], t.AmountAsset[:])
	p += crypto.DigestSize
	copy(buf[p:], t.PriceAsset[:])
	p += crypto.DigestSize
	copy(buf[p:], t.TransactionID[:])
	p += crypto.DigestSize
	buf[p] = byte(t.OrderType)
	p++
	copy(buf[p:], t.Buyer[:])
	p += crypto.PublicKeySize
	copy(buf[p:], t.Seller[:])
	p += crypto.PublicKeySize
	copy(buf[p:], t.Matcher[:])
	p += crypto.PublicKeySize
	binary.BigEndian.PutUint64(buf[p:], t.Price)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], t.Amount)
	p += 8
	binary.BigEndian.PutUint64(buf[p:], t.Timestamp)
	return buf, nil
}

func (t *Trade) UnmarshalBinary(data []byte) error {
	if l := len(data); l < TradeSize {
		return errors.Errorf("%d bytes is not enough for Trade, expected %d", l, TradeSize)
	}
	copy(t.AmountAsset[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	copy(t.PriceAsset[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	copy(t.TransactionID[:], data[:crypto.DigestSize])
	data = data[crypto.DigestSize:]
	t.OrderType = proto.OrderType(data[0])
	data = data[1:]
	copy(t.Buyer[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(t.Seller[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	copy(t.Matcher[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	t.Price = binary.BigEndian.Uint64(data)
	data = data[8:]
	t.Amount = binary.BigEndian.Uint64(data)
	data = data[8:]
	t.Timestamp = binary.BigEndian.Uint64(data)
	return nil
}

// TradeInfo is an API representation of the Trade
type TradeInfo struct {
	Timestamp uint64             `json:"timestamp"`
	ID        crypto.Digest      `json:"id"`
	Confirmed bool               `json:"confirmed"`
	OrderType proto.OrderType    `json:"type"`
	Price     Decimal            `json:"price"`
	Amount    Decimal            `json:"amount"`
	Buyer     proto.WavesAddress `json:"buyer"`
	Seller    proto.WavesAddress `json:"seller"`
	Matcher   proto.WavesAddress `json:"matcher"`
}

func NewTradeInfo(trade Trade, amountAssetPrecision, priceAssetPrecision uint) TradeInfo {
	return TradeInfo{
		Timestamp: trade.Timestamp,
		ID:        trade.TransactionID,
		Confirmed: true,
		OrderType: trade.OrderType,
		Price:     *NewDecimal(trade.Price, 8+priceAssetPrecision-amountAssetPrecision), // decimalPrice * 10^(8 + priceAssetDecimals - amountAssetDecimals)
		Amount:    *NewDecimal(trade.Amount, amountAssetPrecision),
		Buyer:     trade.Buyer,
		Seller:    trade.Seller,
		Matcher:   trade.Matcher,
	}
}

type TradesByTimestampBackward []TradeInfo

func (a TradesByTimestampBackward) Len() int {
	return len(a)
}

func (a TradesByTimestampBackward) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a TradesByTimestampBackward) Less(i, j int) bool {
	return a[i].Timestamp > a[j].Timestamp
}

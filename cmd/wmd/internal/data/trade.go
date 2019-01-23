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
	Buyer         proto.Address
	Seller        proto.Address
	Matcher       proto.Address
	Price         uint64
	Amount        uint64
	Timestamp     uint64
}

func NewTradeFromExchangeV1(scheme byte, tx proto.ExchangeV1) (Trade, error) {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to convert ExchangeV1 to Trade") }
	orderType := 1
	if tx.BuyOrder.Timestamp > tx.SellOrder.Timestamp {
		orderType = 0
	}
	b, err := proto.NewAddressFromPublicKey(scheme, tx.BuyOrder.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	s, err := proto.NewAddressFromPublicKey(scheme, tx.SellOrder.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	m, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return Trade{}, wrapError(err)
	}
	return Trade{
		AmountAsset:   tx.BuyOrder.AssetPair.AmountAsset.ID,
		PriceAsset:    tx.BuyOrder.AssetPair.PriceAsset.ID,
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

func NewTradeFromExchangeV2(scheme byte, tx proto.ExchangeV2) (Trade, error) {
	wrapError := func(err error) error { return errors.Wrap(err, "failed to convert ExchangeV2 to Trade") }
	var buyTS, sellTS uint64
	var buyer, seller, matcher proto.Address
	var err error
	var amountAsset, priceAsset crypto.Digest
	switch tx.BuyOrder.GetVersion() {
	case 1:
		o, ok := tx.BuyOrder.(proto.OrderV1)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect BuyOrder version")
		}
		buyTS = o.Timestamp
		buyer, err = proto.NewAddressFromPublicKey(scheme, o.SenderPK)
		if err != nil {
			return Trade{}, wrapError(err)
		}
		amountAsset = o.AssetPair.AmountAsset.ID
		priceAsset = o.AssetPair.PriceAsset.ID
	case 2:
		o, ok := tx.BuyOrder.(proto.OrderV2)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect BuyOrder version")
		}
		buyTS = o.Timestamp
		buyer, err = proto.NewAddressFromPublicKey(scheme, o.SenderPK)
		if err != nil {
			return Trade{}, wrapError(err)
		}
		amountAsset = o.AssetPair.AmountAsset.ID
		priceAsset = o.AssetPair.PriceAsset.ID
	default:
		return Trade{}, errors.New("unsupported version of BuyOrder")
	}
	switch tx.SellOrder.GetVersion() {
	case 1:
		o, ok := tx.SellOrder.(proto.OrderV1)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect SellOrder version")
		}
		sellTS = o.Timestamp
		seller, err = proto.NewAddressFromPublicKey(scheme, o.SenderPK)
		if err != nil {
			return Trade{}, wrapError(err)
		}
	case 2:
		o, ok := tx.SellOrder.(proto.OrderV2)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect SellOrder version")
		}
		buyTS = o.Timestamp
		seller, err = proto.NewAddressFromPublicKey(scheme, o.SenderPK)
		if err != nil {
			return Trade{}, wrapError(err)
		}
	default:
		return Trade{}, errors.New("unsupported version of SellOrder")
	}
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
	Timestamp uint64          `json:"timestamp"`
	ID        crypto.Digest   `json:"id"`
	Confirmed bool            `json:"confirmed"`
	OrderType proto.OrderType `json:"type"`
	Price     Decimal         `json:"price"`
	Amount    Decimal         `json:"amount"`
	Buyer     proto.Address   `json:"buyer"`
	Seller    proto.Address   `json:"seller"`
	Matcher   proto.Address   `json:"matcher"`
}

func NewTradeInfo(trade Trade, amountAssetPrecision, priceAssetPrecision uint) TradeInfo {
	return TradeInfo{
		Timestamp: trade.Timestamp,
		ID:        trade.TransactionID,
		Confirmed: true,
		OrderType: trade.OrderType,
		Price:     *NewDecimal(trade.Price, priceAssetPrecision),
		Amount:    *NewDecimal(trade.Amount, amountAssetPrecision),
		Buyer:     trade.Buyer,
		Seller:    trade.Seller,
		Matcher:   trade.Matcher,
	}
}

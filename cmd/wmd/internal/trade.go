package internal

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
	AmountAsset   crypto.Digest    `json:"-"`
	PriceAsset    crypto.Digest    `json:"-"`
	TransactionID crypto.Digest    `json:"id"`
	OrderType     proto.OrderType  `json:"type"`
	Buyer         crypto.PublicKey `json:"buyer"`
	Seller        crypto.PublicKey `json:"seller"`
	Matcher       crypto.PublicKey `json:"matcher"`
	Price         uint64           `json:"price"`
	Amount        uint64           `json:"amount"`
	Timestamp     uint64           `json:"timestamp"`
}

func NewTradeFromExchangeV1(tx proto.ExchangeV1) Trade {
	orderType := 1
	if tx.BuyOrder.Timestamp > tx.SellOrder.Timestamp {
		orderType = 0
	}
	return Trade{
		AmountAsset:   tx.BuyOrder.AssetPair.AmountAsset.ID,
		PriceAsset:    tx.BuyOrder.AssetPair.PriceAsset.ID,
		TransactionID: *tx.ID,
		OrderType:     proto.OrderType(orderType),
		Buyer:         tx.BuyOrder.SenderPK,
		Seller:        tx.SellOrder.SenderPK,
		Matcher:       tx.SenderPK,
		Price:         tx.Price,
		Amount:        tx.Amount,
		Timestamp:     tx.Timestamp,
	}
}

func NewTradeFromExchangeV2(tx proto.ExchangeV2) (Trade, error) {
	var buyTS, sellTS uint64
	var buySender, sellSender crypto.PublicKey
	var amountAsset, priceAsset crypto.Digest

	switch tx.BuyOrder.GetVersion() {
	case 1:
		o, ok := tx.BuyOrder.(proto.OrderV1)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect BuyOrder version")
		}
		buyTS = o.Timestamp
		buySender = o.SenderPK
		amountAsset = o.AssetPair.AmountAsset.ID
		priceAsset = o.AssetPair.PriceAsset.ID
	case 2:
		o, ok := tx.BuyOrder.(proto.OrderV2)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect BuyOrder version")
		}
		buyTS = o.Timestamp
		buySender = o.SenderPK
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
		sellSender = o.SenderPK
	case 2:
		o, ok := tx.SellOrder.(proto.OrderV2)
		if !ok {
			return Trade{}, errors.New("failed to create Trade from ExchangeV2, incorrect SellOrder version")
		}
		buyTS = o.Timestamp
		sellSender = o.SenderPK
	default:
		return Trade{}, errors.New("unsupported version of SellOrder")
	}

	orderType := 1
	if buyTS > sellTS {
		orderType = 0
	}

	return Trade{
		AmountAsset:   amountAsset,
		PriceAsset:    priceAsset,
		TransactionID: *tx.ID,
		OrderType:     proto.OrderType(orderType),
		Buyer:         buySender,
		Seller:        sellSender,
		Matcher:       tx.SenderPK,
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

func NewTradeInfo(trade Trade, scheme byte, amountAssetPrecision, priceAssetPrecision uint) (*TradeInfo, error) {
	b, err := proto.NewAddressFromPublicKey(scheme, trade.Buyer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make an Address of the Buyer")
	}
	s, err := proto.NewAddressFromPublicKey(scheme, trade.Seller)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make an Address of the Seller")
	}
	m, err := proto.NewAddressFromPublicKey(scheme, trade.Matcher)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make an Address of the Matcher")
	}
	return &TradeInfo{
		Timestamp: trade.Timestamp,
		ID:        trade.TransactionID,
		Confirmed: true,
		OrderType: trade.OrderType,
		Price:     *NewDecimal(trade.Price, priceAssetPrecision),
		Amount:    *NewDecimal(trade.Amount, amountAssetPrecision),
		Buyer:     b,
		Seller:    s,
		Matcher:   m,
	}, nil
}

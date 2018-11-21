package internal

import (
	"encoding/binary"
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

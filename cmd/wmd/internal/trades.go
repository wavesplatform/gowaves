package internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

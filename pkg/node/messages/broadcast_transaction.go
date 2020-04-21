package messages

import "github.com/wavesplatform/gowaves/pkg/proto"

type BroadcastTransaction struct {
	Response    chan error
	Transaction proto.Transaction
}

func (*BroadcastTransaction) Internal() {
}

func NewBroadcastTransaction(response chan error, transaction proto.Transaction) *BroadcastTransaction {
	return &BroadcastTransaction{Response: response, Transaction: transaction}
}

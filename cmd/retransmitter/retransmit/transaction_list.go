package retransmit

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const idSize = 16

// transactions cache
type TransactionList struct {
	index int
	size  int
	lst   []*proto.Transaction
	id2t  map[[idSize]byte]struct{}
}

func NewTransactionList(size int) *TransactionList {
	return &TransactionList{
		size:  size,
		lst:   make([]*proto.Transaction, size),
		index: 0,
		id2t:  make(map[[idSize]byte]struct{}),
	}
}

func (a *TransactionList) Add(transaction proto.Transaction) {
	if a.Exists(transaction) {
		return
	}

	b := [idSize]byte{}
	copy(b[:], transaction.GetID())
	a.id2t[b] = struct{}{}
	a.clearOldTransaction(transaction)
}

func (a *TransactionList) clearOldTransaction(transaction proto.Transaction) {
	curIdx := a.index % a.size
	curTransaction := a.lst[curIdx]
	if curTransaction != nil {
		b := [idSize]byte{}
		copy(b[:], (*curTransaction).GetID())
		delete(a.id2t, b)
	}
	a.lst[curIdx] = &transaction
	a.index += 1
}

func (a *TransactionList) Exists(transaction proto.Transaction) bool {
	b := [idSize]byte{}
	copy(b[:], transaction.GetID())
	_, ok := a.id2t[b]
	return ok
}

func (a *TransactionList) Len() int {
	return len(a.id2t)
}

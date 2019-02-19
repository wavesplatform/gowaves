package retransmit

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const idSize = 16

// transactions cache
type TransactionList struct {
	index int
	size  int
	lst   [][idSize]byte
	id2t  map[[idSize]byte]struct{}
}

func NewTransactionList(size int) *TransactionList {
	return &TransactionList{
		size:  size,
		lst:   make([][idSize]byte, size),
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
	a.replaceOldTransaction(transaction)
}

func (a *TransactionList) replaceOldTransaction(transaction proto.Transaction) {
	curIdx := a.index % a.size
	curTransaction := a.lst[curIdx]
	delete(a.id2t, curTransaction)
	copy(a.lst[curIdx][:], transaction.GetID())
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

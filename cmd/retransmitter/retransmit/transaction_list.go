package retransmit

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const idSize = 16

// transactions cache
type TransactionList struct {
	index  int
	size   int
	lst    [][idSize]byte
	id2t   map[[idSize]byte]struct{}
	mu     sync.RWMutex
	scheme proto.Scheme
}

func NewTransactionList(size int, scheme proto.Scheme) *TransactionList {
	return &TransactionList{
		size:   size,
		lst:    make([][idSize]byte, size),
		index:  0,
		id2t:   make(map[[idSize]byte]struct{}),
		scheme: scheme,
	}
}

func (a *TransactionList) Add(transaction proto.Transaction) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.exists(transaction) {
		return
	}
	b := [idSize]byte{}
	// TODO: check GetID() error.
	txID, _ := transaction.GetID(a.scheme)
	copy(b[:], txID)
	a.id2t[b] = struct{}{}
	a.replaceOldTransaction(transaction)
}

// non thread safe
func (a *TransactionList) replaceOldTransaction(transaction proto.Transaction) {
	curIdx := a.index % a.size
	curTransaction := a.lst[curIdx]
	delete(a.id2t, curTransaction)
	// TODO: check GetID() error.
	txID, _ := transaction.GetID(a.scheme)
	copy(a.lst[curIdx][:], txID)
	a.index += 1
}

func (a *TransactionList) Exists(transaction proto.Transaction) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.exists(transaction)
}

// non thread safe
func (a *TransactionList) exists(transaction proto.Transaction) bool {
	b := [idSize]byte{}
	// TODO: check GetID() error.
	txID, _ := transaction.GetID(a.scheme)
	copy(b[:], txID)
	_, ok := a.id2t[b]
	return ok
}

func (a *TransactionList) Len() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.id2t)
}

package utxpool

import (
	"container/heap"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionWithBytes struct {
	T proto.Transaction
	B []byte
}

type transactionsHeap []*TransactionWithBytes

func (a transactionsHeap) Len() int { return len(a) }

func (a transactionsHeap) Less(i, j int) bool {
	// skip division by zero, check it when we add transaction
	return a[i].T.GetFee()/uint64(len(a[i].B)) > a[j].T.GetFee()/uint64(len(a[j].B))
}

func (a transactionsHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *transactionsHeap) Push(x interface{}) {
	item := x.(*TransactionWithBytes)
	*a = append(*a, item)
}

func (a *transactionsHeap) Pop() interface{} {
	old := *a
	n := len(old)
	item := old[n-1]
	*a = old[0 : n-1]
	return item
}

type Utx struct {
	mu             sync.Mutex
	transactions   transactionsHeap
	transactionIds map[crypto.Digest]struct{}
	limit          uint // max transaction count
}

func New(limit uint) *Utx {
	return &Utx{
		transactionIds: make(map[crypto.Digest]struct{}),
		limit:          limit,
	}
}

func (a *Utx) AddWithBytes(t proto.Transaction, b []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	tb := &TransactionWithBytes{
		T: t,
		B: b,
	}
	if len(b) == 0 {
		return
	}
	heap.Push(&a.transactions, tb)
	t.GenerateID()
	a.transactionIds[makeDigest(t.GetID())] = struct{}{}
}

func makeDigest(b []byte, e error) crypto.Digest {
	d := crypto.Digest{}
	copy(d[:], b)
	return d
}

func (a *Utx) Exists(t proto.Transaction) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.transactionIds[makeDigest(t.GetID())]
	return ok
}

func (a *Utx) Pop() *TransactionWithBytes {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.transactions.Len() > 0 {
		tb := heap.Pop(&a.transactions).(*TransactionWithBytes)
		delete(a.transactionIds, makeDigest(tb.T.GetID()))
		return tb
	}
	return nil
}

func (a *Utx) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.transactions.Len()
}

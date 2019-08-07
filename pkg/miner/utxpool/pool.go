package utxpool

import (
	"container/heap"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type transactionsHeap []*types.TransactionWithBytes

func (a transactionsHeap) Len() int { return len(a) }

func (a transactionsHeap) Less(i, j int) bool {
	// skip division by zero, check it when we add transaction
	return a[i].T.GetFee()/uint64(len(a[i].B)) > a[j].T.GetFee()/uint64(len(a[j].B))
}

func (a transactionsHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *transactionsHeap) Push(x interface{}) {
	item := x.(*types.TransactionWithBytes)
	*a = append(*a, item)
}

func (a *transactionsHeap) Pop() interface{} {
	old := *a
	n := len(old)
	item := old[n-1]
	*a = old[0 : n-1]
	return item
}

type UtxImpl struct {
	mu             sync.Mutex
	transactions   transactionsHeap
	transactionIds map[crypto.Digest]struct{}
	sizeLimit      uint // max transaction size in bytes
	curSize        uint
}

func New(sizeLimit uint) *UtxImpl {
	return &UtxImpl{
		transactionIds: make(map[crypto.Digest]struct{}),
		sizeLimit:      sizeLimit,
	}
}

func (a *UtxImpl) AddWithBytes(t proto.Transaction, b []byte) (added bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// exceed limit
	if a.curSize+uint(len(b)) > a.sizeLimit {
		return
	}

	tb := &types.TransactionWithBytes{
		T: t,
		B: b,
	}
	if len(b) == 0 {
		return
	}
	if a.exists(t) {
		return
	}
	heap.Push(&a.transactions, tb)
	t.GenerateID()
	a.transactionIds[makeDigest(t.GetID())] = struct{}{}
	a.curSize += uint(len(b))
	added = true
	return
}

func makeDigest(b []byte, e error) crypto.Digest {
	d := crypto.Digest{}
	copy(d[:], b)
	return d
}

func (a *UtxImpl) exists(t proto.Transaction) bool {
	_, ok := a.transactionIds[makeDigest(t.GetID())]
	return ok
}

func (a *UtxImpl) Exists(t proto.Transaction) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.exists(t)
}

func (a *UtxImpl) Pop() *types.TransactionWithBytes {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.transactions.Len() > 0 {
		tb := heap.Pop(&a.transactions).(*types.TransactionWithBytes)
		delete(a.transactionIds, makeDigest(tb.T.GetID()))
		return tb
	}
	return nil
}

func (a *UtxImpl) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.transactions.Len()
}

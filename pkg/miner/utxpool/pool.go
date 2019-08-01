package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"sync"

	"container/heap"
)

type transactionsHeap []proto.Transaction

func (a transactionsHeap) Len() int { return len(a) }

// TODO we should compare by fee/len
func (a transactionsHeap) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return a[i].GetFee() > a[j].GetFee()
}

func (a transactionsHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *transactionsHeap) Push(x interface{}) {
	item := x.(proto.Transaction)
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

// TODO add limits
func (a *Utx) Add(t proto.Transaction) {
	a.mu.Lock()
	heap.Push(&a.transactions, t)
	t.GenerateID()
	a.transactionIds[makeDigest(t.GetID())] = struct{}{}
	a.mu.Unlock()
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

func (a *Utx) Pop() proto.Transaction {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.transactions.Len() > 0 {
		t := heap.Pop(&a.transactions).(proto.Transaction)
		delete(a.transactionIds, makeDigest(t.GetID()))
		return t
	}
	return nil
}

func (a *Utx) Map(f func([]proto.Transaction) []proto.Transaction) {

}

func (a *Utx) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.transactions.Len()
}

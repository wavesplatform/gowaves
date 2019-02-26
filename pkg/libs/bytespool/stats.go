package bytespool

import "sync"

type Stats struct {
	pool      *BytesPool
	putCalled uint64
	getCalled uint64
	mu        sync.Mutex
}

func NewStats(pool *BytesPool) *Stats {
	return &Stats{
		pool: pool,
	}
}

func (a *Stats) Get() []byte {
	a.mu.Lock()
	a.getCalled += 1
	a.mu.Unlock()
	return a.pool.Get()
}

func (a *Stats) Put(bts []byte) {
	a.mu.Lock()
	a.putCalled += 1
	a.mu.Unlock()
	a.pool.Put(bts)
}

func (a *Stats) Stat() (allocations, puts, gets uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pool.Allocations(), a.putCalled, a.getCalled
}

func (a *Stats) BytesLen() int {
	return a.pool.BytesLen()
}

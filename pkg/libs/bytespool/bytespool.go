package bytespool

import (
	"go.uber.org/zap"
	"sync"
)

type BytesPool struct {
	index       int
	poolSize    int
	bytesLen    int
	arr         [][]byte
	mu          sync.Mutex
	allocations uint64
	putCalled   uint64
	getCalled   uint64
}

// poolSize is the maximum number of elements can be stored in pool
// bytesLength is the size of byte array, like cap([]byte{})
func NewBytesPool(poolSize int, bytesLength int) *BytesPool {
	if poolSize < 1 {
		panic("poolSize should be positive")
	}
	if bytesLength < 1 {
		panic("bytesLen should be positive")
	}

	return &BytesPool{
		index:    -1,
		poolSize: poolSize,
		bytesLen: bytesLength,
		arr:      make([][]byte, poolSize),
	}
}

// Returns bytes from pool. If pool is empty or no free bytes available
// new bytes will be allocated.
// Notice, there is no memory zeroing or cleanup is made, bytes returns as they are
func (a *BytesPool) Get() []byte {
	a.mu.Lock()
	a.getCalled += 1
	// no more elements are free, we should allocate another bytes slice
	if a.index == -1 {
		out := a.alloc(a.bytesLen)
		a.mu.Unlock()
		return out
	}
	bts := a.arr[a.index]
	a.arr[a.index] = nil
	a.index--
	a.mu.Unlock()
	return bts
}

// Put bytes back to the pool.
// If the length of provided bytes not equal defined, just ignore them.
func (a *BytesPool) Put(bts []byte) {
	a.mu.Lock()
	a.putCalled += 1
	a.mu.Unlock()
	// something unexpected passed
	if len(bts) != a.bytesLen {
		zap.S().Warnf("BytesPool Put expected bytesLen %d, passed %d", a.bytesLen, len(bts))
		return
	}

	a.mu.Lock()
	if a.index >= a.poolSize-1 {
		a.mu.Unlock()
		return
	}
	a.index++
	a.arr[a.index] = bts
	a.mu.Unlock()
}

// not thread safe
func (a *BytesPool) alloc(size int) []byte {
	a.allocations++
	return make([]byte, size)
}

func (a *BytesPool) Allocations() uint64 {
	a.mu.Lock()
	out := a.allocations
	a.mu.Unlock()
	return out
}

func (a *BytesPool) BytesLen() int {
	return a.bytesLen
}

func (a *BytesPool) Stat() (allocations, puts, gets uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.allocations, a.putCalled, a.getCalled
}

package bytespool

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const size = 2 * 1024 * 1024

func TestNewBytesPool(t *testing.T) {
	pool := NewBytesPool(32, size)
	assert.NotNil(t, pool)
}

func TestNewBytesPool_Panics(t *testing.T) {
	assert.PanicsWithValue(t, "poolSize should be positive", func() {
		NewBytesPool(-1, size)
	})
	assert.PanicsWithValue(t, "bytesLen should be positive", func() {
		NewBytesPool(1, 0)
	})
}

func TestBytesPool_Get(t *testing.T) {
	pool := NewBytesPool(32, size)

	bts := pool.Get()

	assert.Equal(t, size, len(bts))
}

// test than just no panics happens
func TestBytesPool_Put(t *testing.T) {
	pool := NewBytesPool(1, size)
	bts1 := make([]byte, size)
	pool.Put(bts1)
	bts2 := make([]byte, size)
	pool.Put(bts2)

	pool.Get()
}

func TestBytesPool_Get_Put(t *testing.T) {
	pool := NewBytesPool(32, size)
	bts1 := make([]byte, size)
	pool.Put(bts1)
	assert.EqualValues(t, 0, pool.Allocations())

	bts2 := make([]byte, size/2)
	pool.Put(bts2)
	assert.EqualValues(t, 0, pool.Allocations())

	pool.Get()
	assert.EqualValues(t, 0, pool.Allocations())

	pool.Get()
	assert.EqualValues(t, 1, pool.Allocations())
}

func TestBytesPool_Stat(t *testing.T) {
	pool := NewBytesPool(32, size)

	allocations, puts, gets := pool.Stat()
	assert.EqualValues(t, 0, allocations)
	assert.EqualValues(t, 0, puts)
	assert.EqualValues(t, 0, gets)

	pool.Put(pool.Get())
	allocations, puts, gets = pool.Stat()
	assert.EqualValues(t, 1, allocations)
	assert.EqualValues(t, 1, puts)
	assert.EqualValues(t, 1, gets)

	pool.Put(pool.Get())
	allocations, puts, gets = pool.Stat()
	assert.EqualValues(t, 1, allocations)
	assert.EqualValues(t, 2, puts)
	assert.EqualValues(t, 2, gets)
}

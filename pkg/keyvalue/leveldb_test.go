package keyvalue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	cacheSize           = 100
	writeBuffer         = 4 * 1024 * 1024
	sstableSize         = 2 * 1024 * 1024
	compactionTotalSize = 10 * 1024 * 1024
)

func TestKeyVal(t *testing.T) {
	dbDir := t.TempDir()
	params := KeyValParams{
		CacheParams:         CacheParams{cacheSize},
		BloomFilterParams:   BloomFilterParams{n, falsePositiveProbability, NoOpStore{}, false},
		WriteBuffer:         writeBuffer,
		CompactionTableSize: sstableSize,
		CompactionTotalSize: compactionTotalSize,
	}
	kv, err := NewKeyVal(dbDir, params)
	assert.NoError(t, err, "NewKeyVal() failed")

	t.Cleanup(func() {
		err = kv.Close()
		assert.NoError(t, err, "Close() failed")
	})

	// Test direct DB operations.
	keyPrefix := []byte("sampleKey")
	key0 := []byte("sampleKey0")
	val0 := []byte("sampleValue0")
	err = kv.Put(key0, val0)
	assert.NoError(t, err, "Put() failed")
	receivedVal, err := kv.Get(key0)
	assert.NoError(t, err, "Get() failed")
	assert.Equal(t, val0, receivedVal, "saved and retrieved values for same key differ")
	has, err := kv.Has(key0)
	assert.NoError(t, err, "Has() failed")
	assert.Equal(t, has, true, "Has() returned false for value that was saved before")
	err = kv.Delete(key0)
	assert.NoError(t, err, "Delete() failed")
	has, err = kv.Has(key0)
	assert.NoError(t, err, "Has() failed")
	assert.Equal(t, has, false, "Has() returned true for deleted value")
	// Test batch operations.
	key1 := []byte("sampleKey1")
	val1 := []byte("sampleValue1")
	batch, err := kv.NewBatch()
	assert.NoError(t, err, "NewBatch() failed")
	batch.Put(key0, val0)
	batch.Put(key1, val1)
	batch.Delete(key0)
	err = kv.Flush(batch)
	assert.NoError(t, err, "Flush() failed")
	receivedVal, err = kv.Get(key1)
	assert.NoError(t, err, "Get() failed")
	assert.Equal(t, val1, receivedVal, "saved and retrieved values for same key differ")
	has, err = kv.Has(key0)
	assert.NoError(t, err, "Has() failed")
	assert.Equal(t, has, false, "Has() returned true for value that was deleted from batch")

	// Add another key-value pair directly.
	err = kv.Put(key0, val0)
	assert.NoError(t, err)

	// Test iterator's Next().
	iter, err := kv.NewKeyIterator([]byte{})
	assert.NoError(t, err, "NewKeyIterator() failed")
	for iter.Next() {
		key := iter.Key()
		val := iter.Value()
		receivedVal, err = kv.Get(key)
		assert.NoError(t, err, "Get() failed")
		assert.Equal(t, val, receivedVal, "Invalid value in iterator")
	}
	iter.Release()
	err = iter.Error()
	assert.NoError(t, err, "iterator error")

	// Test iterator's First() / Last().
	iter, err = kv.NewKeyIterator(keyPrefix)
	assert.NoError(t, err, "NewKeyIterator() failed")
	moved := iter.Last()
	assert.Equal(t, true, moved)
	assert.Equal(t, key1, iter.Key())
	assert.Equal(t, val1, iter.Value())
	moved = iter.First()
	assert.Equal(t, true, moved)
	assert.Equal(t, key0, iter.Key())
	assert.Equal(t, val0, iter.Value())
	iter.Release()
	err = iter.Error()
	assert.NoError(t, err, "iterator error")
}

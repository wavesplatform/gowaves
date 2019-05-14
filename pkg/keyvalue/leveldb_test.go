package keyvalue

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyVal(t *testing.T) {
	dbDir, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	kv, err := NewKeyVal(dbDir, BloomFilterParams{n, falsePositiveProbability})
	assert.NoError(t, err, "NewKeyVal() failed")

	defer func() {
		err = os.RemoveAll(dbDir)
		assert.NoError(t, err, "os.RemoveAll() failed")
		err = kv.Close()
		assert.NoError(t, err, "Close() failed")
	}()

	// Test direct DB operations.
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
	assert.Equal(t, has, false, "Has() returned false for value that was deleted from batch")
	// Test iterator.
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
}

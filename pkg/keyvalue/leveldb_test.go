package keyvalue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/goleveldb/leveldb"
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

func newTestKeyVal(t *testing.T) *KeyVal {
	params := KeyValParams{
		CacheParams:         CacheParams{1024 * 1024},
		BloomFilterParams:   BloomFilterParams{n, falsePositiveProbability, NoOpStore{}, false},
		WriteBuffer:         writeBuffer,
		CompactionTableSize: sstableSize,
		CompactionTotalSize: compactionTotalSize,
	}
	kv, err := NewKeyVal(t.TempDir(), params)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, kv.Close()) })
	return kv
}

// requireStored checks the DB, the cache and the bloom filter for the key.
func requireStored(t *testing.T, kv *KeyVal, key, val []byte) {
	t.Helper()
	dbVal, err := kv.db.Get(key, nil)
	require.NoError(t, err)
	assert.Equal(t, val, dbVal, "db value for %q", key)
	cacheVal, err := kv.cache.Get(key)
	require.NoError(t, err, "cache value for %q", key)
	assert.Equal(t, val, cacheVal, "cache value for %q", key)
	notInTheSet, err := kv.filter.notInTheSet(key)
	require.NoError(t, err)
	assert.False(t, notInTheSet, "bloom filter misses %q", key)
	got, err := kv.Get(key)
	require.NoError(t, err)
	assert.Equal(t, val, got)
	has, err := kv.Has(key)
	require.NoError(t, err)
	assert.True(t, has)
}

func requireAbsent(t *testing.T, kv *KeyVal, key []byte) {
	t.Helper()
	_, err := kv.db.Get(key, nil)
	assert.ErrorIs(t, err, leveldb.ErrNotFound, "db has %q", key)
	_, err = kv.cache.Get(key)
	assert.Error(t, err, "cache has %q", key)
	_, err = kv.Get(key)
	assert.ErrorIs(t, err, ErrNotFound)
	has, err := kv.Has(key)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestBatchInterleavedPutDelete(t *testing.T) {
	kv := newTestKeyVal(t)
	// Pre-existing values that the batch overrides or removes.
	require.NoError(t, kv.Put([]byte("k-deleted"), []byte("old")))
	require.NoError(t, kv.Put([]byte("k-reput"), []byte("old")))

	b, err := kv.NewBatch()
	require.NoError(t, err)
	b.Put([]byte("k-put-del"), []byte("v1"))
	b.Delete([]byte("k-put-del"))
	b.Delete([]byte("k-del-put"))
	b.Put([]byte("k-del-put"), []byte("v2"))
	b.Put([]byte("k-overwrite"), []byte("first"))
	b.Put([]byte("k-overwrite"), []byte("second"))
	b.Delete([]byte("k-deleted"))
	b.Delete([]byte("k-reput"))
	b.Put([]byte("k-reput"), []byte("new"))
	b.Put([]byte("k-del-put-del"), []byte("v3"))
	b.Delete([]byte("k-del-put-del"))
	require.NoError(t, kv.Flush(b))

	requireAbsent(t, kv, []byte("k-put-del"))
	requireStored(t, kv, []byte("k-del-put"), []byte("v2"))
	requireStored(t, kv, []byte("k-overwrite"), []byte("second"))
	requireAbsent(t, kv, []byte("k-deleted"))
	requireStored(t, kv, []byte("k-reput"), []byte("new"))
	requireAbsent(t, kv, []byte("k-del-put-del"))

	// Flush empties the batch, so flushing it again changes nothing.
	require.NoError(t, kv.Put([]byte("k-put-del"), []byte("later")))
	require.NoError(t, kv.Flush(b))
	requireStored(t, kv, []byte("k-put-del"), []byte("later"))
}

func TestBatchCopiesCallerBuffers(t *testing.T) {
	kv := newTestKeyVal(t)
	b, err := kv.NewBatch()
	require.NoError(t, err)
	key := []byte("key-0")
	val := []byte("value-0")
	b.Put(key, val)
	delKey := []byte("del-0")
	b.Delete(delKey)
	// Reuse the caller buffers before the flush.
	copy(key, "key-1")
	copy(val, "value-1")
	b.Put(key, val)
	copy(key, "XXXXX")
	copy(val, "YYYYYYY")
	copy(delKey, "key-1")
	require.NoError(t, kv.Flush(b))

	requireStored(t, kv, []byte("key-0"), []byte("value-0"))
	requireStored(t, kv, []byte("key-1"), []byte("value-1"))
	requireAbsent(t, kv, []byte("XXXXX"))
}

func TestBatchReset(t *testing.T) {
	kv := newTestKeyVal(t)
	b, err := kv.NewBatch()
	require.NoError(t, err)
	lb, ok := b.(*batch)
	require.True(t, ok)
	b.Put([]byte("a"), []byte("1"))
	b.Delete([]byte("b"))
	b.Reset()
	assert.Zero(t, lb.lb.Len())
	require.NoError(t, kv.Flush(b))
	requireAbsent(t, kv, []byte("a"))

	// The batch is usable after a reset.
	b.Put([]byte("c"), []byte("3"))
	require.NoError(t, kv.Flush(b))
	requireStored(t, kv, []byte("c"), []byte("3"))
	assert.Zero(t, lb.lb.Len())
}

func TestGetDoesNotCacheOnError(t *testing.T) {
	kv := newTestKeyVal(t)
	key := []byte("missing")
	// Make the bloom filter pass the key so the DB is queried.
	require.NoError(t, kv.filter.add(key))
	_, err := kv.Get(key)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = kv.cache.Get(key)
	assert.Error(t, err, "nothing must be cached for a key that is not in the DB")
}

func TestGetDoesNotCacheOnDBError(t *testing.T) {
	params := KeyValParams{
		CacheParams:         CacheParams{1024 * 1024},
		BloomFilterParams:   BloomFilterParams{n, falsePositiveProbability, NoOpStore{}, false},
		WriteBuffer:         writeBuffer,
		CompactionTableSize: sstableSize,
		CompactionTotalSize: compactionTotalSize,
	}
	kv, err := NewKeyVal(t.TempDir(), params)
	require.NoError(t, err)
	key := []byte("key")
	require.NoError(t, kv.filter.add(key))
	require.NoError(t, kv.db.Close())
	_, err = kv.Get(key)
	require.ErrorIs(t, err, leveldb.ErrClosed)
	_, err = kv.cache.Get(key)
	assert.Error(t, err, "nothing must be cached when the DB returns an error")
}

func newUnclosedTestKeyVal(t *testing.T, path string) *KeyVal {
	params := KeyValParams{
		CacheParams:         CacheParams{1024 * 1024},
		BloomFilterParams:   BloomFilterParams{n, falsePositiveProbability, NoOpStore{}, false},
		WriteBuffer:         writeBuffer,
		CompactionTableSize: sstableSize,
		CompactionTotalSize: compactionTotalSize,
	}
	kv, err := NewKeyVal(path, params)
	require.NoError(t, err)
	return kv
}

func TestGetCachesValueFromDB(t *testing.T) {
	kv := newTestKeyVal(t)
	key, val := []byte("key"), []byte("value")
	require.NoError(t, kv.Put(key, val))
	kv.cache.Clear()
	got, err := kv.Get(key)
	require.NoError(t, err)
	assert.Equal(t, val, got)
	requireStored(t, kv, key, val)
}

func TestCacheSkipsOversizedValues(t *testing.T) {
	kv := newTestKeyVal(t)
	// The value is larger than the maximal entry size of the 1 MiB cache.
	big := make([]byte, 64*1024)
	for i := range big {
		big[i] = byte(i)
	}
	put, batched := []byte("put"), []byte("batched")
	require.NoError(t, kv.Put(put, []byte("small")))
	require.NoError(t, kv.Put(put, big))
	b, err := kv.NewBatch()
	require.NoError(t, err)
	b.Put(batched, []byte("small"))
	require.NoError(t, kv.Flush(b))
	b.Put(batched, big)
	require.NoError(t, kv.Flush(b))
	for _, key := range [][]byte{put, batched} {
		_, cErr := kv.cache.Get(key)
		require.Error(t, cErr, "the stale cached value of %q must be removed", key)
		got, gErr := kv.Get(key)
		require.NoError(t, gErr)
		assert.Equal(t, big, got)
	}
}

type foreignBatch struct{}

func (foreignBatch) Delete([]byte)      {}
func (foreignBatch) Put([]byte, []byte) {}
func (foreignBatch) Reset()             {}

func TestFlushRejectsForeignBatch(t *testing.T) {
	kv := newTestKeyVal(t)
	require.Error(t, kv.Flush(foreignBatch{}))
}

func TestWritesFailOnClosedDB(t *testing.T) {
	kv := newUnclosedTestKeyVal(t, t.TempDir())
	require.NoError(t, kv.db.Close())
	require.ErrorIs(t, kv.Put([]byte("key"), []byte("value")), leveldb.ErrClosed)
	b, err := kv.NewBatch()
	require.NoError(t, err)
	b.Put([]byte("key"), []byte("value"))
	require.ErrorIs(t, kv.Flush(b), leveldb.ErrClosed)
	_, err = kv.cache.Get([]byte("key"))
	assert.Error(t, err, "nothing must be cached when the DB write fails")
}

func TestReopenRebuildsBloomFilter(t *testing.T) {
	path := t.TempDir()
	kv := newUnclosedTestKeyVal(t, path)
	keys := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	for _, key := range keys {
		require.NoError(t, kv.Put(key, key))
	}
	require.NoError(t, kv.Close())

	// NoOpStore does not keep the filter, so it is rebuilt from the keys stored in the DB.
	kv = newUnclosedTestKeyVal(t, path)
	t.Cleanup(func() { require.NoError(t, kv.Close()) })
	for _, key := range keys {
		notInTheSet, err := kv.filter.notInTheSet(key)
		require.NoError(t, err)
		assert.False(t, notInTheSet, "bloom filter misses %q", key)
	}
	iter, err := kv.NewKeyIterator(nil)
	require.NoError(t, err)
	var got [][]byte
	for iter.Next() {
		got = append(got, append([]byte(nil), iter.Key()...))
	}
	iter.Release()
	require.NoError(t, iter.Error())
	assert.Equal(t, keys, got)
}

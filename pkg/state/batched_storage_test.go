package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	maxBatchSize   = 12000
	testRecordSize = 8
	prefix         = byte(1)

	size = 10000
)

var (
	key0 = []byte{1}
	key1 = []byte{2}
)

type batchedStorageTestObjects struct {
	stor        *testStorageObjects
	batchedStor *batchedStorage

	rollbackedIds map[crypto.Signature]bool
}

func createBatchedStorage() (*batchedStorageTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	params := &batchedStorParams{maxBatchSize: maxBatchSize, recordSize: testRecordSize, prefix: prefix}
	lock := stor.hs.stateDB.retrieveWriteLock()
	batchedStor := newBatchedStorage(stor.db, stor.dbBatch, lock, stor.hs.stateDB, params)
	return &batchedStorageTestObjects{
		stor:          stor,
		batchedStor:   batchedStor,
		rollbackedIds: make(map[crypto.Signature]bool),
	}, path, nil
}

func (to *batchedStorageTestObjects) addTestRecords(t *testing.T, key []byte, data []testRecord) {
	for _, rc := range data {
		to.stor.addBlock(t, rc.blockID)
		delete(to.rollbackedIds, rc.blockID)
		blockNum, err := to.stor.stateDB.blockIdToNum(rc.blockID)
		assert.NoError(t, err)
		err = to.batchedStor.addRecord(key, rc.record, blockNum)
		assert.NoError(t, err)
	}
}

func (to *batchedStorageTestObjects) testIterator(t *testing.T, key []byte, data []testRecord) {
	testIter := newTestIter(to, data)
	iter, err := to.batchedStor.newBackwardRecordIterator(key)
	assert.NoError(t, err)
	for iter.next() {
		record, err := iter.currentRecord()
		assert.NoError(t, err)
		valid, found := testIter.next()
		assert.Equal(t, true, found)
		assert.Equal(t, valid.record, record)
	}
	iter.release()
	err = iter.error()
	assert.NoError(t, err)
	_, found := testIter.next()
	assert.Equal(t, false, found)
}

func (to *batchedStorageTestObjects) rollbackBlock(t *testing.T, blockID crypto.Signature) {
	err := to.stor.stateDB.rollbackBlock(blockID)
	assert.NoError(t, err)
	to.rollbackedIds[blockID] = true
}

func (to *batchedStorageTestObjects) flush(t *testing.T) {
	err := to.batchedStor.flush()
	assert.NoError(t, err)
	to.batchedStor.reset()
	to.stor.flush(t)
}

type testRecord struct {
	blockID crypto.Signature
	record  []byte
}

type testIter struct {
	to   *batchedStorageTestObjects
	data []testRecord
}

func newTestIter(to *batchedStorageTestObjects, data []testRecord) *testIter {
	return &testIter{to: to, data: data}
}

func (it *testIter) next() (*testRecord, bool) {
	for {
		if len(it.data) == 0 {
			return nil, false
		}
		lastIndex := len(it.data) - 1
		last := it.data[lastIndex]
		it.data = it.data[:lastIndex]
		if _, ok := it.to.rollbackedIds[last.blockID]; !ok {
			return &last, true
		}
	}
}

func genTestRecords(t *testing.T, ids []crypto.Signature) []testRecord {
	res := make([]testRecord, len(ids))
	for i, id := range ids {
		record := make([]byte, testRecordSize)
		_, err := rand.Read(record)
		assert.NoError(t, err)
		res[i].blockID = id
		res[i].record = record
	}
	return res
}

func TestIterators(t *testing.T) {
	to, path, err := createBatchedStorage()
	assert.NoError(t, err, "createBatchedStorage() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	ids := genRandBlockIds(t, size)
	key0Records := genTestRecords(t, ids)
	key1Records := genTestRecords(t, ids)
	to.addTestRecords(t, key0, key0Records[:size/2])
	to.addTestRecords(t, key1, key1Records[:size/2])
	to.flush(t)
	to.testIterator(t, key0, key0Records[:size/2])
	to.testIterator(t, key1, key1Records[:size/2])

	to.addTestRecords(t, key0, key0Records[size/2:])
	to.addTestRecords(t, key1, key1Records[size/2:])
	to.flush(t)

	to.testIterator(t, key0, key0Records)
	to.testIterator(t, key1, key1Records)

	// Rollback.
	for _, id := range ids[size/2:] {
		to.rollbackBlock(t, id)
	}
	to.testIterator(t, key0, key0Records)
	to.testIterator(t, key1, key1Records)

	// Add again.
	to.addTestRecords(t, key0, key0Records[size/2:])
	to.addTestRecords(t, key1, key1Records[size/2:])
	to.flush(t)

	to.testIterator(t, key0, key0Records)
	to.testIterator(t, key1, key1Records)
}

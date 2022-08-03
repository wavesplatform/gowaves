package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	maxBatchSize   = 12000
	testRecordSize = 8
	prefix         = byte(1)

	size         = 10000
	testMemLimit = 10 * proto.MiB
)

var (
	key0 = []byte{1}
	key1 = []byte{2}
)

type batchedStorageTestObjects struct {
	stor        *testStorageObjects
	batchedStor *batchedStorage

	rollbackedIds map[proto.BlockID]bool
}

func createBatchedStorage(t *testing.T, recordSize int) *batchedStorageTestObjects {
	stor := createStorageObjects(t, true)
	params := &batchedStorParams{maxBatchSize: maxBatchSize, recordSize: recordSize, prefix: prefix}
	batchedStor, err := newBatchedStorage(stor.db, stor.hs.stateDB, params, testMemLimit, 1000, stor.hs.amend)
	require.NoError(t, err)
	return &batchedStorageTestObjects{
		stor:          stor,
		batchedStor:   batchedStor,
		rollbackedIds: make(map[proto.BlockID]bool),
	}
}

func (to *batchedStorageTestObjects) addTestRecords(t *testing.T, key []byte, data []testRecord) {
	for _, rc := range data {
		to.stor.addBlock(t, rc.blockID)
		delete(to.rollbackedIds, rc.blockID)
		blockNum, err := to.stor.stateDB.newestBlockIdToNum(rc.blockID)
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

func (to *batchedStorageTestObjects) rollbackBlock(t *testing.T, blockID proto.BlockID) {
	to.stor.rollbackBlock(t, blockID)
	to.rollbackedIds[blockID] = true
}

func (to *batchedStorageTestObjects) flush(t *testing.T) {
	err := to.batchedStor.flush()
	assert.NoError(t, err)
	to.batchedStor.reset()
	to.stor.flush(t)
}

type testRecord struct {
	blockID proto.BlockID
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

func genTestRecords(t *testing.T, ids []proto.BlockID) []testRecord {
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

func TestLastRecordByKeyWithRollback(t *testing.T) {
	to := createBatchedStorage(t, testRecordSize)

	ids := genRandBlockIds(t, size)
	key0Records := genTestRecords(t, ids)
	key1Records := genTestRecords(t, ids)
	to.addTestRecords(t, key0, key0Records)
	to.addTestRecords(t, key1, key1Records)
	to.flush(t)

	// Rollback.
	for _, id := range ids[size/2:] {
		to.rollbackBlock(t, id)
	}

	last, err := to.batchedStor.lastRecordByKey(key0)
	assert.NoError(t, err)
	assert.Equal(t, key0Records[size/2-1].record, last)
	last, err = to.batchedStor.newestLastRecordByKey(key0)
	assert.NoError(t, err)
	assert.Equal(t, key0Records[size/2-1].record, last)
}

func TestLastRecordByKey(t *testing.T) {
	to := createBatchedStorage(t, 3)

	to.stor.addBlock(t, blockID0)
	blockNum0, err := to.stor.stateDB.newestBlockIdToNum(blockID0)
	assert.NoError(t, err)
	key0 := []byte{1, 2, 3}
	record0 := []byte{4, 5, 6}
	err = to.batchedStor.addRecord(key0, record0, blockNum0)
	assert.NoError(t, err)
	res, err := to.batchedStor.newestLastRecordByKey(key0)
	assert.NoError(t, err)
	assert.Equal(t, record0, res)

	key1 := []byte{7, 8, 9}
	record1 := []byte{10, 11, 12}
	err = to.batchedStor.addRecord(key1, record1, blockNum0)
	assert.NoError(t, err)
	res, err = to.batchedStor.newestLastRecordByKey(key1)
	assert.NoError(t, err)
	assert.Equal(t, record1, res)

	to.flush(t)

	res, err = to.batchedStor.newestLastRecordByKey(key0)
	assert.NoError(t, err)
	assert.Equal(t, record0, res)
	res, err = to.batchedStor.newestLastRecordByKey(key1)
	assert.NoError(t, err)
	assert.Equal(t, record1, res)

	res, err = to.batchedStor.lastRecordByKey(key0)
	assert.NoError(t, err)
	assert.Equal(t, record0, res)
	res, err = to.batchedStor.lastRecordByKey(key1)
	assert.NoError(t, err)
	assert.Equal(t, record1, res)

	to.stor.addBlock(t, blockID1)
	blockNum1, err := to.stor.stateDB.newestBlockIdToNum(blockID1)
	assert.NoError(t, err)
	key2 := []byte{13, 14, 15}
	record2 := []byte{16, 17, 18}
	err = to.batchedStor.addRecord(key2, record2, blockNum1)
	assert.NoError(t, err)
	res, err = to.batchedStor.newestLastRecordByKey(key2)
	assert.NoError(t, err)
	assert.Equal(t, record2, res)

	to.flush(t)

	res, err = to.batchedStor.newestLastRecordByKey(key2)
	assert.NoError(t, err)
	assert.Equal(t, record2, res)
	res, err = to.batchedStor.lastRecordByKey(key2)
	assert.NoError(t, err)
	assert.Equal(t, record2, res)
}

func TestIterators(t *testing.T) {
	to := createBatchedStorage(t, testRecordSize)

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

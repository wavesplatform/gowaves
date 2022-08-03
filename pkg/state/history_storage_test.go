package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	keySize = 32
	valSize = 1024
)

func TestAddNewEntry(t *testing.T) {
	to := createStorageObjects(t, true)

	to.addBlock(t, blockID0)
	key := bytes.Repeat([]byte{0xff}, keySize)
	val := bytes.Repeat([]byte{0x1a}, valSize)
	err := to.hs.addNewEntry(accountScript, key, val, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	entry, err := to.hs.newestTopEntry(key)
	assert.NoError(t, err, "newestTopEntry() failed")
	assert.Equal(t, val, entry.data)
	data, err := to.hs.newestTopEntryData(key)
	assert.NoError(t, err, "newestTopEntryData() failed")
	assert.Equal(t, val, data)
	entries, err := to.hs.newestEntriesDataInHeightRange(key, 1, 1)
	assert.NoError(t, err, "newestEntriesDataInHeightRange() failed")
	assert.Equal(t, [][]byte{val}, entries)

	blockID, err := to.hs.newestBlockOfTheTopEntry(key)
	assert.NoError(t, err, "newestBlockOfTheTopEntry() failed")
	assert.Equal(t, blockID0, blockID)

	to.flush(t)

	entry, err = to.hs.topEntry(key)
	assert.NoError(t, err, "topEntry() failed")
	assert.Equal(t, val, entry.data)
	data, err = to.hs.topEntryData(key)
	assert.NoError(t, err, "topEntryData() failed")
	assert.Equal(t, val, data)
	blockID, err = to.hs.blockOfTheTopEntry(key)
	assert.NoError(t, err, "blockOfTheTopEntry() failed")
	assert.Equal(t, blockID0, blockID)

	// Check entryDataAtHeight().

	data, err = to.hs.entryDataAtHeight(key, 1)
	assert.NoError(t, err)
	assert.Equal(t, val, data)
	to.addBlock(t, blockID1)
	val2 := bytes.Repeat([]byte{0x2a}, valSize)
	to.flush(t)
	err = to.hs.addNewEntry(accountScript, key, val2, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)
	data, err = to.hs.entryDataAtHeight(key, 1)
	assert.NoError(t, err)
	assert.Equal(t, val, data)
	data, err = to.hs.entryDataAtHeight(key, 2)
	assert.NoError(t, err)
	assert.Equal(t, val2, data)
}

func TestNewestDataIterator(t *testing.T) {
	to := createStorageObjects(t, true)

	// Add some entries and flush.
	to.addBlock(t, blockID0)
	key0 := accountScriptKey{addr: testGlobal.senderInfo.addr.ID()}
	val0 := []byte{1, 2, 3}
	key1 := accountScriptKey{addr: testGlobal.minerInfo.addr.ID()}
	val1 := []byte{100}
	key2 := assetScriptKey{assetID: proto.AssetIDFromDigest(testGlobal.asset0.asset.ID)}
	val2 := []byte{88}
	err := to.hs.addNewEntry(accountScript, key0.bytes(), val0, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(accountScript, key1.bytes(), val1, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(assetScript, key2.bytes(), val2, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)
	// Add more entries after flush() and test iterator.
	to.addBlock(t, blockID1)
	key3 := accountScriptKey{addr: testGlobal.issuerInfo.addr.ID()}
	val3 := []byte{11, 12, 13}
	err = to.hs.addNewEntry(accountScript, key3.bytes(), val3, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	val4 := []byte{144, 169}
	err = to.hs.addNewEntry(accountScript, key0.bytes(), val4, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")

	// Test accountScript iterator.
	correctValues := map[string][]byte{
		string(key0.bytes()): val4,
		string(key3.bytes()): val3,
		string(key1.bytes()): val1,
	}
	keys := make(map[string]bool)
	iter, err := to.hs.newNewestTopEntryIterator(accountScript)
	assert.NoError(t, err)
	for iter.Next() {
		key := iter.Key()
		correctVal, ok := correctValues[string(key)]
		assert.Equal(t, true, ok)
		val := iter.Value()
		assert.Equal(t, correctVal, val)
		keys[string(key)] = true
	}
	assert.Equal(t, len(correctValues), len(keys))
	iter.Release()
	err = iter.Error()
	assert.NoError(t, err)

	// Test assetScript iterator.
	correctValues = map[string][]byte{
		string(key2.bytes()): val2,
	}
	iter, err = to.hs.newNewestTopEntryIterator(assetScript)
	assert.NoError(t, err)
	keys = make(map[string]bool)
	for iter.Next() {
		key := iter.Key()
		correctVal, ok := correctValues[string(key)]
		assert.Equal(t, true, ok)
		val := iter.Value()
		assert.Equal(t, correctVal, val)
		keys[string(key)] = true
	}
	assert.Equal(t, len(correctValues), len(keys))
	iter.Release()
	err = iter.Error()
	assert.NoError(t, err)
}

func TestVariableSizes(t *testing.T) {
	to := createStorageObjects(t, true)

	// Add some entries and flush.
	to.addBlock(t, blockID0)
	key1 := accountScriptKey{addr: testGlobal.senderInfo.addr.ID()}
	val11 := []byte{1, 2, 3}
	key2 := accountScriptKey{addr: testGlobal.minerInfo.addr.ID()}
	val21 := []byte{9, 8, 7, 6}
	err := to.hs.addNewEntry(accountScript, key1.bytes(), val11, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(accountScript, key2.bytes(), val21, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)

	to.addBlock(t, blockID1)
	val12 := []byte{4, 5, 6, 7}
	val22 := []byte{5, 4, 3, 2, 1}
	err = to.hs.addNewEntry(accountScript, key1.bytes(), val12, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(accountScript, key2.bytes(), val22, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)

	h1, err := to.hs.getHistory(key1.bytes(), true)
	assert.NoError(t, err)
	s1, err := h1.countTotalSize()
	assert.NoError(t, err)
	assert.Equal(t, 1+4+4+3+4+4+4, s1)
	h2, err := to.hs.getHistory(key2.bytes(), true)
	assert.NoError(t, err)
	s2, err := h2.countTotalSize()
	assert.NoError(t, err)
	assert.Equal(t, 1+4+4+4+4+4+5, s2)
}

func TestFixedRecordSizes(t *testing.T) {
	to := createStorageObjects(t, true)

	to.addBlock(t, blockID0)
	val1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	err := to.hs.addNewEntry(blockReward, blockRewardKeyBytes, val1, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)

	to.addBlock(t, blockID1)
	val2 := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	err = to.hs.addNewEntry(blockReward, blockRewardKeyBytes, val2, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)

	h, err := to.hs.getHistory(blockRewardKeyBytes, true)
	assert.NoError(t, err)
	s, err := h.countTotalSize()
	assert.NoError(t, err)
	assert.Equal(t, 1+blockRewardRecordSize+4+blockRewardRecordSize+4, s)
}

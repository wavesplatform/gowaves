package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	keySize = 32
	valSize = 1024
)

func TestAddNewEntry(t *testing.T) {
	to, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.addBlock(t, blockID0)
	key := bytes.Repeat([]byte{0xff}, keySize)
	val := bytes.Repeat([]byte{0x1a}, valSize)
	err = to.hs.addNewEntry(accountScript, key, val, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	entry, err := to.hs.freshLatestEntry(key, true)
	assert.NoError(t, err, "freshLatestEntry() failed")
	assert.Equal(t, val, entry.data)
	data, err := to.hs.freshLatestEntryData(key, true)
	assert.NoError(t, err, "freshLatestEntryData() failed")
	assert.Equal(t, val, data)
	entries, err := to.hs.entriesDataInHeightRange(key, 1, 1, true)
	assert.NoError(t, err, "entriesDataInHeightRange() failed")
	assert.Equal(t, [][]byte{val}, entries)

	blockID, err := to.hs.freshBlockOfTheLatestEntry(key, true)
	assert.NoError(t, err, "freshBlockOfTheLatestEntry() failed")
	assert.Equal(t, blockID0, blockID)

	to.flush(t)

	entry, err = to.hs.latestEntry(key, true)
	assert.NoError(t, err, "latestEntry() failed")
	assert.Equal(t, val, entry.data)
	data, err = to.hs.latestEntryData(key, true)
	assert.NoError(t, err, "latestEntryData() failed")
	assert.Equal(t, val, data)
	blockID, err = to.hs.blockOfTheLatestEntry(key, true)
	assert.NoError(t, err, "blockOfTheLatestEntry() failed")
	assert.Equal(t, blockID0, blockID)

	// Check entryDataAtHeight().

	data, err = to.hs.entryDataAtHeight(key, 1, true)
	assert.NoError(t, err)
	assert.Equal(t, val, data)
	to.addBlock(t, blockID1)
	val2 := bytes.Repeat([]byte{0x2a}, valSize)
	to.flush(t)
	err = to.hs.addNewEntry(accountScript, key, val2, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)
	data, err = to.hs.entryDataAtHeight(key, 1, true)
	assert.NoError(t, err)
	assert.Equal(t, val, data)
	data, err = to.hs.entryDataAtHeight(key, 2, true)
	assert.NoError(t, err)
	assert.Equal(t, val2, data)
}

func TestNewestDataIterator(t *testing.T) {
	to, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Add some entries and flush.
	to.addBlock(t, blockID0)
	key0 := accountScriptKey{addr: testGlobal.senderInfo.addr}
	val0 := []byte{1, 2, 3}
	key1 := accountScriptKey{addr: testGlobal.minerInfo.addr}
	val1 := []byte{100}
	key2 := assetScriptKey{asset: testGlobal.asset0.asset.ID}
	val2 := []byte{88}
	err = to.hs.addNewEntry(accountScript, key0.bytes(), val0, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(accountScript, key1.bytes(), val1, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	err = to.hs.addNewEntry(assetScript, key2.bytes(), val2, blockID0)
	assert.NoError(t, err, "addNewEntry() failed")
	to.flush(t)
	// Add more entries after flush() and test iterator.
	to.addBlock(t, blockID1)
	key3 := accountScriptKey{addr: testGlobal.issuerInfo.addr}
	val3 := []byte{11, 12, 13}
	err = to.hs.addNewEntry(accountScript, key3.bytes(), val3, blockID1)
	assert.NoError(t, err, "addNewEntry() failed")
	val4 := []byte{144, 169}
	err = to.hs.addNewEntry(accountScript, key0.bytes(), val4, blockID1)

	// Test accountScript iterator.
	correctValues := map[string][]byte{
		string(key0.bytes()): val4,
		string(key3.bytes()): val3,
		string(key1.bytes()): val1,
	}
	keys := make(map[string]bool)
	iter, err := newNewestDataIterator(to.hs, accountScript)
	assert.NoError(t, err)
	for iter.Next() {
		key := iter.Key()
		correctVal, ok := correctValues[string(key)]
		assert.Equal(t, true, ok)
		val, err := to.hs.freshLatestEntryData(key, true)
		assert.NoError(t, err)
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
	iter, err = newNewestDataIterator(to.hs, assetScript)
	assert.NoError(t, err)
	keys = make(map[string]bool)
	for iter.Next() {
		key := iter.Key()
		correctVal, ok := correctValues[string(key)]
		assert.Equal(t, true, ok)
		val, err := to.hs.freshLatestEntryData(key, true)
		assert.NoError(t, err)
		assert.Equal(t, correctVal, val)
		keys[string(key)] = true
	}
	assert.Equal(t, len(correctValues), len(keys))
	iter.Release()
	err = iter.Error()
	assert.NoError(t, err)
}

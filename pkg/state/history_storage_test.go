package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util"
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

		err = util.CleanTemporaryDirs(path)
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

	to.flush(t)

	entry, err = to.hs.latestEntry(key, true)
	assert.NoError(t, err, "latestEntry() failed")
	assert.Equal(t, val, entry.data)
	data, err = to.hs.latestEntryData(key, true)
	assert.NoError(t, err, "latestEntryData() failed")
	assert.Equal(t, val, data)
	blockID, err := to.hs.blockOfTheLatestEntry(key, true)
	assert.NoError(t, err, "blockOfTheLatestEntry() failed")
	assert.Equal(t, blockID0, blockID)
}

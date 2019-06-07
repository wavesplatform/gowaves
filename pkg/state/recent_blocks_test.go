package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	rangeSize = rollbackMaxBlocks
)

type idChecker = func(crypto.Signature) (uint64, error)

func checkHeights(t *testing.T, ids []crypto.Signature, properHeights []uint64, check idChecker) {
	for i, id := range ids {
		height, err := check(id)
		assert.NoError(t, err, "blockIDToHeight failed")
		assert.Equal(t, properHeights[i], height, "blockIDToHeight() returned incorrect result")
	}
}

func TestIsInRange(t *testing.T) {
	rw, path, err := createBlockReadWriter(8, 8)
	assert.NoError(t, err, "createBlockReadWriter() failed")
	rb, err := newRecentBlocks(rangeSize, rw)
	assert.NoError(t, err, "newRecentBlocks() failed")

	defer func() {
		err = rw.close()
		assert.NoError(t, err, "failed to close blockReadWriter")
		err = rw.db.Close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	ids := genBlockIds(t, rangeSize)
	heights := make([]uint64, rangeSize)
	// Test indirect addition of IDs.
	for i, id := range ids {
		err := rw.startBlock(id)
		assert.NoError(t, err, "rw.startBlock() failed")
		err = rw.finishBlock(id)
		assert.NoError(t, err, "rw.finishBlock() failed")
		err = rb.addNewBlockID(id)
		assert.NoError(t, err, "addNewBlockID() failed")
		heights[i] = uint64(i + 1)
	}
	checkHeights(t, ids, heights, rb.newBlockIDToHeight)
	rb.flush()
	checkHeights(t, ids, heights, rb.blockIDToHeight)
	err = rw.flush()
	assert.NoError(t, err, "rw.flush() failed")
	err = rw.db.Flush(rw.dbBatch)
	assert.NoError(t, err, "db.Flush() failed")
	// Now test direct addition of IDs.
	rb.reset()
	height, err := rb.height()
	assert.NoError(t, err, "rb.height() failed")
	assert.Equal(t, uint64(rangeSize+1), height, "height() returned incorrect result")
	checkHeights(t, ids, heights, rb.blockIDToHeight)
}

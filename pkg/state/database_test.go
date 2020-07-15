package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func TestAddBlock(t *testing.T) {
	to, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.addBlock(t, blockID0)
	blockNum, err := to.stateDB.newestBlockIdToNum(blockID0)
	assert.NoError(t, err, "blockIdToNum() failed")
	assert.Equal(t, uint32(0), blockNum)
	blockID, err := to.stateDB.newestBlockNumToId(blockNum)
	assert.NoError(t, err, "blockNumToId() failed")
	assert.Equal(t, blockID0, blockID)
	blockNum, err = to.stateDB.newestBlockNumByHeight(1)
	assert.NoError(t, err, "newestBlockNumByHeight() failed")
	assert.Equal(t, uint32(0), blockNum)
	isValid, err := to.stateDB.isValidBlock(blockNum)
	assert.NoError(t, err, "isValidBlock() failed")
	assert.Equal(t, false, isValid)

	to.flush(t)

	blockNum, err = to.stateDB.blockNumByHeight(1)
	assert.NoError(t, err, "blockNumByHeight() failed")
	assert.Equal(t, uint32(0), blockNum)
	isValid, err = to.stateDB.isValidBlock(blockNum)
	assert.NoError(t, err, "isValidBlock() failed")
	assert.Equal(t, true, isValid)
	height, err := to.stateDB.getHeight()
	assert.NoError(t, err, "getHeight() failed")
	assert.Equal(t, uint64(1), height)

	// Rollback.
	to.rollbackBlock(t, blockID0)
	isValid, err = to.stateDB.isValidBlock(blockNum)
	assert.NoError(t, err, "isValidBlock() failed")
	assert.Equal(t, false, isValid)
}

package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func TestSyncRw(t *testing.T) {
	to, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Add block.
	err = to.rw.startBlock(blockID0)
	assert.NoError(t, err, "startBlock() failed")
	err = to.rw.finishBlock(blockID0)
	assert.NoError(t, err, "finishBlock() failed")

	to.flush(t)

	rwHeight, err := to.rw.getHeight()
	assert.NoError(t, err, "getHeight() failed")
	assert.Equal(t, uint64(1), rwHeight)

	err = to.stateDB.syncRw()
	assert.NoError(t, err, "syncRw() failed")

	// Block that is not present in DB should be removed after sync.
	rwHeight, err = to.rw.getHeight()
	assert.NoError(t, err, "getHeight() failed")
	assert.Equal(t, uint64(0), rwHeight)
}

func TestAddBlock(t *testing.T) {
	to, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.addBlock(t, blockID0)
	blockNum, err := to.stateDB.blockIdToNum(blockID0)
	assert.NoError(t, err, "blockIdToNum() failed")
	assert.Equal(t, uint32(0), blockNum)
	blockID, err := to.stateDB.blockNumToId(blockNum)
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
	err = to.stateDB.rollbackBlock(blockID0)
	assert.NoError(t, err, "rollbackBlock() failed")
	isValid, err = to.stateDB.isValidBlock(blockNum)
	assert.NoError(t, err, "isValidBlock() failed")
	assert.Equal(t, false, isValid)
	height, err = to.stateDB.getHeight()
	assert.NoError(t, err, "getHeight() failed")
	assert.Equal(t, uint64(0), height)
}

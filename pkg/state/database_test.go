package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestAddBlock(t *testing.T) {
	to := createStorageObjects(t, true)

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

func addAndFlushBlocks(t *testing.T, to *testStorageObjects, n int) []proto.BlockID {
	ids := genRandBlockIDs(t, n)
	for _, id := range ids {
		to.addBlock(t, id)
	}
	to.flush(t)
	return ids
}

func cacheIsValid(c *blockNumCache) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid
}

func TestMinAcceptableBlockNumCache(t *testing.T) {
	to := createStorageObjects(t, true)
	ids := addAndFlushBlocks(t, to, 1000)

	num, err := to.stateDB.getMinAcceptableBlockNum()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), num) // Rollback min height is still 1.
	assert.True(t, cacheIsValid(&to.stateDB.minAcceptableBlockNum))

	// Reset invalidates the cache.
	to.stateDB.reset()
	assert.False(t, cacheIsValid(&to.stateDB.minAcceptableBlockNum))
	num, err = to.stateDB.getMinAcceptableBlockNum()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), num)

	// Height change moves the rollback min height, the cache must follow it.
	ids = append(ids, addAndFlushBlocks(t, to, 1500)...)
	rollbackMinHeight, err := to.stateDB.getRollbackMinHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(2500-rollbackMaxBlocks), rollbackMinHeight)
	num, err = to.stateDB.getMinAcceptableBlockNum()
	require.NoError(t, err)
	expected, err := to.stateDB.blockIdToNum(ids[rollbackMinHeight-1])
	require.NoError(t, err)
	assert.Equal(t, expected, num)

	// Rollback invalidates the cache.
	to.fullRollbackBlockClearCache(t, ids[2400-1])
	assert.False(t, cacheIsValid(&to.stateDB.minAcceptableBlockNum))
	num, err = to.stateDB.getMinAcceptableBlockNum()
	require.NoError(t, err)
	assert.Equal(t, expected, num)
}

func TestValidBlockNumWatermark(t *testing.T) {
	to := createStorageObjects(t, true)

	// Empty state.
	w, err := to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), w)

	ids := addAndFlushBlocks(t, to, 300)
	to.stateDB.validBlockNumWatermark.invalidate()
	w, err = to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), w) // No rollbacks, all block numbers are valid.

	// Rollback the top 100 blocks, block numbers 200..299 become invalid.
	to.fullRollbackBlockClearCache(t, ids[199])
	assert.False(t, cacheIsValid(&to.stateDB.validBlockNumWatermark))
	w, err = to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, uint32(300), w)

	// New blocks do not change the cached watermark, their numbers are above it.
	newIDs := addAndFlushBlocks(t, to, 50)
	assert.True(t, cacheIsValid(&to.stateDB.validBlockNumWatermark))
	for _, id := range newIDs {
		num, nErr := to.stateDB.blockIdToNum(id)
		require.NoError(t, nErr)
		assert.GreaterOrEqual(t, num, w)
	}
	// Recalculation (as after a restart) finds the gap left by the rolled back numbers.
	to.stateDB.validBlockNumWatermark.invalidate()
	w, err = to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, uint32(300), w)

	// Every number at or above the watermark that was assigned is valid, the rolled back ones are below it.
	for num := range uint32(350) {
		valid, vErr := to.stateDB.isValidBlock(num)
		require.NoError(t, vErr)
		if num >= w {
			assert.True(t, valid, "block number %d", num)
		}
		if num >= 200 && num < 300 {
			assert.False(t, valid, "block number %d", num)
		}
	}

	// Rollback of a block without the height update (as in rollbackBlock helper) also invalidates the watermark.
	to.rollbackBlock(t, newIDs[len(newIDs)-1])
	assert.False(t, cacheIsValid(&to.stateDB.validBlockNumWatermark))
	w, err = to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, uint32(350), w)
}

func TestBlockNumCacheDoesNotStoreStaleOrFailedValues(t *testing.T) {
	var c blockNumCache
	// The value computed concurrently with an invalidation is returned but not cached.
	v, err := c.get(func() (uint32, error) {
		c.invalidate()
		return 1, nil
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(1), v)
	assert.False(t, cacheIsValid(&c))
	// Errors are not cached.
	_, err = c.get(func() (uint32, error) { return 0, errors.New("compute failed") })
	require.Error(t, err)
	assert.False(t, cacheIsValid(&c))
	v, err = c.get(func() (uint32, error) { return 2, nil })
	require.NoError(t, err)
	assert.Equal(t, uint32(2), v)
	assert.True(t, cacheIsValid(&c))
	v, err = c.get(func() (uint32, error) { return 3, nil })
	require.NoError(t, err)
	assert.Equal(t, uint32(2), v) // Cached.
}

func TestValidBlockNumWatermarkInvalidatedOnRollbackFlush(t *testing.T) {
	to := createStorageObjects(t, true)
	ids := addAndFlushBlocks(t, to, 10)
	top := ids[len(ids)-1]
	topNum, err := to.stateDB.blockIdToNum(top)
	require.NoError(t, err)

	require.NoError(t, to.stateDB.rollbackBlock(top))
	// The watermark read before the rollback is written is computed from the committed data and cached.
	w, err := to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.LessOrEqual(t, w, topNum)
	assert.True(t, cacheIsValid(&to.stateDB.validBlockNumWatermark))

	// Writing the rollback invalidates it.
	require.NoError(t, to.stateDB.flushBatch())
	assert.False(t, cacheIsValid(&to.stateDB.validBlockNumWatermark))
	w, err = to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Greater(t, w, topNum)
}

func TestValidBlockNumWatermarkWalkLimit(t *testing.T) {
	to := createStorageObjects(t, true)
	const n = rollbackMaxBlocks + 100
	addAndFlushBlocks(t, to, n)
	to.stateDB.validBlockNumWatermark.invalidate()
	last, err := to.stateDB.getLastBlockNum()
	require.NoError(t, err)
	w, err := to.stateDB.getValidBlockNumWatermark()
	require.NoError(t, err)
	assert.Equal(t, last-rollbackMaxBlocks, w) // The walk stops at the limit, not at the first block.
}

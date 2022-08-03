package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rollbackEdge = 3500
	totalBlocks  = 4000
)

type historyTestObjects struct {
	stor *testStorageObjects
	fmt  *historyFormatter
}

func createHistory(t *testing.T) *historyTestObjects {
	stor := createStorageObjects(t, true)
	fmt, err := newHistoryFormatter(stor.stateDB)
	require.NoError(t, err)
	return &historyTestObjects{stor, fmt}
}

func TestNormalizeFeatureVote(t *testing.T) {
	// featureVote entity does not need cuts.
	to := createHistory(t)

	// Create history record and add blocks.
	ids := genRandBlockIds(t, totalBlocks)
	history := newHistoryRecord(featureVote)
	for _, id := range ids {
		to.stor.addBlock(t, id)
		blockNum, err := to.stor.stateDB.newestBlockIdToNum(id)
		assert.NoError(t, err, "blockIdToNum() failed")
		entry := historyEntry{nil, blockNum}
		err = history.appendEntry(entry)
		assert.NoError(t, err, "appendEntry() failed")
	}
	to.stor.flush(t)

	historyBackup := make([]historyEntry, len(history.entries))
	copy(historyBackup, history.entries)

	// Normalize and check that nothing has changed.
	changed, err := to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	assert.Equal(t, false, changed)
	assert.Equal(t, historyBackup, history.entries)

	// Now rollback the last block to check filtering.
	id := ids[len(ids)-1]
	to.stor.rollbackBlock(t, id)

	// Normalize and check the result.
	changed, err = to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	assert.Equal(t, true, changed)
	assert.Equal(t, historyBackup[:len(historyBackup)-1], history.entries)
}

func TestNormalize(t *testing.T) {
	to := createHistory(t)

	// Create history record and add blocks.
	ids := genRandBlockIds(t, totalBlocks)
	history := newHistoryRecord(alias)
	for _, id := range ids {
		to.stor.addBlock(t, id)
		blockNum, err := to.stor.stateDB.newestBlockIdToNum(id)
		assert.NoError(t, err, "blockIdToNum() failed")
		entry := historyEntry{nil, blockNum}
		err = history.appendEntry(entry)
		assert.NoError(t, err, "appendEntry() failed")
	}
	to.stor.flush(t)

	// Rollback some of blocks.
	for i, id := range ids {
		if i >= rollbackEdge {
			to.stor.rollbackBlock(t, id)
		}
	}

	// Normalize and check the result.
	changed, err := to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	assert.Equal(t, true, changed)
	rollbackMinHeight, err := to.stor.stateDB.getRollbackMinHeight()
	assert.NoError(t, err, "getRollbackMinHeight() failed")
	oldRecordNumber := 0
	for _, entry := range history.entries {
		blockID, err := to.stor.stateDB.blockNumToId(entry.blockNum)
		assert.NoError(t, err, "blockNumToId() failed")
		entryHeight, err := to.stor.rw.newestHeightByBlockID(blockID)
		assert.NoError(t, err, "newestHeightByBlockID failed")
		if entryHeight < rollbackMinHeight {
			oldRecordNumber++
		}
		if entryHeight > rollbackEdge {
			t.Errorf("History formatter did not erase invalid blocks.")
		}
	}
	if oldRecordNumber != 1 {
		t.Errorf("History formatter did not cut old blocks.")
	}
}

package state

import (
	"fmt"
	"math/rand/v2"
	"slices"
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
	fmt := newHistoryFormatter(stor.stateDB)
	return &historyTestObjects{stor, fmt}
}

func TestNormalizeFeatureVote(t *testing.T) {
	// featureVote entity does not need cuts.
	to := createHistory(t)

	// Create history record and add blocks.
	ids := genRandBlockIDs(t, totalBlocks)
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
	ids := genRandBlockIDs(t, totalBlocks)
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

// referenceFilter is the previous implementation of historyFormatter.filter, it checks every entry in the DB.
func referenceFilter(t *testing.T, db *stateDB, history *historyRecord) bool {
	property, ok := entityProperties(history.entityType)
	require.True(t, ok)
	if !property.needToFilter {
		return false
	}
	changed := false
	for i := range slices.Backward(history.entries) {
		valid, err := db.isValidBlock(history.entries[i].blockNum)
		require.NoError(t, err)
		if valid {
			break
		}
		history.entries = history.entries[:i]
		changed = true
	}
	return changed
}

// referenceCut is the previous implementation of historyFormatter.cut, it reads the DB on every call.
// It returns whether the entries were actually changed.
func referenceCut(t *testing.T, db *stateDB, history *historyRecord) bool {
	property, ok := entityProperties(history.entityType)
	require.True(t, ok)
	if !property.needToCut {
		return false
	}
	rollbackMinHeight, err := db.getRollbackMinHeight()
	require.NoError(t, err)
	minAcceptableBlockNum, err := db.blockNumByHeight(rollbackMinHeight)
	require.NoError(t, err)
	firstNeeded := 0
	for i, entry := range history.entries {
		if entry.blockNum < minAcceptableBlockNum {
			firstNeeded = i
			continue
		}
		break
	}
	history.entries = history.entries[firstNeeded:]
	return firstNeeded > 0
}

func entriesOf(nums ...uint32) []historyEntry {
	r := make([]historyEntry, len(nums))
	for i, n := range nums {
		r[i] = historyEntry{data: []byte{byte(i)}, blockNum: n}
	}
	return r
}

func TestHistoryFormatterMatchesReference(t *testing.T) {
	to := createHistory(t)
	// Heights 1..2500, block numbers 0..2499, rollback min height is 500 with block number 499.
	ids := addAndFlushBlocks(t, to.stor, 2500)
	// Rollback to height 2400, block numbers 2400..2499 become invalid.
	to.stor.fullRollbackBlockClearCache(t, ids[2399])
	// New blocks at heights 2401..2450 get block numbers 2500..2549.
	addAndFlushBlocks(t, to.stor, 50)

	minNum, err := to.fmt.db.getMinAcceptableBlockNum()
	require.NoError(t, err)
	require.Equal(t, uint32(499), minNum)

	rnd := rand.New(rand.NewPCG(1, 2))
	randomHistory := func() []historyEntry {
		var nums []uint32
		for n := range uint32(2550) {
			if rnd.IntN(100) == 0 {
				nums = append(nums, n)
			}
		}
		return entriesOf(nums...)
	}
	tests := []struct {
		name    string
		entries []historyEntry
	}{
		{name: "empty", entries: nil},
		{name: "single old", entries: entriesOf(10)},
		{name: "single invalid", entries: entriesOf(2450)},
		{name: "single new", entries: entriesOf(2520)},
		{name: "one old then acceptable", entries: entriesOf(10, 600, 700)},
		{name: "many old", entries: entriesOf(1, 2, 3, 100, 498, 499, 500, 1000)},
		{name: "all old", entries: entriesOf(1, 2, 3, 100, 498)},
		{name: "old entry at the boundary", entries: entriesOf(1, 2, 499)},
		{name: "all acceptable", entries: entriesOf(499, 500, 2399)},
		{name: "invalid on top", entries: entriesOf(1, 2, 600, 2399, 2400, 2450, 2499)},
		{name: "new above invalid", entries: entriesOf(1, 600, 2410, 2500, 2549)},
		{name: "only invalid", entries: entriesOf(2400, 2401, 2499)},
		{name: "old and invalid", entries: entriesOf(1, 2, 3, 2400, 2499)},
		{name: "random 1", entries: randomHistory()},
		{name: "random 2", entries: randomHistory()},
		{name: "random 3", entries: randomHistory()},
	}
	for _, et := range []blockchainEntity{wavesBalance, alias, featureVote} {
		for _, amend := range []bool{true, false} {
			for _, tc := range tests {
				t.Run(fmt.Sprintf("%s/entity=%d/amend=%t", tc.name, et, amend), func(t *testing.T) {
					expected := &historyRecord{entityType: et, entries: slices.Clone(tc.entries)}
					expectedChanged := false
					if amend {
						expectedChanged = referenceFilter(t, to.stor.stateDB, expected)
					}
					expectedChanged = referenceCut(t, to.stor.stateDB, expected) || expectedChanged

					actual := &historyRecord{entityType: et, entries: slices.Clone(tc.entries)}
					changed, nErr := to.fmt.normalize(actual, amend)
					require.NoError(t, nErr)
					assert.Equal(t, expectedChanged, changed)
					assert.Equal(t, expected.entries, actual.entries)
					assert.Equal(t, len(tc.entries) != len(actual.entries), changed)
				})
			}
		}
	}
}

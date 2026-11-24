package state

import (
	"cmp"
	"slices"

	"github.com/pkg/errors"
)

// historyFormatter formats histories. It can `cut` and `filter` histories.
// `Cut` removes outdated blocks (blocks that are more than `rollbackMaxBlocks` in the past)
// from the beginning of the history.
// `Filter` removes invalid blocks from the end of the history. Blocks become invalid when they are rolled back.
// It simply looks at the list of valid blocks, and considers block as invalid if its unique number is not in this list.
type historyFormatter struct {
	db *stateDB
}

func newHistoryFormatter(db *stateDB) *historyFormatter {
	return &historyFormatter{db: db}
}

// filter removes entries from the history record that belongs to a removed by roll back blocks.
func (hfmt *historyFormatter) filter(history *historyRecord) (bool, error) {
	property, ok := entityProperties(history.entityType)
	if !ok {
		return false, errors.Errorf("bad entity type: %v", history.entityType)
	}
	if !property.needToFilter || len(history.entries) == 0 {
		// This type of entities needs no filtering.
		return false, nil
	}
	// All block numbers starting from the watermark are known to be valid, no DB lookups are needed for them.
	watermark, err := hfmt.db.getValidBlockNumWatermark()
	if err != nil {
		return false, err
	}
	changed := false
	for i := range slices.Backward(history.entries) {
		entry := history.entries[i]
		if entry.blockNum >= watermark {
			break
		}
		valid, vErr := hfmt.db.isValidBlock(entry.blockNum)
		if vErr != nil {
			return false, vErr
		}
		if valid {
			// Is valid entry.
			break
		}
		// Erase invalid entry.
		history.entries = history.entries[:i]
		changed = true
	}
	return changed, nil
}

// cut removes the oldest entries from the history record.
// cut always left one entry even if it block number is less than least acceptable block number.
// History entries are ordered by block number, because block numbers are assigned sequentially.
// The result is reported as changed only if some entries were actually removed.
func (hfmt *historyFormatter) cut(history *historyRecord) (bool, error) {
	property, ok := entityProperties(history.entityType)
	if !ok {
		return false, errors.Errorf("bad entity type: %v", history.entityType)
	}
	if !property.needToCut || len(history.entries) < 2 {
		// This type of entities needs no cuts, or there is nothing to cut.
		return false, nil
	}
	minAcceptableBlockNum, err := hfmt.db.getMinAcceptableBlockNum()
	if err != nil {
		return false, err
	}
	if history.entries[0].blockNum >= minAcceptableBlockNum {
		return false, nil // The oldest entry is acceptable, nothing to cut.
	}
	// Number of entries with block number less than minimal acceptable, it's at least 1 here.
	n, _ := slices.BinarySearchFunc(history.entries, minAcceptableBlockNum, func(e historyEntry, num uint32) int {
		return cmp.Compare(e.blockNum, num)
	})
	// 1 entry BEFORE minAcceptableHeight is needed.
	firstNeeded := n - 1
	if firstNeeded == 0 {
		return false, nil
	}
	history.entries = history.entries[firstNeeded:]
	return true, nil
}

func (hfmt *historyFormatter) normalize(history *historyRecord, amend bool) (bool, error) {
	filtered := false
	if amend {
		var err error
		filtered, err = hfmt.filter(history)
		if err != nil {
			return false, err
		}
	}
	cut, err := hfmt.cut(history)
	if err != nil {
		return false, err
	}
	return filtered || cut, nil
}

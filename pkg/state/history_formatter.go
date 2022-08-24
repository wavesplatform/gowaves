package state

import (
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

func newHistoryFormatter(db *stateDB) (*historyFormatter, error) {
	return &historyFormatter{db}, nil
}

// filter removes entries from the history record that belongs to a removed by roll back blocks.
func (hfmt *historyFormatter) filter(history *historyRecord) (bool, error) {
	property, ok := properties[history.entityType]
	if !ok {
		return false, errors.Errorf("bad entity type: %v", history.entityType)
	}
	if !property.needToFilter {
		// This type of entities needs no filtering.
		return false, nil
	}
	changed := false
	for i := len(history.entries) - 1; i >= 0; i-- {
		entry := history.entries[i]
		valid, err := hfmt.db.isValidBlock(entry.blockNum)
		if err != nil {
			return false, err
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

func (hfmt *historyFormatter) calculateMinAcceptableBlockNum() (uint32, error) {
	rollbackMinHeight, err := hfmt.db.getRollbackMinHeight()
	if err != nil {
		return 0, err
	}

	minAcceptableBlockNum, err := hfmt.db.blockNumByHeight(rollbackMinHeight)
	if err != nil {
		return 0, err
	}
	return minAcceptableBlockNum, nil
}

// cut removes the oldest entries from the history record.
// cut always left one entry even if it block number is less than least acceptable block number.
func (hfmt *historyFormatter) cut(history *historyRecord) (bool, error) {
	property, ok := properties[history.entityType]
	if !ok {
		return false, errors.Errorf("bad entity type: %v", history.entityType)
	}
	if !property.needToCut {
		// This type of entities needs no cuts.
		return false, nil
	}
	changed := false
	firstNeeded := 0
	minAcceptableBlockNum, err := hfmt.calculateMinAcceptableBlockNum()
	if err != nil {
		return false, err
	}

	for i, entry := range history.entries {
		if entry.blockNum < minAcceptableBlockNum {
			// 1 entry BEFORE minAcceptableHeight is needed.
			firstNeeded = i
			changed = true
			continue
		}
		break
	}
	history.entries = history.entries[firstNeeded:]
	return changed, nil
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

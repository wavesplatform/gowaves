package state

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
)

func isOldBlock(rw *blockReadWriter, stateDB *stateDB, blockNum uint32) (bool, error) {
	currentHeight := rw.recentHeight()
	blockID, err := stateDB.blockNumToId(blockNum)
	if err != nil {
		return false, err
	}
	blockHeight, err := rw.heightByBlockID(blockID)
	if err != nil {
		return false, err
	}
	if (blockHeight == 0) || (currentHeight-blockHeight > uint64(rollbackMaxBlocks)) {
		return true, nil
	}
	return false, nil
}

type historyFormatter struct {
	db *stateDB
	rw *blockReadWriter
}

func newHistoryFormatter(db *stateDB, rw *blockReadWriter) (*historyFormatter, error) {
	return &historyFormatter{db, rw}, nil
}

func (hfmt *historyFormatter) getID(record []byte) ([]byte, error) {
	if len(record) < idSize {
		return nil, errors.New("invalid record size")
	}
	return record[len(record)-idSize:], nil
}

func (hfmt *historyFormatter) addRecord(history *historyRecord, record []byte) error {
	if len(history.records) == 0 {
		// History is empty, new record is the first one.
		history.records = append(history.records, record)
		return nil
	}
	lastRecord, err := hfmt.getLatest(history)
	if err != nil {
		return err
	}
	lastID, err := hfmt.getID(lastRecord)
	if err != nil {
		return err
	}
	curID, err := hfmt.getID(record)
	if err != nil {
		return err
	}
	if bytes.Equal(lastID, curID) {
		// If the last ID is the same, rewrite the last record.
		history.records[len(history.records)-1] = record
	} else {
		// Append new record to the end.
		history.records = append(history.records, record)
	}
	return nil
}

func (hfmt *historyFormatter) getLatest(history *historyRecord) ([]byte, error) {
	if len(history.records) < 1 {
		return nil, errors.Errorf("invalid history size")
	}
	return history.records[len(history.records)-1], nil
}

func (hfmt *historyFormatter) filter(history *historyRecord) (bool, error) {
	changed := false
	for i := len(history.records) - 1; i >= 0; i-- {
		record := history.records[i]
		blockNumBytes, err := hfmt.getID(record)
		if err != nil {
			return false, err
		}
		blockNum := binary.BigEndian.Uint32(blockNumBytes)
		valid, err := hfmt.db.isValidBlock(blockNum)
		if err != nil {
			return false, err
		}
		if valid {
			// Is valid record.
			break
		}
		// Erase invalid record.
		history.records = history.records[:i]
		changed = true
	}
	return changed, nil
}

func (hfmt *historyFormatter) cut(history *historyRecord) (bool, error) {
	changed := false
	firstNeeded := 0
	for i, record := range history.records {
		blockNumBytes, err := hfmt.getID(record)
		if err != nil {
			return false, err
		}
		blockNum := binary.BigEndian.Uint32(blockNumBytes)
		isOld, err := isOldBlock(hfmt.rw, hfmt.db, blockNum)
		if err != nil {
			return false, err
		}
		if isOld {
			// 1 record BEFORE minHeight is needed.
			firstNeeded = i
			changed = true
			continue
		}
		break
	}
	history.records = history.records[firstNeeded:]
	return changed, nil
}

func (hfmt *historyFormatter) normalize(history *historyRecord, filter bool) (bool, error) {
	filtered := false
	if filter {
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
	return (filtered || cut), nil
}

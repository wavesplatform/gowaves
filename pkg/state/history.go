package state

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
)

type historyFormatter struct {
	recordSize int
	idSize     int
	db         *stateDB
	rw         *blockReadWriter
}

func newHistoryFormatter(recordSize, idSize int, db *stateDB, rw *blockReadWriter) (*historyFormatter, error) {
	if recordSize <= 0 || idSize <= 0 {
		return nil, errors.New("invalid record or id size")
	}
	if recordSize < idSize {
		return nil, errors.New("recordSize is < idSize")
	}
	return &historyFormatter{recordSize: recordSize, idSize: idSize, db: db, rw: rw}, nil
}

func (hfmt *historyFormatter) getID(record []byte) ([]byte, error) {
	if len(record) < hfmt.recordSize {
		return nil, errors.New("invalid record size")
	}
	return record[hfmt.recordSize-hfmt.idSize:], nil
}

func (hfmt *historyFormatter) addRecord(history []byte, record []byte) ([]byte, error) {
	if len(history) < hfmt.recordSize {
		// History is empty, new record is the first one.
		return record, nil
	}
	lastRecord, err := hfmt.getLatest(history)
	if err != nil {
		return nil, err
	}
	lastID, err := hfmt.getID(lastRecord)
	if err != nil {
		return nil, err
	}
	curID, err := hfmt.getID(record)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(lastID, curID) {
		// If the last ID is the same, rewrite the last record.
		copy(history[len(history)-hfmt.recordSize:], record)
	} else {
		// Append new record to the end.
		history = append(history, record...)
	}
	return history, nil
}

func (hfmt *historyFormatter) getLatest(history []byte) ([]byte, error) {
	if len(history) < hfmt.recordSize {
		return nil, errors.Errorf("invalid history size %d, min is %d\n", len(history), hfmt.recordSize)
	}
	return history[len(history)-hfmt.recordSize:], nil
}

func (hfmt *historyFormatter) filter(history []byte) ([]byte, error) {
	for i := len(history); i >= hfmt.recordSize; i -= hfmt.recordSize {
		record := history[i-hfmt.recordSize : i]
		blockNumBytes, err := hfmt.getID(record)
		if err != nil {
			return nil, err
		}
		blockNum := binary.BigEndian.Uint32(blockNumBytes)
		valid, err := hfmt.db.isValidBlock(blockNum)
		if err != nil {
			return nil, err
		}
		if valid {
			// Is valid record.
			break
		}
		// Erase invalid record.
		history = history[:i-hfmt.recordSize]
	}
	return history, nil
}

func (hfmt *historyFormatter) cut(history []byte) ([]byte, error) {
	currentHeight := hfmt.rw.recentHeight()
	firstNeeded := 0
	for i := hfmt.recordSize; i <= len(history); i += hfmt.recordSize {
		recordStart := i - hfmt.recordSize
		record := history[recordStart:i]
		blockNumBytes, err := hfmt.getID(record)
		if err != nil {
			return nil, err
		}
		blockNum := binary.BigEndian.Uint32(blockNumBytes)
		blockID, err := hfmt.db.blockNumToId(blockNum)
		if err != nil {
			return nil, err
		}
		blockHeight, err := hfmt.rw.heightByBlockID(blockID)
		if err != nil {
			return nil, err
		}
		if (blockHeight == 0) || (currentHeight-blockHeight > uint64(rollbackMaxBlocks)) {
			// 1 record BEFORE minHeight is needed.
			firstNeeded = recordStart
			continue
		}
		break
	}
	return history[firstNeeded:], nil
}

func (hfmt *historyFormatter) normalize(history []byte, filter bool) ([]byte, error) {
	var err error
	if filter {
		history, err = hfmt.filter(history)
		if err != nil {
			return nil, err
		}
	}
	history, err = hfmt.cut(history)
	if err != nil {
		return nil, err
	}
	return history, nil
}

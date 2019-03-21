package state

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

type id2Key func(id []byte) ([]byte, error)
type id2Height func(id []byte) (uint64, error)

type historyFormatter struct {
	recordSize int
	idSize     int
}

func newHistoryFormatter(recordSize, idSize int) (*historyFormatter, error) {
	if recordSize <= 0 || idSize <= 0 {
		return nil, errors.New("invalid record or id size")
	}
	if recordSize <= idSize {
		return nil, errors.New("recordSize is <= idSize")
	}
	return &historyFormatter{recordSize: recordSize, idSize: idSize}, nil
}

func (h *historyFormatter) getID(record []byte) ([]byte, error) {
	if len(record) < h.recordSize {
		return nil, errors.New("invalid record size")
	}
	return record[h.recordSize-h.idSize:], nil
}

func (h *historyFormatter) addRecord(history []byte, record []byte) ([]byte, error) {
	if len(history) < h.recordSize {
		// History is empty, new record is the first one.
		return record, nil
	}
	lastRecord, err := h.getLatest(history)
	if err != nil {
		return nil, err
	}
	lastID, err := h.getID(lastRecord)
	if err != nil {
		return nil, err
	}
	curID, err := h.getID(record)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(lastID, curID) {
		// If the last ID is the same, rewrite the last record.
		copy(history[len(history)-h.recordSize:], record)
	} else {
		// Append new record to the end.
		history = append(history, record...)
	}
	return history, nil
}

func (h *historyFormatter) getLatest(history []byte) ([]byte, error) {
	if len(history) < h.recordSize {
		return nil, errors.New("invalid history size")
	}
	return history[len(history)-h.recordSize:], nil
}

func (h *historyFormatter) filter(history []byte, db keyvalue.KeyValue, id2key id2Key) ([]byte, error) {
	for i := len(history); i >= h.recordSize; i -= recordSize {
		record := history[i-recordSize : i]
		id, err := h.getID(record)
		if err != nil {
			return nil, err
		}
		key, err := id2key(id)
		if err != nil {
			return nil, err
		}
		has, err := db.Has(key)
		if err != nil {
			return nil, err
		}
		if has {
			// Is valid record.
			break
		}
		// Erase invalid record.
		history = history[:i-recordSize]
	}
	return history, nil
}

func (h *historyFormatter) cut(history []byte, id2height id2Height, curHeight uint64, exceptID []byte) ([]byte, error) {
	firstNeeded := 0
	for i := recordSize; i <= len(history); i += recordSize {
		recordStart := i - recordSize
		record := history[recordStart:i]
		id, err := h.getID(record)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(id, exceptID) {
			blockHeight, err := id2height(id)
			if err != nil {
				return nil, err
			}
			if curHeight-blockHeight > uint64(rollbackMaxBlocks) {
				// 1 record BEFORE minHeight is needed.
				firstNeeded = recordStart
				continue
			}
			break
		}
	}
	return history[firstNeeded:], nil
}

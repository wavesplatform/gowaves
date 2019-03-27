package history

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type blockInfo interface {
	IsValidBlock(blockID crypto.Signature) (bool, error)
}

type heightInfo interface {
	Height() (uint64, error)
	BlockIDToHeight(blockID crypto.Signature) (uint64, error)
	RollbackMax() uint64
}

type HistoryFormatter struct {
	recordSize int
	idSize     int
	hInfo      heightInfo
	bInfo      blockInfo
}

func NewHistoryFormatter(recordSize, idSize int, hInfo heightInfo, bInfo blockInfo) (*HistoryFormatter, error) {
	if recordSize <= 0 || idSize <= 0 {
		return nil, errors.New("invalid record or id size")
	}
	if recordSize < idSize {
		return nil, errors.New("recordSize is < idSize")
	}
	return &HistoryFormatter{recordSize: recordSize, idSize: idSize, hInfo: hInfo, bInfo: bInfo}, nil
}

func (hfmt *HistoryFormatter) GetID(record []byte) ([]byte, error) {
	if len(record) < hfmt.recordSize {
		return nil, errors.New("invalid record size")
	}
	return record[hfmt.recordSize-hfmt.idSize:], nil
}

func (hfmt *HistoryFormatter) AddRecord(history []byte, record []byte) ([]byte, error) {
	if len(history) < hfmt.recordSize {
		// History is empty, new record is the first one.
		return record, nil
	}
	lastRecord, err := hfmt.GetLatest(history)
	if err != nil {
		return nil, err
	}
	lastID, err := hfmt.GetID(lastRecord)
	if err != nil {
		return nil, err
	}
	curID, err := hfmt.GetID(record)
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

func (hfmt *HistoryFormatter) GetLatest(history []byte) ([]byte, error) {
	if len(history) < hfmt.recordSize {
		return nil, errors.New("invalid history size")
	}
	return history[len(history)-hfmt.recordSize:], nil
}

func (hfmt *HistoryFormatter) Filter(history []byte) ([]byte, error) {
	for i := len(history); i >= hfmt.recordSize; i -= hfmt.recordSize {
		record := history[i-hfmt.recordSize : i]
		id, err := hfmt.GetID(record)
		if err != nil {
			return nil, err
		}
		blockID, err := crypto.NewSignatureFromBytes(id)
		if err != nil {
			return nil, err
		}
		valid, err := hfmt.bInfo.IsValidBlock(blockID)
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

func (hfmt *HistoryFormatter) Cut(history []byte) ([]byte, error) {
	firstNeeded := 0
	for i := hfmt.recordSize; i <= len(history); i += hfmt.recordSize {
		recordStart := i - hfmt.recordSize
		record := history[recordStart:i]
		id, err := hfmt.GetID(record)
		if err != nil {
			return nil, err
		}
		blockID, err := crypto.NewSignatureFromBytes(id)
		if err != nil {
			return nil, err
		}
		blockHeight, err := hfmt.hInfo.BlockIDToHeight(blockID)
		if err != nil {
			return nil, err
		}
		currentHeight, err := hfmt.hInfo.Height()
		if err != nil {
			return nil, err
		}
		if currentHeight-blockHeight > hfmt.hInfo.RollbackMax() {
			// 1 record BEFORE minHeight is needed.
			firstNeeded = recordStart
			continue
		}
		break
	}
	return history[firstNeeded:], nil
}

func (hfmt *HistoryFormatter) Normalize(history []byte) ([]byte, error) {
	history, err := hfmt.Filter(history)
	if err != nil {
		return nil, err
	}
	history, err = hfmt.Cut(history)
	if err != nil {
		return nil, err
	}
	return history, nil
}

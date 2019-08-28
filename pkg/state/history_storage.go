package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var errEmptyHist = errors.New("empty history for this record")

type blockchainEntity byte

const (
	alias blockchainEntity = iota
	asset
	lease
	wavesBalance
	assetBalance
	featureVote
	approvedFeature
	activatedFeature
	sponsorship
	dataEntry
	accountScript

	idSize = 4
)

var recordSizes = map[blockchainEntity]int{
	alias:            aliasRecordSize,
	asset:            assetRecordSize,
	lease:            leasingRecordSize,
	wavesBalance:     wavesBalanceRecordSize,
	assetBalance:     assetBalanceRecordSize,
	featureVote:      votesFeaturesRecordSize,
	approvedFeature:  approvedFeaturesRecordSize,
	activatedFeature: activatedFeaturesRecordSize,
	sponsorship:      sponsorshipRecordSize,
}

type historyRecord struct {
	fixedSize bool
	// recordSize is specified if fixedSize is true.
	// Otherwise records sizes are 4 first bytes of each record.
	recordSize uint32
	records    [][]byte
}

func newHistoryRecordFromBytes(data []byte) (*historyRecord, error) {
	dataSize := uint32(len(data))
	if dataSize < 1 {
		return nil, errors.New("invalid data size")
	}
	fixedSize, err := proto.Bool(data)
	if err != nil {
		return nil, err
	}
	recordSize := uint32(0)
	var records [][]byte
	if fixedSize {
		if dataSize < 5 {
			return nil, errors.New("invalid data size")
		}
		recordSize = binary.BigEndian.Uint32(data[1:5])
		if dataSize < 5+recordSize {
			return nil, errors.New("invalid data size")
		}
		for i := uint32(5); i <= dataSize-recordSize; i += recordSize {
			record := data[i : i+recordSize]
			records = append(records, record)
		}
	} else {
		for i := uint32(1); i <= dataSize-4; {
			recordSize := binary.BigEndian.Uint32(data[i : i+4])
			i += 4
			if dataSize < i+recordSize {
				return nil, errors.New("invalid data size")
			}
			record := data[i : i+recordSize]
			records = append(records, record)
			i += recordSize
		}
	}
	return &historyRecord{fixedSize, recordSize, records}, nil
}

func (hr *historyRecord) countTotalSize() int {
	totalSize := 1
	if hr.fixedSize {
		totalSize += 4
	}
	for _, r := range hr.records {
		totalSize += len(r)
		if !hr.fixedSize {
			totalSize += 4
		}
	}
	return totalSize
}

func (hr *historyRecord) marshalBinary() ([]byte, error) {
	data := make([]byte, hr.countTotalSize())
	proto.PutBool(data, hr.fixedSize)
	curPos := 1
	if hr.fixedSize {
		// Add size of all records.
		binary.BigEndian.PutUint32(data[curPos:curPos+4], hr.recordSize)
		curPos += 4
	}
	for _, r := range hr.records {
		if !hr.fixedSize {
			// Add size of this record.
			size := len(r)
			binary.BigEndian.PutUint32(data[curPos:curPos+4], uint32(size))
			curPos += 4
		}
		copy(data[curPos:], r)
		curPos += len(r)
	}
	return data, nil
}

func (hr *historyRecord) merge(newHr *historyRecord) error {
	if hr.fixedSize != newHr.fixedSize {
		return errors.New("trying to merge incompatible histories")
	}
	if hr.recordSize != newHr.recordSize {
		return errors.New("trying to merge incompatible histories")
	}
	hr.records = append(hr.records, newHr.records...)
	return nil
}

type historyStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch

	stateDB *stateDB
	rw      *blockReadWriter

	stor *localHistoryStorage
	fmt  *historyFormatter
}

func newHistoryStorage(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	rw *blockReadWriter,
	stateDB *stateDB,
) (*historyStorage, error) {
	stor, err := newLocalHistoryStorage()
	if err != nil {
		return nil, err
	}
	fmt, err := newHistoryFormatter(stateDB, rw)
	if err != nil {
		return nil, err
	}
	return &historyStorage{db, dbBatch, stateDB, rw, stor, fmt}, nil
}

func (hs *historyStorage) set(entityType blockchainEntity, key, value []byte) error {
	history, err := hs.stor.get(key)
	if err == errNotFound {
		fixedSize := true
		size, ok := recordSizes[entityType]
		if !ok {
			fixedSize = false
		}
		history = &historyRecord{fixedSize: fixedSize, recordSize: uint32(size)}
	} else if err != nil {
		return err
	}
	if err := hs.fmt.addRecord(history, value); err != nil {
		return err
	}
	if err := hs.stor.set(key, history); err != nil {
		return err
	}
	return nil
}

func (hs *historyStorage) cleanDbRecord(key []byte) error {
	// If the history is empty after normalizing, it means that all the records were removed due to rollback.
	// In this case, it should be removed from the DB as well.
	return hs.db.Delete(key)
}

func (hs *historyStorage) getHistory(key []byte, filter, update bool) (*historyRecord, error) {
	historyBytes, err := hs.db.Get(key)
	if err != nil {
		return nil, err
	}
	history, err := newHistoryRecordFromBytes(historyBytes)
	if err != nil {
		return nil, err
	}
	changed, err := hs.fmt.normalize(history, filter)
	if err != nil {
		return nil, err
	}
	if len(history.records) == 0 {
		if err := hs.cleanDbRecord(key); err != nil {
			return nil, err
		}
		return nil, errEmptyHist
	} else if changed && update {
		newHistoryBytes, err := history.marshalBinary()
		if err != nil {
			return nil, err
		}
		if err := hs.db.Put(key, newHistoryBytes); err != nil {
			return nil, err
		}
	}
	return history, nil
}

func (hs *historyStorage) get(key []byte, filter bool) ([]byte, error) {
	history, err := hs.getHistory(key, filter, false)
	if err != nil {
		return nil, err
	}
	return hs.fmt.getLatest(history)
}

func (hs *historyStorage) getFresh(key []byte, filter bool) ([]byte, error) {
	if newHist, err := hs.stor.get(key); err == nil {
		return hs.fmt.getLatest(newHist)
	}
	return hs.get(key, filter)
}

func (hs *historyStorage) combineHistories(key []byte, newHist *historyRecord, filter bool) (*historyRecord, error) {
	prevHist, err := hs.getHistory(key, filter, true)
	if err == keyvalue.ErrNotFound {
		// New history.
		return newHist, nil
	} else if err != nil {
		return nil, err
	}
	if err := prevHist.merge(newHist); err != nil {
		return nil, err
	}
	return prevHist, nil
}

// fullHistory returns combination of history from DB and the local storage (if any).
func (hs *historyStorage) fullHistory(key []byte, filter bool) (*historyRecord, error) {
	newHist, err := hs.stor.get(key)
	if err == errNotFound {
		return hs.getHistory(key, filter, true)
	} else if err != nil {
		return nil, err
	}
	return hs.combineHistories(key, newHist, filter)
}

func (hs *historyStorage) recordsInHeightRange(key []byte, startHeight, endHeight uint64, filter bool) ([][]byte, error) {
	history, err := hs.fullHistory(key, filter)
	if err != nil {
		return nil, err
	}
	foundAtLeastOne := false
	var records [][]byte
	for i := len(history.records) - 1; i >= 0; i-- {
		recordBytes := history.records[i]
		idBytes, err := hs.fmt.getID(recordBytes)
		if err != nil {
			return nil, err
		}
		blockNum := binary.BigEndian.Uint32(idBytes)
		blockID, err := hs.stateDB.blockNumToId(blockNum)
		if err != nil {
			return nil, err
		}
		height, err := hs.rw.newestHeightByBlockID(blockID)
		if err != nil {
			return nil, err
		}
		if height > endHeight {
			continue
		}
		if height < startHeight && foundAtLeastOne {
			break
		}
		foundAtLeastOne = true
		records = append(records, recordBytes)
	}
	return records, nil
}

func (hs *historyStorage) reset() {
	hs.stor.reset()
}

func (hs *historyStorage) flush(filter bool) error {
	entries := hs.stor.getEntries()
	sortEntries(entries)
	for _, entry := range entries {
		newEntry, err := hs.combineHistories(entry.key, entry.value, filter)
		if err != nil {
			return err
		}
		newEntryBytes, err := newEntry.marshalBinary()
		if err != nil {
			return err
		}
		hs.dbBatch.Put(entry.key, newEntryBytes)
	}
	hs.stor.reset()
	return nil
}

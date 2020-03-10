package state

import (
	"encoding/binary"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
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
	ordersVolume
	sponsorship
	dataEntry
	accountScript
	assetScript
	accountScriptComplexity
	assetScriptComplexity
	rewardVotes
	blockReward
	invokeResult
)

type blockchainEntityProperties struct {
	needToFilter bool
	needToCut    bool

	fixedSize  bool
	recordSize int
}

// + 4 bytes for blockNum at the end of each record.
var properties = map[blockchainEntity]blockchainEntityProperties{
	alias: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   aliasRecordSize + 4,
	},
	asset: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	lease: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   leasingRecordSize + 4,
	},
	wavesBalance: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   wavesBalanceRecordSize + 4,
	},
	assetBalance: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   assetBalanceRecordSize + 4,
	},
	featureVote: {
		needToFilter: true,
		needToCut:    false, // Do not cut for votesAtHeight().
		fixedSize:    true,
		recordSize:   votesFeaturesRecordSize + 4,
	},
	approvedFeature: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   approvedFeaturesRecordSize + 4,
	},
	activatedFeature: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   activatedFeaturesRecordSize + 4,
	},
	ordersVolume: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   orderVolumeRecordSize + 4,
	},
	sponsorship: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   sponsorshipRecordSize + 4,
	},
	dataEntry: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	accountScript: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	assetScript: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	accountScriptComplexity: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	assetScriptComplexity: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   assetScriptComplexityRecordSize + 4,
	},
	rewardVotes: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   rewardVotesRecordSize + 4,
	},
	blockReward: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   blockRewardRecordSize + 4,
	},
	invokeResult: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
}

type historyEntry struct {
	data     []byte
	blockNum uint32
}

func (he *historyEntry) size() int {
	return len(he.data) + 4
}

func (he *historyEntry) marshalBinary() ([]byte, error) {
	res := make([]byte, len(he.data)+4)
	pos := 0
	copy(res[:len(he.data)], he.data)
	pos += len(he.data)
	binary.BigEndian.PutUint32(res[pos:pos+4], he.blockNum)
	return res, nil
}

func (he *historyEntry) unmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return errInvalidDataSize
	}
	he.data = make([]byte, len(data)-4)
	copy(he.data, data[:len(data)-4])
	he.blockNum = binary.BigEndian.Uint32(data[len(data)-4:])
	return nil
}

type historyRecord struct {
	entityType blockchainEntity
	entries    []historyEntry
}

func newHistoryRecord(entityType blockchainEntity) *historyRecord {
	return &historyRecord{entityType: entityType}
}

func newHistoryRecordFromBytes(data []byte) (*historyRecord, error) {
	if len(data) < 1 {
		return nil, errInvalidDataSize
	}
	entityType := blockchainEntity(data[0])
	property, ok := properties[entityType]
	if !ok {
		return nil, errors.Errorf("bad entity type %v", entityType)
	}
	var entries []historyEntry
	if property.fixedSize {
		dataSize := len(data)
		recordSize := property.recordSize
		if dataSize < 1+recordSize {
			return nil, errInvalidDataSize
		}
		for i := 1; i <= dataSize-recordSize; i += recordSize {
			var entry historyEntry
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
	} else {
		dataSize := uint32(len(data))
		for i := uint32(1); i <= dataSize-4; {
			recordSize := binary.BigEndian.Uint32(data[i : i+4])
			i += 4
			if dataSize < i+recordSize {
				return nil, errInvalidDataSize
			}
			var entry historyEntry
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil {
				return nil, err
			}
			entries = append(entries, entry)
			i += recordSize
		}
	}
	return &historyRecord{entityType, entries}, nil
}

func (hr *historyRecord) fixedSize() (bool, error) {
	property, ok := properties[hr.entityType]
	if !ok {
		return false, errors.Errorf("bad entity type %v", hr.entityType)
	}
	return property.fixedSize, nil
}

func (hr *historyRecord) countTotalSize() (int, error) {
	totalSize := 1
	fixedSize, err := hr.fixedSize()
	if err != nil {
		return 0, err
	}
	for _, r := range hr.entries {
		totalSize += r.size()
		if !fixedSize {
			totalSize += 4
		}
	}
	return totalSize, nil
}

func (hr *historyRecord) marshalBinary() ([]byte, error) {
	totalSize, err := hr.countTotalSize()
	if err != nil {
		return nil, err
	}
	data := make([]byte, totalSize)
	data[0] = byte(hr.entityType)
	curPos := 1
	for _, entry := range hr.entries {
		fixedSize, err := hr.fixedSize()
		if err != nil {
			return nil, err
		}
		if !fixedSize {
			// Add size of this record.
			size := entry.size()
			binary.BigEndian.PutUint32(data[curPos:curPos+4], uint32(size))
			curPos += 4
		}
		entryBytes, err := entry.marshalBinary()
		if err != nil {
			return nil, err
		}
		copy(data[curPos:], entryBytes)
		curPos += entry.size()
	}
	return data, nil
}

func (hr *historyRecord) appendEntry(entry historyEntry) error {
	if len(hr.entries) == 0 {
		// History is empty, new record is the first one.
		hr.entries = append(hr.entries, entry)
	}
	latestEntry, err := hr.latestEntry()
	if err != nil {
		return err
	}
	if latestEntry.blockNum == entry.blockNum {
		// The block is the same, rewrite the last entry.
		hr.entries[len(hr.entries)-1] = entry
	} else {
		// Append new entry to the end.
		hr.entries = append(hr.entries, entry)
	}
	return nil
}

func (hr *historyRecord) latestEntry() (historyEntry, error) {
	if len(hr.entries) < 1 {
		return historyEntry{}, errors.New("empty history")
	}
	return hr.entries[len(hr.entries)-1], nil
}

// historyStorage manages the way per-block records are stored in.
// Unlike blockchain entities parts, it does not know *what* it stores, but it does know *how*.
type historyStorage struct {
	db        keyvalue.IterableKeyVal
	dbBatch   keyvalue.Batch
	writeLock *sync.Mutex
	stateDB   *stateDB

	stor *localHistoryStorage
	fmt  *historyFormatter
}

func newHistoryStorage(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	stateDB *stateDB,
) (*historyStorage, error) {
	stor, err := newLocalHistoryStorage()
	if err != nil {
		return nil, err
	}
	fmt, err := newHistoryFormatter(stateDB)
	if err != nil {
		return nil, err
	}
	return &historyStorage{
		db:        db,
		dbBatch:   dbBatch,
		writeLock: stateDB.retrieveWriteLock(),
		stateDB:   stateDB,
		stor:      stor,
		fmt:       fmt,
	}, nil
}

func (hs *historyStorage) addNewEntry(entityType blockchainEntity, key, value []byte, blockID crypto.Signature) error {
	blockNum, err := hs.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	entry := historyEntry{value, blockNum}
	history, err := hs.stor.get(key)
	if err == errNotFound {
		history = newHistoryRecord(entityType)
	} else if err != nil {
		return err
	}
	if err := history.appendEntry(entry); err != nil {
		return err
	}
	if err := hs.stor.set(key, history); err != nil {
		return err
	}
	return nil
}

// manageDbUpdate() saves updated history records directly (without batch) to database.
func (hs *historyStorage) manageDbUpdate(key []byte, history *historyRecord) error {
	if len(history.entries) == 0 {
		// If the history is empty, it means that all the entries were removed due to rollback.
		// In this case, it should be removed from the DB.
		return hs.db.Delete(key)
	}
	historyBytes, err := history.marshalBinary()
	if err != nil {
		return err
	}
	return hs.db.Put(key, historyBytes)
}

// getHistory() retrieves history record from DB. It also normalizes it,
// saving the result back to DB, if update argument is true.
func (hs *historyStorage) getHistory(key []byte, filter, update bool) (*historyRecord, error) {
	// Lock the write lock.
	// It is necessary because if we read value *before* the main write batch is written,
	// and manageDbUpdate() happens *after* it is written,
	// we might rewrite some keys that were in the batch.
	// So we do both read and write under same lock.
	hs.writeLock.Lock()
	defer hs.writeLock.Unlock()

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
	if changed && update {
		if err := hs.manageDbUpdate(key, history); err != nil {
			return nil, err
		}
	}
	if len(history.entries) == 0 {
		return nil, errEmptyHist
	}
	return history, nil
}

func (hs *historyStorage) latestEntry(key []byte, filter bool) (historyEntry, error) {
	history, err := hs.getHistory(key, filter, false)
	if err != nil {
		return historyEntry{}, err
	}
	return history.latestEntry()
}

func (hs *historyStorage) freshLatestEntry(key []byte, filter bool) (historyEntry, error) {
	if newHist, err := hs.stor.get(key); err == nil {
		return newHist.latestEntry()
	}
	return hs.latestEntry(key, filter)
}

func (hs *historyStorage) combineHistories(key []byte, newHist *historyRecord, filter bool) (*historyRecord, error) {
	prevHist, err := hs.getHistory(key, filter, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// New history.
		return newHist, nil
	} else if err != nil {
		return nil, err
	}
	if prevHist.entityType != newHist.entityType {
		return nil, errors.Errorf("trying to combine histories of different types %v and %v", prevHist.entityType, newHist.entityType)
	}
	prevHist.entries = append(prevHist.entries, newHist.entries...)
	return prevHist, nil
}

// fullHistory() returns combination of history from DB and the local storage (if any).
func (hs *historyStorage) fullHistory(key []byte, filter bool) (*historyRecord, error) {
	newHist, err := hs.stor.get(key)
	if err == errNotFound {
		return hs.getHistory(key, filter, true)
	} else if err != nil {
		return nil, err
	}
	return hs.combineHistories(key, newHist, filter)
}

// latestEntryData() returns bytes of the latest entry.
func (hs *historyStorage) latestEntryData(key []byte, filter bool) ([]byte, error) {
	entry, err := hs.latestEntry(key, filter)
	if err != nil {
		return nil, err
	}
	return entry.data, nil
}

// freshLatestEntryData() returns bytes of the latest fresh (from local storage or DB) entry.
func (hs *historyStorage) freshLatestEntryData(key []byte, filter bool) ([]byte, error) {
	entry, err := hs.freshLatestEntry(key, filter)
	if err != nil {
		return nil, err
	}
	return entry.data, nil
}

// freshBlockOfTheLatestEntry() returns block ID of the latest fresh (mem or DB) entry.
func (hs *historyStorage) freshBlockOfTheLatestEntry(key []byte, filter bool) (crypto.Signature, error) {
	entry, err := hs.freshLatestEntry(key, filter)
	if err != nil {
		return crypto.Signature{}, err
	}
	return hs.stateDB.blockNumToId(entry.blockNum)
}

// blockOfTheLatestEntry() returns block ID of the latest entry from DB.
func (hs *historyStorage) blockOfTheLatestEntry(key []byte, filter bool) (crypto.Signature, error) {
	entry, err := hs.latestEntry(key, filter)
	if err != nil {
		return crypto.Signature{}, err
	}
	return hs.stateDB.blockNumToId(entry.blockNum)
}

type entryNumsCmp func(uint32, uint32) bool

func (hs *historyStorage) entryDataWithHeightFilter(
	key []byte,
	limitHeight uint64,
	filter bool,
	cmp entryNumsCmp,
) ([]byte, error) {
	limitBlockNum, err := hs.stateDB.blockNumByHeight(limitHeight)
	if err != nil {
		return nil, err
	}
	history, err := hs.getHistory(key, filter, false)
	if err != nil {
		return nil, err
	}
	var res historyEntry
	for _, entry := range history.entries {
		if cmp(entry.blockNum, limitBlockNum) {
			res = entry
		} else {
			break
		}
	}
	return res.data, nil
}

func (hs *historyStorage) entryDataBeforeHeight(key []byte, height uint64, filter bool) ([]byte, error) {
	cmp := func(entryNum, limitNum uint32) bool {
		return entryNum < limitNum
	}
	return hs.entryDataWithHeightFilter(key, height, filter, cmp)
}

func (hs *historyStorage) entryDataAtHeight(key []byte, height uint64, filter bool) ([]byte, error) {
	cmp := func(entryNum, limitNum uint32) bool {
		return entryNum <= limitNum
	}
	return hs.entryDataWithHeightFilter(key, height, filter, cmp)
}

// freshEntryBeforeHeight() returns bytes of the latest fresh (from local storage or DB) entry before given height.
func (hs *historyStorage) freshEntryDataBeforeHeight(key []byte, height uint64, filter bool) ([]byte, error) {
	limitBlockNum, err := hs.stateDB.newestBlockNumByHeight(height)
	if err != nil {
		return nil, err
	}
	history, err := hs.fullHistory(key, filter)
	if err != nil {
		return nil, err
	}
	var res historyEntry
	for _, entry := range history.entries {
		if entry.blockNum < limitBlockNum {
			res = entry
		} else {
			break
		}
	}
	return res.data, nil
}

func (hs *historyStorage) entriesDataInHeightRangeCommon(history *historyRecord, startBlockNum, endBlockNum uint32) [][]byte {
	var entriesData [][]byte
	for i := len(history.entries) - 1; i >= 0; i-- {
		entry := history.entries[i]
		if entry.blockNum > endBlockNum {
			continue
		}
		if entry.blockNum < startBlockNum {
			break
		}
		entriesData = append(entriesData, entry.data)
	}
	return entriesData
}

func (hs *historyStorage) entriesDataInHeightRangeStable(key []byte, startHeight, endHeight uint64, filter bool) ([][]byte, error) {
	history, err := hs.getHistory(key, filter, false)
	if err != nil {
		return nil, err
	}
	if len(history.entries) == 0 {
		return nil, nil
	}
	startBlockNum, err := hs.stateDB.blockNumByHeight(startHeight)
	if err != nil {
		return nil, err
	}
	endBlockNum, err := hs.stateDB.blockNumByHeight(endHeight)
	if err != nil {
		return nil, err
	}
	return hs.entriesDataInHeightRangeCommon(history, startBlockNum, endBlockNum), nil
}

// entriesDataInHeightRange() returns bytes of entries that fit into specified height interval.
func (hs *historyStorage) entriesDataInHeightRange(key []byte, startHeight, endHeight uint64, filter bool) ([][]byte, error) {
	history, err := hs.fullHistory(key, filter)
	if err != nil {
		return nil, err
	}
	if len(history.entries) == 0 {
		return nil, nil
	}
	startBlockNum, err := hs.stateDB.newestBlockNumByHeight(startHeight)
	if err != nil {
		return nil, err
	}
	endBlockNum, err := hs.stateDB.newestBlockNumByHeight(endHeight)
	if err != nil {
		return nil, err
	}
	return hs.entriesDataInHeightRangeCommon(history, startBlockNum, endBlockNum), nil
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

package state

import (
	"encoding/binary"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
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
	ordersVolume
	sponsorship
	dataEntry
	accountScript
	assetScript
	scriptBasicInfo
	accountScriptComplexity
	assetScriptComplexity
	rewardVotes
	blockReward
	invokeResult
	score
	stateHash
	hitSource
	feeDistr
	accountOriginalEstimatorVersion
)

type blockchainEntityProperties struct {
	needToFilter bool
	needToCut    bool

	fixedSize  bool
	recordSize int
}

// Note on size calculation.
// 1) For fixed size records. We add 4 bytes for storing block number to each record.
// 2) For variable size records record size counts of length of data, 4 bytes for storing block number and
//    4 bytes for storing each record length.

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
		fixedSize:    false,
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
	scriptBasicInfo: {
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
		fixedSize:    false,
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
	score: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	stateHash: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	hitSource: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   hitSourceSize + 4,
	},
	feeDistr: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	accountOriginalEstimatorVersion: {
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
	return len(he.data) + 4 // 4 bytes to store block number added here
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
			return nil, errors.Wrapf(errInvalidDataSize, "entity type %v", entityType)
		}
		for i := 1; i <= dataSize-recordSize; i += recordSize {
			var entry historyEntry
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil { // Returns only `errInvalidDataSize`
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
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil { // Returns only `errInvalidDataSize`
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
			totalSize += 4 // This is 4 bytes for storing each record size
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
	topEntry, err := hr.topEntry()
	if err != nil {
		return err
	}
	if topEntry.blockNum == entry.blockNum {
		// The block is the same, rewrite the last entry.
		hr.entries[len(hr.entries)-1] = entry
	} else {
		// Append new entry to the end.
		hr.entries = append(hr.entries, entry)
	}
	return nil
}

func (hr *historyRecord) topEntry() (historyEntry, error) {
	if len(hr.entries) < 1 {
		return historyEntry{}, errors.New("empty history")
	}
	return hr.entries[len(hr.entries)-1], nil
}

type topEntryIterator struct {
	dbIter keyvalue.Iterator
	fmt    *historyFormatter
	amend  bool

	err    error
	curKey []byte
	curVal []byte
}

func (i *topEntryIterator) Next() bool {
	for i.dbIter.Next() {
		historyBytes := i.dbIter.Value()
		history, err := newHistoryRecordFromBytes(historyBytes)
		if err != nil {
			i.err = err
			return false
		}
		if _, err := i.fmt.normalize(history, i.amend); err != nil {
			i.err = err
			return false
		}
		if len(history.entries) == 0 {
			continue
		}
		topEntry, err := history.topEntry()
		if err != nil {
			i.err = err
			return false
		}
		i.curKey = i.dbIter.Key()
		i.curVal = topEntry.data
		return true
	}
	return false
}

func (i *topEntryIterator) Release() {
	i.dbIter.Release()
}

func (i *topEntryIterator) Error() error {
	if i.err != nil {
		return i.err
	}
	return i.dbIter.Error()
}

func (i *topEntryIterator) Key() []byte {
	return i.curKey
}

func (i *topEntryIterator) Value() []byte {
	return i.curVal
}

type newestTopEntryIterator struct {
	entity blockchainEntity

	hsEntries   []history
	hsPos       int
	visitedKeys map[string]bool

	dbIter *topEntryIterator

	err    error
	curKey []byte
	curVal []byte
}

func (i *newestTopEntryIterator) Next() bool {
	// Iterate in-mem history until the end or first entity of the type we need.
	for i.hsPos < len(i.hsEntries) {
		history := i.hsEntries[i.hsPos].value
		key := i.hsEntries[i.hsPos].key
		i.hsPos++
		if history.entityType != i.entity {
			continue
		}
		i.visitedKeys[string(key)] = true
		topEntry, err := history.topEntry()
		if err != nil {
			i.err = err
			return false
		}
		i.curKey = key
		i.curVal = topEntry.data
		return true
	}
	// Iterate db until the end or first unvisited key.
	for i.dbIter.Next() {
		key := i.dbIter.Key()
		if i.visitedKeys[string(key)] {
			continue
		}
		i.curKey = key
		i.curVal = i.dbIter.Value()
		return true
	}
	return false
}

func (i *newestTopEntryIterator) Value() []byte {
	return i.curVal
}

func (i *newestTopEntryIterator) Key() []byte {
	return i.curKey
}

func (i *newestTopEntryIterator) Error() error {
	return i.dbIter.Error()
}

func (i *newestTopEntryIterator) Release() {
	i.dbIter.Release()
}

// historyStorage manages the way per-block records are stored in.
// Unlike blockchain entities parts, it does not know *what* it stores, but it does know *how*.
type historyStorage struct {
	db        keyvalue.IterableKeyVal
	dbBatch   keyvalue.Batch
	writeLock *sync.Mutex
	stateDB   *stateDB
	amend     bool // if true, the records will be filtered which is important after rollback
	stor      *localHistoryStorage
	fmt       *historyFormatter
}

func newHistoryStorage(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	stateDB *stateDB,
	amend bool,
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
		amend:     amend,
	}, nil
}

func (hs *historyStorage) newTopEntryIteratorByPrefix(prefix []byte) (*topEntryIterator, error) {
	dbIter, err := hs.db.NewKeyIterator(prefix)
	if err != nil {
		return nil, err
	}
	return &topEntryIterator{dbIter: dbIter, fmt: hs.fmt, amend: hs.amend}, nil
}

func (hs *historyStorage) newTopEntryIterator(entity blockchainEntity) (*topEntryIterator, error) {
	prefix, err := prefixByEntity(entity)
	if err != nil {
		return nil, err
	}
	return hs.newTopEntryIteratorByPrefix(prefix)
}

func (hs *historyStorage) newNewestTopEntryIterator(entity blockchainEntity) (*newestTopEntryIterator, error) {
	i := &newestTopEntryIterator{entity: entity}
	i.hsEntries = hs.stor.getEntries()
	dbIter, err := hs.newTopEntryIterator(entity)
	if err != nil {
		return nil, err
	}
	i.dbIter = dbIter
	i.visitedKeys = make(map[string]bool)
	return i, nil
}

func (hs *historyStorage) addNewEntry(entityType blockchainEntity, key, value []byte, blockID proto.BlockID) error {
	blockNum, err := hs.stateDB.newestBlockIdToNum(blockID)
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
func (hs *historyStorage) getHistory(key []byte, update bool) (*historyRecord, error) {
	// Lock the write lock.
	// It is necessary because if we read value *before* the main write batch is written,
	// and manageDbUpdate() happens *after* it is written,
	// we might rewrite some keys that were in the batch.
	// So we do both read and write under same lock.
	hs.writeLock.Lock()
	defer hs.writeLock.Unlock()

	historyBytes, err := hs.db.Get(key)
	if err != nil {
		return nil, err // `keyvalue.ErrNotFound` is possible here along with other unwrapped DB errors
	}
	history, err := newHistoryRecordFromBytes(historyBytes) // Size check and binary errors here
	if err != nil {
		return nil, errs.Extend(err, "newHistoryRecordFromBytes")
	}
	changed, err := hs.fmt.normalize(history, hs.amend)
	if err != nil {
		return nil, err
	}
	if changed && update {
		if err := hs.manageDbUpdate(key, history); err != nil {
			return nil, errs.Extend(err, "manageDbUpdate")
		}
	}
	if len(history.entries) == 0 {
		return nil, errEmptyHist
	}
	return history, nil
}

func (hs *historyStorage) topEntry(key []byte) (historyEntry, error) {
	history, err := hs.getHistory(key, false)
	if err != nil {
		return historyEntry{}, err // keyvalue.ErrNotFoundHere
	}
	return history.topEntry() // untyped error "empty history" here
}

func (hs *historyStorage) newestTopEntry(key []byte) (historyEntry, error) {
	if newHist, err := hs.stor.get(key); err == nil {
		return newHist.topEntry()
	}
	return hs.topEntry(key)
}

func (hs *historyStorage) combineHistories(key []byte, newHist *historyRecord) (*historyRecord, error) {
	prevHist, err := hs.getHistory(key, true)
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
func (hs *historyStorage) fullHistory(key []byte) (*historyRecord, error) {
	newHist, err := hs.stor.get(key)
	if err == errNotFound {
		return hs.getHistory(key, true)
	} else if err != nil {
		return nil, err
	}
	return hs.combineHistories(key, newHist)
}

// topEntryData() returns bytes of the top entry.
func (hs *historyStorage) topEntryData(key []byte) ([]byte, error) {
	entry, err := hs.topEntry(key)
	if err != nil {
		return nil, err
	}
	return entry.data, nil
}

// newestTopEntryData() returns bytes of the top entry from local storage or DB.
func (hs *historyStorage) newestTopEntryData(key []byte) ([]byte, error) {
	entry, err := hs.newestTopEntry(key)
	if err != nil {
		return nil, err
	}
	return entry.data, nil
}

// newestBlockOfTheTopEntry() returns block ID of the top entry from local storage or DB.
func (hs *historyStorage) newestBlockOfTheTopEntry(key []byte) (proto.BlockID, error) {
	entry, err := hs.newestTopEntry(key)
	if err != nil {
		return proto.BlockID{}, err
	}
	return hs.stateDB.newestBlockNumToId(entry.blockNum)
}

// blockOfTheTopEntry() returns block ID of the top entry from DB.
func (hs *historyStorage) blockOfTheTopEntry(key []byte) (proto.BlockID, error) {
	entry, err := hs.topEntry(key)
	if err != nil {
		return proto.BlockID{}, err
	}
	return hs.stateDB.blockNumToId(entry.blockNum)
}

type entryNumsCmp func(uint32, uint32) bool

func (hs *historyStorage) entryDataWithHeightFilter(
	key []byte,
	limitHeight uint64,
	cmp entryNumsCmp,
) ([]byte, error) {
	limitBlockNum, err := hs.stateDB.blockNumByHeight(limitHeight)
	if err != nil {
		return nil, err
	}
	history, err := hs.getHistory(key, false)
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

func (hs *historyStorage) entryDataAtHeight(key []byte, height uint64) ([]byte, error) {
	cmp := func(entryNum, limitNum uint32) bool {
		return entryNum <= limitNum
	}
	return hs.entryDataWithHeightFilter(key, height, cmp)
}

// blockRangeEntries() returns list of entries corresponding to given block interval.
// IMPORTANTLY, it does not simply return list of entries with block nums between startBlockNum and endBlockNum,
// instead this function returns values which are relevant for this block range.
// This actually means that it MIGHT include entries BEFORE startBlockNum, because they are relevant for the range start.
// For example, if the first entry in the range is at startBlockNum + 1, then at startBlockNum we should use the value
// from the past.
func (hs *historyStorage) blockRangeEntries(history *historyRecord, startBlockNum, endBlockNum uint32) [][]byte {
	var records [][]byte
	startPos := 0
	for i, entry := range history.entries {
		if entry.blockNum > endBlockNum {
			break
		}
		records = append(records, entry.data)
		if entry.blockNum <= startBlockNum {
			startPos = i
		}
	}
	return records[startPos:]
}

// entriesDataInHeightRange() returns bytes of entries that fit into specified height interval.
// WARNING: see comment about blockRangeEntries() to understand how this function actually works.
func (hs *historyStorage) entriesDataInHeightRange(key []byte, startHeight, endHeight uint64) ([][]byte, error) {
	history, err := hs.getHistory(key, false)
	if err != nil {
		return nil, errs.Extend(err, "getHistory")
	}
	if len(history.entries) == 0 {
		return nil, nil
	}
	startBlockNum, err := hs.stateDB.blockNumByHeight(startHeight)
	if err != nil {
		return nil, errs.Extend(err, "blockNumByHeight")
	}
	endBlockNum, err := hs.stateDB.blockNumByHeight(endHeight)
	if err != nil {
		return nil, errs.Extend(err, "blockNumByHeight")
	}
	return hs.blockRangeEntries(history, startBlockNum, endBlockNum), nil
}

// WARNING: see comment about blockRangeEntries() to understand how this function actually works.
func (hs *historyStorage) newestEntriesDataInHeightRange(key []byte, startHeight, endHeight uint64) ([][]byte, error) {
	history, err := hs.fullHistory(key)
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
	return hs.blockRangeEntries(history, startBlockNum, endBlockNum), nil
}

func (hs *historyStorage) reset() {
	hs.stor.reset()
}

func (hs *historyStorage) flush() error {
	entries := hs.stor.getEntries()
	sortEntries(entries)
	for _, entry := range entries {
		newEntry, err := hs.combineHistories(entry.key, entry.value)
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

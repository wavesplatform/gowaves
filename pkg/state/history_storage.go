package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
	accountScriptComplexity
	assetScriptComplexity
	rewardVotes
	blockReward
)

// + 4 bytes for blockNum at the end of each record.
var recordSizes = map[blockchainEntity]int{
	alias:                 aliasRecordSize + 4,
	asset:                 assetRecordSize + 4,
	lease:                 leasingRecordSize + 4,
	wavesBalance:          wavesBalanceRecordSize + 4,
	assetBalance:          assetBalanceRecordSize + 4,
	featureVote:           votesFeaturesRecordSize + 4,
	approvedFeature:       approvedFeaturesRecordSize + 4,
	activatedFeature:      activatedFeaturesRecordSize + 4,
	sponsorship:           sponsorshipRecordSize + 4,
	assetScriptComplexity: assetScriptComplexityRecordSize + 4,
	rewardVotes:           rewardVotesRecordSize + 4,
	blockReward:           blockRewardRecordSize + 4,
	// TODO: uncomment when changing state structure next time.
	//ordersVolume:        orderVolumeRecordSize + 4,
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
		return errors.New("invalid data size")
	}
	he.data = make([]byte, len(data)-4)
	copy(he.data, data[:len(data)-4])
	he.blockNum = binary.BigEndian.Uint32(data[len(data)-4:])
	return nil
}

type historyRecord struct {
	fixedSize bool
	// recordSize is specified if fixedSize is true.
	// Otherwise entries sizes are 4 first bytes of each record.
	recordSize uint32
	entries    []historyEntry
}

func newHistoryRecord(entityType blockchainEntity) (*historyRecord, error) {
	fixedSize := true
	size, ok := recordSizes[entityType]
	if !ok {
		fixedSize = false
	}
	return &historyRecord{fixedSize: fixedSize, recordSize: uint32(size)}, nil
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
	var entries []historyEntry
	if fixedSize {
		if dataSize < 5 {
			return nil, errors.New("invalid data size")
		}
		recordSize = binary.BigEndian.Uint32(data[1:5])
		if dataSize < 5+recordSize {
			return nil, errors.New("invalid data size")
		}
		for i := uint32(5); i <= dataSize-recordSize; i += recordSize {
			var entry historyEntry
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
	} else {
		for i := uint32(1); i <= dataSize-4; {
			recordSize := binary.BigEndian.Uint32(data[i : i+4])
			i += 4
			if dataSize < i+recordSize {
				return nil, errors.New("invalid data size")
			}
			var entry historyEntry
			if err := entry.unmarshalBinary(data[i : i+recordSize]); err != nil {
				return nil, err
			}
			entries = append(entries, entry)
			i += recordSize
		}
	}
	return &historyRecord{fixedSize, recordSize, entries}, nil
}

func (hr *historyRecord) countTotalSize() int {
	totalSize := 1
	if hr.fixedSize {
		totalSize += 4
	}
	for _, r := range hr.entries {
		totalSize += r.size()
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
		// Add size of all entries.
		binary.BigEndian.PutUint32(data[curPos:curPos+4], hr.recordSize)
		curPos += 4
	}
	for _, entry := range hr.entries {
		if !hr.fixedSize {
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
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	stateDB *stateDB

	stor *localHistoryStorage
	fmt  *historyFormatter
}

func newHistoryStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, stateDB *stateDB) (*historyStorage, error) {
	stor, err := newLocalHistoryStorage()
	if err != nil {
		return nil, err
	}
	fmt, err := newHistoryFormatter(stateDB)
	if err != nil {
		return nil, err
	}
	return &historyStorage{
		db:      db,
		dbBatch: dbBatch,
		stateDB: stateDB,
		stor:    stor,
		fmt:     fmt,
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
		if history, err = newHistoryRecord(entityType); err != nil {
			return err
		}
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

func (hs *historyStorage) cleanDbRecord(key []byte) error {
	// If the history is empty after normalizing, it means that all the entries were removed due to rollback.
	// In this case, it should be removed from the DB as well.
	return hs.db.Delete(key)
}

// getHistory() retrieves history record from DB. It also normalizes it,
// saving the result back to DB, if update argument is true.
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
	if len(history.entries) == 0 {
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
	if prevHist.fixedSize != newHist.fixedSize {
		return nil, errors.New("trying to combine incompatible histories")
	}
	if prevHist.recordSize != newHist.recordSize {
		return nil, errors.New("trying to combine incompatible histories")
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

// blockOfTheLatestEntry() returns block ID of the latest entry from DB.
func (hs *historyStorage) blockOfTheLatestEntry(key []byte, filter bool) (crypto.Signature, error) {
	entry, err := hs.latestEntry(key, filter)
	if err != nil {
		return crypto.Signature{}, err
	}
	return hs.stateDB.blockNumToId(entry.blockNum)
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

// entriesDataInHeightRange() returns bytes of entries that fit into specified height interval.
func (hs *historyStorage) entriesDataInHeightRange(key []byte, startHeight, endHeight uint64, filter bool) ([][]byte, error) {
	history, err := hs.fullHistory(key, filter)
	if err != nil {
		return nil, err
	}
	if (len(history.entries)) == 0 {
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
	return entriesData, nil
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

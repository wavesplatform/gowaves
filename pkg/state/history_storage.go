package state

import (
	"bytes"
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
	_ blockchainEntity = iota // this is a placeholder for a zero value
	alias
	addressToAliases
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
	rewardChanges
	invokeResult
	score
	legacyStateHash
	snapshotStateHash
	hitSource
	feeDistr
	snapshots
	patches
	challengedAddress
	commitment
	finalization
	generators
)

type blockchainEntityProperties struct {
	needToFilter bool
	needToCut    bool

	fixedSize  bool
	recordSize int
}

// Note on size calculation.
// 1) For fixed size records. We add 4 bytes for storing block number to each record.
// 2) For variable size records record size consists of length of data, 4 bytes for storing block number and
//    the uvarint encoded size of each record (the length of data plus 4 bytes of block number) before it.
//    Up to state version 27 the size of each record was stored as fixed 4 bytes big-endian uint32,
//    see migration_v27.go.

// properties holds the properties of each blockchain entity, indexed by the entity type.
// Use entityProperties to read it, it rejects unknown entity types.
// + 4 bytes for blockNum at the end of each record.
var properties = [...]blockchainEntityProperties{ //nolint:gochecknoglobals // Read-only table of properties.
	alias: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    true,
		recordSize:   aliasRecordSize + 4,
	},
	addressToAliases: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
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
		fixedSize:    false,
	},
	assetBalance: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
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
	legacyStateHash: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	snapshotStateHash: {
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
	rewardChanges: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	snapshots: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	patches: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	challengedAddress: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	commitment: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	finalization: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
	generators: {
		needToFilter: true,
		needToCut:    true,
		fixedSize:    false,
	},
}

// entityProperties returns the properties of the given entity type.
// It returns false for unknown entity types, including the zero placeholder and types without properties.
func entityProperties(entity blockchainEntity) (blockchainEntityProperties, bool) {
	if int(entity) >= len(properties) {
		return blockchainEntityProperties{}, false
	}
	p := properties[entity]
	// A zero value means that the properties for the entity are not defined.
	// No defined entity has zero properties, because each of them needs to be filtered.
	if p == (blockchainEntityProperties{}) {
		return blockchainEntityProperties{}, false
	}
	return p, true
}

type historyEntry struct {
	data     []byte
	blockNum uint32
}

func (he *historyEntry) size() int {
	return len(he.data) + 4 // 4 bytes to store block number added here
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

// historyRecordWithEntry keeps a record together with the storage for its first entry,
// so a record with a single entry, which is the most common case, takes one allocation instead of two.
type historyRecordWithEntry struct {
	record historyRecord
	first  [1]historyEntry
}

// newHistoryRecordWithLen returns a record with n zero entries.
// The entries of a record with at most one entry are allocated together with the record.
func newHistoryRecordWithLen(entityType blockchainEntity, n int) *historyRecord {
	if n > 1 {
		return &historyRecord{entityType: entityType, entries: make([]historyEntry, n)}
	}
	r := &historyRecordWithEntry{record: historyRecord{entityType: entityType}}
	r.record.entries = r.first[:n:1] // The capacity limit makes a second append reallocate.
	return &r.record
}

func newHistoryRecord(entityType blockchainEntity) *historyRecord {
	return newHistoryRecordWithLen(entityType, 0)
}

// newHistoryRecordFromBytes decodes the history record from the given bytes.
// The data of decoded entries points into the given slice, so the caller must not modify or reuse it while the
// record is in use. Values returned by keyvalue.KeyVal.Get are owned by the caller, iterator values are not and
// must be copied before decoding.
func newHistoryRecordFromBytes(data []byte) (*historyRecord, error) {
	if len(data) < 1 {
		return nil, errors.Wrap(errInvalidDataSize, "data too short")
	}
	entityType := blockchainEntity(data[0])
	property, ok := entityProperties(entityType)
	if !ok {
		return nil, errors.Errorf("bad entity type %v", entityType)
	}
	if property.fixedSize {
		n := (len(data) - 1) / property.recordSize
		if n == 0 {
			return nil, errors.Wrapf(errInvalidDataSize,
				"fixed size record too short (expected %d, actual %d), entity type %v",
				property.recordSize, len(data)-1, entityType)
		}
		hr := newHistoryRecordWithLen(entityType, n)
		decodeFixedSizeEntries(hr.entries, data, property.recordSize)
		return hr, nil
	}
	n, err := variableSizeEntriesCount(data)
	if err != nil {
		return nil, errors.Wrapf(err, "entity type %v", entityType)
	}
	if n == 0 {
		return &historyRecord{entityType: entityType}, nil
	}
	hr := newHistoryRecordWithLen(entityType, n)
	decodeVariableSizeEntries(hr.entries, data)
	return hr, nil
}

// decodeFixedSizeEntries decodes len(entries) entries of the fixed size record into entries.
// The data must hold at least len(entries) complete entries after the entity type byte.
// Trailing bytes that do not form a complete entry are ignored.
func decodeFixedSizeEntries(entries []historyEntry, data []byte, recordSize int) {
	for i, pos := 0, 1; i < len(entries); i, pos = i+1, pos+recordSize {
		end := pos + recordSize
		dataEnd := end - historyBlockNumSize
		entries[i] = historyEntry{
			data:     data[pos:dataEnd:dataEnd],
			blockNum: binary.BigEndian.Uint32(data[dataEnd:end]),
		}
	}
}

const (
	// uvarintContinuationBit is set in every byte of the uvarint encoding except the last one.
	uvarintContinuationBit = 0x80
	// uvarintPayloadMask selects the seven payload bits of a byte of the uvarint encoding.
	uvarintPayloadMask = 0x7f
	// uvarintPayloadBits is the number of payload bits in a byte of the uvarint encoding.
	uvarintPayloadBits = 7
)

// variableSizeEntriesCount checks the layout of variable size record and returns the number of entries in it.
// The layout is checked strictly: each entry must be preceded by its uvarint encoded size, which must be
// in canonical (shortest) form and cover at least the block number, and the entries must end exactly
// at the end of the data.
func variableSizeEntriesCount(data []byte) (int, error) {
	// The loop decodes sizes of one and two bytes, which cover entries up to 16 KiB, inline and has no calls on
	// the happy path. A longer size continues the count with the generic loop.
	n := 0
	for i := 1; i < len(data); n++ {
		recordSize, sizeLen := uint64(data[i]), 1
		if recordSize >= uvarintContinuationBit {
			// Not the canonical size of two bytes.
			if i+1 >= len(data) || data[i+1]-1 >= uvarintPayloadMask { //nolint:gosec // i+1 < len(data) is checked.
				return variableSizeEntriesCountFrom(data, i, n)
			}
			//nolint:gosec // i+1 < len(data) is checked above.
			recordSize, sizeLen = recordSize&uvarintPayloadMask|uint64(data[i+1])<<uvarintPayloadBits, 2
		}
		i += sizeLen
		// i <= len(data) here, the difference is not negative.
		if uint64(len(data)-i) < recordSize || recordSize < historyBlockNumSize { // #nosec G115 // Not negative.
			return 0, invalidEntrySizeError(recordSize, len(data)-i)
		}
		i += int(recordSize) // #nosec G115 // recordSize is not greater than len(data) here.
	}
	return n, nil
}

// variableSizeEntriesCountFrom continues the count of entries of variable size record from position i, where
// n entries are already counted.
func variableSizeEntriesCountFrom(data []byte, i, n int) (int, error) {
	for ; i < len(data); n++ {
		recordSize, sizeLen := readEntrySize(data[i:])
		if sizeLen == 0 {
			return 0, errors.Wrap(errInvalidDataSize, "invalid size of variable size record")
		}
		i += sizeLen
		// i <= len(data) here, the difference is not negative.
		if uint64(len(data)-i) < recordSize || recordSize < historyBlockNumSize { // #nosec G115 // Not negative.
			return 0, invalidEntrySizeError(recordSize, len(data)-i)
		}
		i += int(recordSize) // #nosec G115 // recordSize is not greater than len(data) here.
	}
	return n, nil
}

func invalidEntrySizeError(recordSize uint64, available int) error {
	if recordSize < historyBlockNumSize {
		return errors.Wrapf(errInvalidDataSize, "variable size record size %d is too small", recordSize)
	}
	return errors.Wrapf(errInvalidDataSize,
		"variable size record too short (expected %d, actual %d)", recordSize, available)
}

// readEntrySize reads the uvarint encoded size of the history entry from the beginning of b.
// It returns the size and the number of bytes read, or zero number of bytes if the size is truncated, overflows
// uint64, or is not in the canonical (shortest) form. It is used only for sizes that the inline loops of
// variableSizeEntriesCount and decodeVariableSizeEntries do not handle.
func readEntrySize(b []byte) (uint64, int) {
	v, n := binary.Uvarint(b)
	if n <= 0 || (n > 1 && b[n-1] == 0) { // the last byte of the canonical encoding longer than one byte is not zero
		return 0, 0
	}
	return v, n
}

// uvarintSize returns the number of bytes of the uvarint encoding of v.
func uvarintSize(v uint64) int {
	n := 1
	for v >= uvarintContinuationBit {
		v >>= uvarintPayloadBits
		n++
	}
	return n
}

// decodeVariableSizeEntries decodes len(entries) entries of the variable size record into entries.
// The layout of the data must be checked with variableSizeEntriesCount beforehand.
func decodeVariableSizeEntries(entries []historyEntry, data []byte) {
	// Sizes of one and two bytes are decoded inline, a longer size continues with the generic loop.
	pos := 1
	for i := range entries {
		size, sizeLen := uint64(data[pos]), 1
		if size >= uvarintContinuationBit {
			if data[pos+1] >= uvarintContinuationBit {
				decodeVariableSizeEntriesFrom(entries[i:], data, pos)
				return
			}
			size, sizeLen = size&uvarintPayloadMask|uint64(data[pos+1])<<uvarintPayloadBits, 2
		}
		pos += sizeLen
		end := pos + int(size) // #nosec G115 // checked by variableSizeEntriesCount.
		dataEnd := end - historyBlockNumSize
		entries[i] = historyEntry{
			data:     data[pos:dataEnd:dataEnd],
			blockNum: binary.BigEndian.Uint32(data[dataEnd:end]),
		}
		pos = end
	}
}

// decodeVariableSizeEntriesFrom decodes entries of the variable size record starting at position pos.
func decodeVariableSizeEntriesFrom(entries []historyEntry, data []byte, pos int) {
	for i := range entries {
		size, sizeLen := readEntrySize(data[pos:])
		pos += sizeLen
		end := pos + int(size) // #nosec G115 // checked by variableSizeEntriesCount.
		dataEnd := end - historyBlockNumSize
		entries[i] = historyEntry{
			data:     data[pos:dataEnd:dataEnd],
			blockNum: binary.BigEndian.Uint32(data[dataEnd:end]),
		}
		pos = end
	}
}

func (hr *historyRecord) fixedSize() (bool, error) {
	property, ok := entityProperties(hr.entityType)
	if !ok {
		return false, errors.Errorf("bad entity type %v", hr.entityType)
	}
	return property.fixedSize, nil
}

func (hr *historyRecord) countTotalSize() (int, error) {
	fixedSize, err := hr.fixedSize()
	if err != nil {
		return 0, err
	}
	return 1 + entriesSize(hr.entries, fixedSize), nil
}

// entriesSize returns the encoded size of the entries without the entity type byte.
func entriesSize(entries []historyEntry, fixedSize bool) int {
	size := 0
	for i := range entries {
		entrySize := entries[i].size()
		size += entrySize
		if !fixedSize {
			size += uvarintSize(uint64(entrySize)) // #nosec G115 // Encoded size of the record, always positive.
		}
	}
	return size
}

func (hr *historyRecord) marshalBinary() ([]byte, error) {
	return marshalHistoryInto(nil, hr.entityType, hr.entries, nil)
}

// marshalHistoryInto encodes a history record of the given type that holds the entries of head followed by
// the entries of tail. It produces the same bytes as marshalBinary of a record with the combined entries,
// without building the combined entries slice. The encoding goes into buf, which grows if needed.
func marshalHistoryInto(buf []byte, entityType blockchainEntity, head, tail []historyEntry) ([]byte, error) {
	property, ok := entityProperties(entityType)
	if !ok {
		return nil, errors.Errorf("bad entity type %v", entityType)
	}
	fixedSize := property.fixedSize
	size := 1 + entriesSize(head, fixedSize) + entriesSize(tail, fixedSize)
	var data []byte
	if cap(buf) >= size {
		data = buf[:size]
	} else {
		data = make([]byte, size)
	}
	data[0] = byte(entityType)
	curPos := putEntries(data, 1, head, fixedSize)
	putEntries(data, curPos, tail, fixedSize)
	return data, nil
}

// putEntries writes the entries into data starting at pos and returns the position after the last entry.
func putEntries(data []byte, pos int, entries []historyEntry, fixedSize bool) int {
	for i := range entries {
		entry := &entries[i]
		if !fixedSize {
			// Add size of this record.
			if size := entry.size(); size < uvarintContinuationBit {
				data[pos] = byte(size) // #nosec G115 // size < 0x80 is checked.
				pos++
			} else {
				pos += binary.PutUvarint(data[pos:], uint64(size)) // #nosec G115 // Always positive.
			}
		}
		pos += copy(data[pos:], entry.data)
		binary.BigEndian.PutUint32(data[pos:pos+historyBlockNumSize], entry.blockNum)
		pos += historyBlockNumSize
	}
	return pos
}

func (hr *historyRecord) appendEntry(entry historyEntry) error {
	if n := len(hr.entries); n > 0 && hr.entries[n-1].blockNum == entry.blockNum {
		// The block is the same, rewrite the last entry.
		hr.entries[n-1] = entry
		return nil
	}
	// The history is empty or the block is new, append new entry to the end.
	hr.entries = append(hr.entries, entry)
	return nil
}

func (hr *historyRecord) topEntry() (historyEntry, error) {
	if len(hr.entries) == 0 {
		return historyEntry{}, errEmptyHist
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
		// Decoded entries point into the iterator buffer, which is reused on the next move.
		// The history is used only within this call, and the returned top entry data is copied below.
		history, err := newHistoryRecordFromBytes(i.dbIter.Value())
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
		i.curVal = bytes.Clone(topEntry.data) // Copy once, the value must stay valid after the iterator moves.
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
) *historyStorage {
	stor := newLocalHistoryStorage()
	fmt := newHistoryFormatter(stateDB)
	return &historyStorage{
		db:        db,
		dbBatch:   dbBatch,
		writeLock: stateDB.retrieveWriteLock(),
		stateDB:   stateDB,
		stor:      stor,
		fmt:       fmt,
		amend:     amend,
	}
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
	if errors.Is(err, errNotFound) {
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
//
// Read-path audit (parallel invoke execution): this is the only place where a read may write to the DB.
// With update set (fullHistory, reached from RIDE through newestEntriesDataInHeightRange when the generating
// balance is calculated) a history that was changed by normalization is saved back. The read and the write
// are done under writeLock and the saved record holds the same valid entries, so concurrent readers are safe
// and observe equivalent data. normalize itself modifies only the freshly decoded record, which is private
// to the call; the block number caches it consults (stateDB.minAcceptableBlockNum, validBlockNumWatermark)
// are guarded by their own mutexes.
// Note that every DB read of a history takes writeLock, so concurrent readers are serialised here.
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
		return historyEntry{}, err // keyvalue.ErrNotFound here
	}
	return history.topEntry() // errEmptyHist here
}

// newestTopEntry returns the top entry from the local storage or DB.
// Read-path audit (parallel invoke execution): the local storage is only read here (a map lookup), it is written
// by the appender goroutine only. The returned entry data aliases the stored record, callers must not modify it.
func (hs *historyStorage) newestTopEntry(key []byte) (historyEntry, error) {
	if newHist, err := hs.stor.get(key); err == nil {
		return newHist.topEntry()
	}
	return hs.topEntry(key)
}

func (hs *historyStorage) combineHistories(key []byte, newHist *historyRecord) (*historyRecord, error) {
	prevHist, err := hs.previousHistory(key, newHist)
	if err != nil {
		return nil, err
	}
	if prevHist == nil {
		// New history.
		return newHist, nil
	}
	prevHist.entries = append(prevHist.entries, newHist.entries...)
	return prevHist, nil
}

// previousHistory returns the normalized history stored in DB under the key, saving the normalized history back
// to DB if it changed. It returns nil if there is no stored history or it is empty after normalization.
// The stored history must have the same entity type as newHist.
func (hs *historyStorage) previousHistory(key []byte, newHist *historyRecord) (*historyRecord, error) {
	prevHist, err := hs.getHistory(key, true)
	if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
		return nil, nil //nolint:nilnil // No previous history is a valid result, see the function comment.
	} else if err != nil {
		return nil, err
	}
	if prevHist.entityType != newHist.entityType {
		return nil, errors.Errorf("trying to combine histories of different types %v and %v", prevHist.entityType, newHist.entityType)
	}
	return prevHist, nil
}

// fullHistory() returns combination of history from DB and the local storage (if any).
func (hs *historyStorage) fullHistory(key []byte) (*historyRecord, error) {
	newHist, err := hs.stor.get(key)
	if errors.Is(err, errNotFound) {
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
	// The batch copies values on Put, so a single encoding buffer is reused for all records.
	var buf []byte
	for _, entry := range entries {
		prevHist, err := hs.previousHistory(entry.key, entry.value)
		if err != nil {
			return err
		}
		// Encode the stored entries followed by the new ones, it is the same as encoding the combined history.
		var prevEntries []historyEntry
		if prevHist != nil {
			prevEntries = prevHist.entries
		}
		buf, err = marshalHistoryInto(buf, entry.value.entityType, prevEntries, entry.value.entries)
		if err != nil {
			return err
		}
		hs.dbBatch.Put(entry.key, buf)
	}
	hs.stor.reset()
	return nil
}

// isNotFoundInHistoryOrDBErr checks if the error is errEmptyHist or keyvalue.ErrNotFound.
// TODO: Should be renamed to isNotFoundInHistoryOrEmptyHistory.
func isNotFoundInHistoryOrDBErr(err error) bool {
	return errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist)
}

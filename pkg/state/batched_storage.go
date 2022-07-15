package state

// batched_storage.go - stores arbitrary data for given key prefix, batching it in a way
// that no single value in db is larger than specified `batchSize` in bytes.
// data is sequence of records of similar size, batchedStorage also provides iterators
// to move through records, in range from most recent to oldest.

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

const (
	firstBatchNum = 0
	blockNumLen   = 4 // 4 bytes for block number.
)

type record struct {
	blockNum uint32
	data     []byte
}

func newRecordFromBytes(data []byte) (*record, error) {
	if len(data) < blockNumLen {
		return nil, errInvalidDataSize
	}
	blockNum := binary.BigEndian.Uint32(data[:blockNumLen])
	return &record{blockNum: blockNum, data: data[blockNumLen:]}, nil
}

func (r *record) marshalBinary() []byte {
	buf := make([]byte, blockNumLen+len(r.data))
	binary.BigEndian.PutUint32(buf, r.blockNum)
	copy(buf[blockNumLen:], r.data)
	return buf
}

func (r *record) recordBytes() []byte {
	return r.data
}

type recordIterator struct {
	iter       *batchIterator
	batch      []byte
	recordSize int
	err        error
}

func newRecordIterator(iter *batchIterator, recordSize int) *recordIterator {
	return &recordIterator{iter: iter, recordSize: recordSize}
}

func (i *recordIterator) loadNextBatch() bool {
	for {
		if !i.iter.next() {
			return false
		}
		batch, err := i.iter.currentBatch()
		if err != nil {
			i.err = err
			return false
		}
		if len(batch) == 0 {
			// We need to find first not empty batch.
			continue
		}
		i.batch = batch
		return true
	}
}

func (i *recordIterator) next() bool {
	size := i.recordSize
	if len(i.batch) > size {
		i.batch = i.batch[:len(i.batch)-size]
		return true
	}
	return i.loadNextBatch()
}

func (i *recordIterator) currentRecord() ([]byte, error) {
	size := i.recordSize
	if len(i.batch) < size {
		return nil, errInvalidDataSize
	}
	recordBytes := i.batch[len(i.batch)-size:]
	r, err := newRecordFromBytes(recordBytes)
	if err != nil {
		i.err = err
		return nil, err
	}
	return r.recordBytes(), nil
}

func (i *recordIterator) error() error {
	if err := i.iter.error(); err != nil {
		return err
	}
	return i.err
}

func (i *recordIterator) release() {
	i.batch = nil
	i.iter.release()
}

type batchIterator struct {
	stor *batchedStorage
	iter keyvalue.Iterator
	used bool
}

func newBatchIterator(stor *batchedStorage, iter keyvalue.Iterator) *batchIterator {
	return &batchIterator{stor, iter, false}
}

func (i *batchIterator) next() bool {
	if i.used {
		return i.iter.Prev()
	}
	i.used = true
	return i.iter.Last()
}

func (i *batchIterator) currentBatch() ([]byte, error) {
	val := keyvalue.SafeValue(i.iter)
	return i.stor.normalize(val)
}

func (i *batchIterator) error() error {
	return i.iter.Error()
}

func (i *batchIterator) release() {
	i.iter.Release()
}

type batch struct {
	pos        int
	data       []byte
	num        uint32
	recordSize int
}

func newBatchWithData(data []byte, maxSize, recordSize int, batchNum uint32) (*batch, error) {
	if len(data) > maxSize {
		return nil, errInvalidDataSize
	}
	b := &batch{pos: len(data), num: batchNum, recordSize: recordSize}
	b.data = make([]byte, maxSize)
	copy(b.data, data)
	return b, nil
}

func newBatch(maxSize, recordSize int, batchNum uint32) *batch {
	return &batch{pos: 0, data: make([]byte, maxSize), num: batchNum, recordSize: recordSize}
}

func (b *batch) canAddRecord(record []byte) bool {
	return b.pos+len(record) <= len(b.data)
}

func (b *batch) addRecord(record []byte) {
	copy(b.data[b.pos:], record)
	b.pos += len(record)
}

func (b *batch) bytes() []byte {
	return b.data[:b.pos]
}

func (b *batch) lastRecord() (*record, error) {
	if b.pos < b.recordSize {
		return nil, errors.New("batch is too small")
	}
	recordBytes := b.data[b.pos-b.recordSize : b.pos]
	record, err := newRecordFromBytes(recordBytes)
	if err != nil {
		return nil, err
	}
	return record, nil
}

type batchesGroup struct {
	maxBatchSize int
	recordSize   int
	batches      []*batch
}

func newBatchesGroup(maxBatchSize, recordSize int) (*batchesGroup, error) {
	if recordSize > maxBatchSize {
		return nil, errors.New("recordSize is greater than maxBatchSize")
	}
	return &batchesGroup{
		maxBatchSize: maxBatchSize,
		recordSize:   recordSize,
	}, nil
}

func (bg *batchesGroup) initFirstBatch(first *batch) {
	bg.batches = make([]*batch, 1)
	bg.batches[0] = first
}

func (bg *batchesGroup) initFirstBatchEmpty() {
	bg.batches = make([]*batch, 1)
	bg.batches[0] = newBatch(bg.maxBatchSize, bg.recordSize, firstBatchNum)
}

func (bg *batchesGroup) appendNewRecord(record []byte) error {
	if len(record) != bg.recordSize {
		// Sanity check.
		return errInvalidDataSize
	}
	if len(bg.batches) == 0 {
		bg.initFirstBatchEmpty()
	}
	lastBatch := bg.batches[len(bg.batches)-1]
	if lastBatch.canAddRecord(record) {
		lastBatch.addRecord(record)
		return nil
	}
	if lastBatch.num == math.MaxUint32 {
		// Sanity check to prevent overflow.
		return errors.New("too many batches, can't add new!")
	}
	nextBatchNum := lastBatch.num + 1
	newBatch := newBatch(bg.maxBatchSize, bg.recordSize, nextBatchNum)
	newBatch.addRecord(record)
	bg.batches = append(bg.batches, newBatch)
	return nil
}

func (bg *batchesGroup) lastRecord() (*record, error) {
	if len(bg.batches) == 0 {
		return nil, errors.New("no batches")
	}
	lastBatch := bg.batches[len(bg.batches)-1]
	return lastBatch.lastRecord()
}

type batchedStorParams struct {
	maxBatchSize, recordSize int
	prefix                   byte
}

type batchedStorage struct {
	db        keyvalue.IterableKeyVal
	dbBatch   keyvalue.Batch
	writeLock *sync.Mutex
	stateDB   *stateDB
	amend     bool
	params    *batchedStorParams
	localStor map[string]*batchesGroup
	memSize   int // Total size (in bytes) of what was added.
	memLimit  int // When memSize >= memLimit, we should flush().
	maxKeys   int
}

func newBatchedStorage(
	db keyvalue.IterableKeyVal,
	stateDB *stateDB,
	params *batchedStorParams,
	memLimit int,
	maxKeys int,
	amend bool,
) (*batchedStorage, error) {
	// Actual record size is greater by blockNumLen.
	params.recordSize += blockNumLen
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create db batch")
	}
	return &batchedStorage{
		db:        db,
		dbBatch:   dbBatch,
		writeLock: stateDB.retrieveWriteLock(),
		stateDB:   stateDB,
		params:    params,
		localStor: make(map[string]*batchesGroup),
		memSize:   0,
		memLimit:  memLimit,
		maxKeys:   maxKeys,
		amend:     amend,
	}, nil
}

func (s *batchedStorage) lastRecordByKey(key []byte) ([]byte, error) {
	last, err := s.readLastBatch(key)
	if err != nil {
		return nil, err
	}
	record, err := last.lastRecord()
	if err != nil {
		return nil, err
	}
	return record.recordBytes(), nil
}

func (s *batchedStorage) newestLastRecordByKey(key []byte) ([]byte, error) {
	bg, ok := s.localStor[string(key)]
	if !ok {
		return s.lastRecordByKey(key)
	}
	record, err := bg.lastRecord()
	if err != nil {
		return nil, err
	}
	return record.recordBytes(), nil
}

func (s *batchedStorage) newBatchGroupForKey(key []byte) (*batchesGroup, error) {
	bg, err := newBatchesGroup(s.params.maxBatchSize, s.params.recordSize)
	if err != nil {
		return nil, err
	}
	last, err := s.readLastBatch(key)
	if err == errNotFound {
		return bg, nil
	} else if err != nil {
		return nil, err
	}
	bg.initFirstBatch(last)
	return bg, nil
}

func (s *batchedStorage) addRecordBytes(key, record []byte) error {
	keyStr := string(key)
	bg, ok := s.localStor[keyStr]
	if ok {
		if err := bg.appendNewRecord(record); err != nil {
			return err
		}
		s.memSize += len(record)
	} else {
		newGroup, err := s.newBatchGroupForKey(key)
		if err != nil {
			return err
		}
		if err := newGroup.appendNewRecord(record); err != nil {
			return err
		}
		s.localStor[keyStr] = newGroup
		s.memSize += len(key) + len(record)
	}
	if s.memSize >= s.memLimit || len(s.localStor) >= s.maxKeys {
		if err := s.flush(); err != nil {
			return err
		}
		s.reset()
	}
	return nil
}

// Appends one more record (at the end) for specified key.
func (s *batchedStorage) addRecord(key []byte, data []byte, blockNum uint32) error {
	r := &record{data: data, blockNum: blockNum}
	recordBytes := r.marshalBinary()
	return s.addRecordBytes(key, recordBytes)
}

func (s *batchedStorage) batchByNum(key []byte, num uint32) (*batch, error) {
	batchKey := batchedStorKey{prefix: s.params.prefix, internalKey: key, batchNum: num}
	batch, err := s.db.Get(batchKey.bytes())
	if err != nil {
		return nil, err
	}
	return newBatchWithData(batch, s.params.maxBatchSize, s.params.recordSize, num)
}

func (s *batchedStorage) moveLastBatchPointer(key []byte, lastNum uint32) error {
	if lastNum == firstBatchNum {
		if err := s.removeLastBatchNum(key); err != nil {
			return errors.Wrap(err, "failed to remove last batch num")
		}
	} else {
		if err := s.saveLastBatchNumDirectly(key, lastNum-1); err != nil {
			return errors.Wrap(err, "failed to save batch num to db")
		}
	}
	return nil
}

func (s *batchedStorage) handleEmptyBatch(key []byte, batchNum uint32) error {
	if err := s.moveLastBatchPointer(key, batchNum); err != nil {
		return errors.Wrap(err, "failed to update last batch num")
	}
	if err := s.removeBatchByNum(key, batchNum); err != nil {
		return errors.Wrap(err, "failed to remove batch by num")
	}
	return nil
}

func (s *batchedStorage) normalizeBatches(key []byte) error {
	// Lock the write lock.
	// Normalized batches will be written back to database, so we need to make sure
	// we read and write them under the same lock.
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	lastBatchNum, err := s.readLastBatchNum(key)
	if err != nil {
		// Nothing to normalize for this key.
		return nil
	}
	batchNum := lastBatchNum
	for {
		// Iterate until we find first non-empty (after filtering) batch.
		batchKey := batchedStorKey{prefix: s.params.prefix, internalKey: key, batchNum: batchNum}
		batch, err := s.db.Get(batchKey.bytes())
		if err != nil {
			return errors.Wrap(err, "failed to get batch by key")
		}
		newBatch, err := s.newestNormalize(batch)
		if err != nil {
			return errors.Wrap(err, "failed to normalize batch")
		}
		batchChanged := len(newBatch) != len(batch)
		if batchChanged {
			// Write normalized version of batch to database.
			if err := s.writeBatchDirectly(batchKey.bytes(), newBatch); err != nil {
				return errors.Wrap(err, "failed to write batch")
			}
		}
		if len(newBatch) == 0 {
			// Batch is empty.
			if err := s.handleEmptyBatch(key, batchNum); err != nil {
				return errors.Wrap(err, "failed to handle empty batch")
			}
			if batchNum == firstBatchNum {
				return nil
			}
			batchNum--
			continue
		}
		return nil
	}
}

func (s *batchedStorage) readLastBatch(key []byte) (*batch, error) {
	if s.amend {
		if err := s.normalizeBatches(key); err != nil {
			return nil, errors.Wrap(err, "failed to normalize")
		}
	}
	lastBatchNum, err := s.readLastBatchNum(key)
	if err != nil {
		return nil, errNotFound
	}
	return s.batchByNum(key, lastBatchNum)
}

// newBackwardRecordIterator() returns backward iterator for iterating single records.
func (s *batchedStorage) newBackwardRecordIterator(key []byte) (*recordIterator, error) {
	k := batchedStorKey{prefix: s.params.prefix, internalKey: key}
	rawIter, err := s.db.NewKeyIterator(k.prefixUntilBatch())
	if err != nil {
		return nil, err
	}
	batchIter := newBatchIterator(s, rawIter)
	return newRecordIterator(batchIter, s.params.recordSize), nil
}

type blockValidationFunc func(blockNum uint32) (bool, error)

func (s *batchedStorage) normalizeCommon(batch []byte, isValidBlock blockValidationFunc) ([]byte, error) {
	size := s.params.recordSize
	if (len(batch) % size) != 0 {
		return nil, errInvalidDataSize
	}
	for i := len(batch); i >= size; i -= size {
		recordBytes := batch[i-size : i]
		record, err := newRecordFromBytes(recordBytes)
		if err != nil {
			return nil, err
		}
		isValid, err := isValidBlock(record.blockNum)
		if err != nil {
			return nil, err
		}
		if isValid {
			break
		}
		batch = batch[:i-size]
	}
	return batch, nil
}

func (s *batchedStorage) normalize(batch []byte) ([]byte, error) {
	return s.normalizeCommon(batch, s.stateDB.isValidBlock)
}

func (s *batchedStorage) newestNormalize(batch []byte) ([]byte, error) {
	return s.normalizeCommon(batch, s.stateDB.newestIsValidBlock)
}

func (s *batchedStorage) removeBatchByNum(key []byte, num uint32) error {
	batchKey := batchedStorKey{prefix: s.params.prefix, internalKey: key, batchNum: num}
	if err := s.db.Delete(batchKey.bytes()); err != nil {
		return errors.Wrap(err, "failed to delete batch")
	}
	return nil
}

func (s *batchedStorage) removeLastBatchNum(key []byte) error {
	numKey := lastBatchKey{prefix: s.params.prefix, internalKey: key}
	if err := s.db.Delete(numKey.bytes()); err != nil {
		return errors.Wrap(err, "failed to delete last batch num")
	}
	return nil
}

func (s *batchedStorage) writeBatchDirectly(key, batch []byte) error {
	return s.db.Put(key, batch)
}

func (s *batchedStorage) saveLastBatchNumDirectly(key []byte, num uint32) error {
	k := lastBatchKey{prefix: s.params.prefix, internalKey: key}
	numBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numBytes, num)
	return s.db.Put(k.bytes(), numBytes)
}

func (s *batchedStorage) saveLastBatchNum(key []byte, num uint32) {
	k := lastBatchKey{prefix: s.params.prefix, internalKey: key}
	numBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numBytes, num)
	s.dbBatch.Put(k.bytes(), numBytes)
}

func (s *batchedStorage) readLastBatchNum(key []byte) (uint32, error) {
	k := lastBatchKey{prefix: s.params.prefix, internalKey: key}
	numBytes, err := s.db.Get(k.bytes())
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(numBytes), nil
}

func (s *batchedStorage) writeBatchGroup(key []byte, bg *batchesGroup) {
	k := batchedStorKey{prefix: s.params.prefix, internalKey: key}
	lastBatchNum := uint32(0)
	for _, batch := range bg.batches {
		lastBatchNum = batch.num
		k.batchNum = batch.num
		s.dbBatch.Put(k.bytes(), batch.bytes())
	}
	s.saveLastBatchNum(key, lastBatchNum)
}

func (s *batchedStorage) reset() {
	s.localStor = make(map[string]*batchesGroup)
	s.memSize = 0
	s.dbBatch.Reset()
}

func (s *batchedStorage) flush() error {
	for key, bg := range s.localStor {
		s.writeBatchGroup([]byte(key), bg)
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	if err := s.db.Flush(s.dbBatch); err != nil {
		return err
	}
	return nil
}

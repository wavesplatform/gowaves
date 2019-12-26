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
	data     []byte
	blockNum uint32
}

func newRecordFromBytes(data []byte) (*record, error) {
	if len(data) < blockNumLen {
		return nil, errInvalidDataSize
	}
	blockNum := binary.BigEndian.Uint32(data[len(data)-blockNumLen:])
	return &record{blockNum: blockNum, data: data[:len(data)-blockNumLen]}, nil
}

func (r *record) marshalBinary() []byte {
	buf := make([]byte, blockNumLen+len(r.data))
	copy(buf[:len(r.data)], r.data)
	binary.BigEndian.PutUint32(buf[len(buf)-blockNumLen:], r.blockNum)
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
	size := int(i.recordSize)
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
	key := keyvalue.SafeKey(i.iter)
	val := keyvalue.SafeValue(i.iter)
	// manageNormalization() MUST be called with `update` set to false.
	// Because values (and even keys) from iterator might not correspond with
	// current state of database.
	// It happens if iterator is retrieved before database gets updated, and is used after
	// it is updated.
	return i.stor.manageNormalization(key, val, false)
}

func (i *batchIterator) error() error {
	return i.iter.Error()
}

func (i *batchIterator) release() {
	i.iter.Release()
}

type batch struct {
	pos  int
	data []byte
	num  uint32
}

func newBatchWithData(data []byte, maxSize int, batchNum uint32) (*batch, error) {
	if len(data) > int(maxSize) {
		return nil, errInvalidDataSize
	}
	b := &batch{pos: len(data), num: batchNum}
	b.data = make([]byte, maxSize)
	copy(b.data, data)
	return b, nil
}

func newBatch(maxSize int, batchNum uint32) *batch {
	return &batch{pos: 0, data: make([]byte, maxSize), num: batchNum}
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
	bg.batches[0] = newBatch(bg.maxBatchSize, firstBatchNum)
}

func (bg *batchesGroup) appendNewRecord(record []byte) error {
	if len(record) != int(bg.recordSize) {
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
	newBatch := newBatch(bg.maxBatchSize, nextBatchNum)
	newBatch.addRecord(record)
	bg.batches = append(bg.batches, newBatch)
	return nil
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

	params    *batchedStorParams
	localStor map[string]*batchesGroup
	memSize   int // Total size (in bytes) of what was added.
	memLimit  int // When memSize >= memLimit, we should flush().
}

func newBatchedStorage(
	db keyvalue.IterableKeyVal,
	writeLock *sync.Mutex,
	stateDB *stateDB,
	params *batchedStorParams,
	memLimit int,
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
		writeLock: writeLock,
		stateDB:   stateDB,
		params:    params,
		localStor: make(map[string]*batchesGroup),
		memSize:   0,
		memLimit:  memLimit,
	}, nil
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

// Appends one more record (at the end) for specified key.
func (s *batchedStorage) addRecord(key []byte, data []byte, blockNum uint32) error {
	keyStr := string(key)
	r := &record{data: data, blockNum: blockNum}
	recordBytes := r.marshalBinary()
	bg, ok := s.localStor[keyStr]
	if ok {
		return bg.appendNewRecord(recordBytes)
	}
	newGroup, err := s.newBatchGroupForKey(key)
	if err != nil {
		return err
	}
	if err := newGroup.appendNewRecord(recordBytes); err != nil {
		return err
	}
	s.localStor[keyStr] = newGroup
	s.memSize += len(key) + len(data) + blockNumLen
	if s.memSize >= s.memLimit {
		if err := s.flush(); err != nil {
			return err
		}
		s.reset()
	}
	return nil
}

func (s *batchedStorage) readLastBatch(key []byte) (*batch, error) {
	// Lock the write lock.
	// Normalized history will be written to database, so we need to make sure
	// we read it and write under the same lock.
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	k := batchedStorKey{prefix: s.params.prefix, internalKey: key}
	iter, err := s.db.NewKeyIterator(k.prefixUntilBatch())
	if err != nil {
		return nil, err
	}
	if !iter.Last() {
		// Nothing to iterate.
		return nil, errNotFound
	}
	keyBytes := iter.Key()
	valBytes := keyvalue.SafeValue(iter)
	var batchKey batchedStorKey
	if err := batchKey.unmarshal(keyBytes); err != nil {
		return nil, err
	}
	normalized, err := s.manageNormalization(key, valBytes, true)
	if err != nil {
		return nil, err
	}
	return newBatchWithData(normalized, s.params.maxBatchSize, batchKey.batchNum)
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

func (s *batchedStorage) normalize(batch []byte) ([]byte, error) {
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
		isValid, err := s.stateDB.isValidBlock(record.blockNum)
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

func (s *batchedStorage) writeBatchDirectly(key, batch []byte) error {
	if len(batch) == 0 {
		return s.db.Delete(key)
	}
	return s.db.Put(key, batch)
}

func (s *batchedStorage) manageNormalization(key, batch []byte, update bool) ([]byte, error) {
	newBatch, err := s.normalize(batch)
	if err != nil {
		return nil, err
	}
	changed := len(newBatch) != len(batch)
	if changed && update {
		if err := s.writeBatchDirectly(key, newBatch); err != nil {
			return nil, err
		}
	}
	return newBatch, nil
}

func (s *batchedStorage) writeBatchGroup(key []byte, bg *batchesGroup) {
	k := batchedStorKey{prefix: s.params.prefix, internalKey: key}
	for _, batch := range bg.batches {
		k.batchNum = batch.num
		s.dbBatch.Put(k.bytes(), batch.bytes())
	}
}

func (s *batchedStorage) reset() {
	s.localStor = make(map[string]*batchesGroup)
	s.dbBatch.Reset()
}

func (s *batchedStorage) flush() error {
	for key, bg := range s.localStor {
		s.writeBatchGroup([]byte(key), bg)
	}
	s.writeLock.Lock()
	if err := s.db.Flush(s.dbBatch); err != nil {
		return err
	}
	s.writeLock.Unlock()
	return nil
}

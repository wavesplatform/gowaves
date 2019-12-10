package state

// batched_storage.go - stores arbitrary data for given key prefix, batching it in a way
// that no single value in db is larger than specified `batchSize` in bytes.
// data is sequence of records of similar size, batchedStorage also provides iterators
// to move through records, in range from most recent to oldest.

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

const (
	firstBatchNum = math.MaxUint32 // Hack instead of backward iteration.
	blockNumLen   = 4              // 4 bytes for block number.
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

type keyInfo struct {
	recordSize   uint32
	maxBatchSize uint32
}

func (inf *keyInfo) marshalBinary() []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint32(res[:4], inf.recordSize)
	binary.LittleEndian.PutUint32(res[4:], inf.maxBatchSize)
	return res
}

func (inf *keyInfo) unmarshalBinary(data []byte) error {
	if len(data) != 8 {
		return errInvalidDataSize
	}
	inf.recordSize = binary.LittleEndian.Uint32(data[:4])
	inf.maxBatchSize = binary.LittleEndian.Uint32(data[4:])
	return nil
}

type recordIterator struct {
	iter       *batchIterator
	batch      []byte
	recordSize uint32
	err        error
}

func newRecordIterator(iter *batchIterator, recordSize uint32) *recordIterator {
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
	size := int(i.recordSize)
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

func (i *recordIterator) getError() error {
	if err := i.iter.error(); err != nil {
		return err
	}
	return i.err
}

func (i *recordIterator) release() {
	i.iter.release()
}

type batchIterator struct {
	stor       *batchedStorage
	iter       keyvalue.Iterator
	recordSize uint32
}

func newBatchIterator(stor *batchedStorage, iter keyvalue.Iterator, recordSize uint32) *batchIterator {
	return &batchIterator{stor, iter, recordSize}
}

func (i *batchIterator) next() bool {
	return i.iter.Next()
}

func (i *batchIterator) currentBatch() ([]byte, error) {
	key := keyvalue.SafeKey(i.iter)
	val := keyvalue.SafeValue(i.iter)
	return i.stor.manageNormalization(key, val, i.recordSize)
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

func newBatchWithData(data []byte, maxSize, batchNum uint32) (*batch, error) {
	if len(data) > int(maxSize) {
		return nil, errInvalidDataSize
	}
	b := &batch{pos: len(data), num: batchNum}
	b.data = make([]byte, maxSize)
	copy(b.data, data)
	return b, nil
}

func newBatch(maxSize, batchNum uint32) *batch {
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
	maxBatchSize uint32
	recordSize   uint32
	batches      []*batch
}

func newBatchesGroup(maxBatchSize, recordSize uint32) (*batchesGroup, error) {
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
	if lastBatch.num == 0 {
		// Sanity check to prevent underflow.
		return errors.New("too many batches, can't add new!")
	}
	nextBatchNum := lastBatch.num - 1
	newBatch := newBatch(bg.maxBatchSize, nextBatchNum)
	newBatch.addRecord(record)
	bg.batches = append(bg.batches, newBatch)
	return nil
}

type batchedStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	stateDB *stateDB

	localStor map[string]*batchesGroup
}

func newBatchedStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, stateDB *stateDB) *batchedStorage {
	return &batchedStorage{
		db:        db,
		dbBatch:   dbBatch,
		stateDB:   stateDB,
		localStor: make(map[string]*batchesGroup),
	}
}

// createNewKey() stores key's basic parameters.
// Must be called during initialisation, keys can't be used until created.
func (s *batchedStorage) createNewKey(key []byte, maxBatchSize, recordSize int) error {
	// Append block num size to record size.
	recordSize += blockNumLen
	if maxBatchSize < recordSize {
		return errors.Errorf("bad recordSize %d > maxBatchSize %d", recordSize, maxBatchSize)
	}
	inf := &keyInfo{maxBatchSize: uint32(maxBatchSize), recordSize: uint32(recordSize)}
	infBytes := inf.marshalBinary()
	k := batchedInfoKey{key}
	s.dbBatch.Put(k.bytes(), infBytes)
	return nil
}

func (s *batchedStorage) infoByKey(key []byte) (*keyInfo, error) {
	k := batchedInfoKey{key}
	infBytes, err := s.db.Get(k.bytes())
	if err != nil {
		return nil, err
	}
	var inf keyInfo
	if err := inf.unmarshalBinary(infBytes); err != nil {
		return nil, err
	}
	return &inf, nil
}

func (s *batchedStorage) newBatchGroupForKey(key []byte) (*batchesGroup, error) {
	inf, err := s.infoByKey(key)
	if err != nil {
		return nil, err
	}
	bg, err := newBatchesGroup(inf.maxBatchSize, inf.recordSize)
	if err != nil {
		return nil, err
	}
	last, err := s.readLastBatch(key, inf.maxBatchSize, inf.recordSize)
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
	return nil
}

func (s *batchedStorage) readLastBatch(key []byte, maxBatchSize, recordSize uint32) (*batch, error) {
	k := batchedStorKey{internalKey: key}
	iter, err := s.db.NewKeyIterator(k.prefix())
	if err != nil {
		return nil, err
	}
	if !iter.First() {
		// Nothing to iterate.
		return nil, errNotFound
	}
	keyBytes := iter.Key()
	valBytes := keyvalue.SafeValue(iter)
	var batchKey batchedStorKey
	if err := batchKey.unmarshal(keyBytes); err != nil {
		return nil, err
	}
	normalized, err := s.manageNormalization(key, valBytes, recordSize)
	if err != nil {
		return nil, err
	}
	return newBatchWithData(normalized, maxBatchSize, batchKey.batchNum)
}

// newBackwardRecordIterator() returns backward iterator for iterating single records.
func (s *batchedStorage) newBackwardRecordIterator(key []byte) (*recordIterator, error) {
	k := batchedStorKey{internalKey: key}
	rawIter, err := s.db.NewKeyIterator(k.prefix())
	if err != nil {
		return nil, err
	}
	inf, err := s.infoByKey(key)
	if err != nil {
		return nil, err
	}
	batchIter := newBatchIterator(s, rawIter, inf.recordSize)
	return newRecordIterator(batchIter, inf.recordSize), nil
}

func (s *batchedStorage) normalize(batch []byte, recordSize uint32) ([]byte, error) {
	size := int(recordSize)
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

func (s *batchedStorage) manageNormalization(key, batch []byte, recordSize uint32) ([]byte, error) {
	newBatch, err := s.normalize(batch, recordSize)
	if err != nil {
		return nil, err
	}
	changed := len(newBatch) != len(batch)
	if changed {
		if err := s.writeBatchDirectly(key, newBatch); err != nil {
			return nil, err
		}
	}
	return newBatch, nil
}

func (s *batchedStorage) writeBatchGroup(key []byte, bg *batchesGroup) {
	k := batchedStorKey{internalKey: key}
	for _, batch := range bg.batches {
		k.batchNum = batch.num
		s.dbBatch.Put(k.bytes(), batch.bytes())
	}
}

func (s *batchedStorage) flush() error {
	for key, bg := range s.localStor {
		s.writeBatchGroup([]byte(key), bg)
	}
	s.localStor = make(map[string]*batchesGroup)
	return nil
}

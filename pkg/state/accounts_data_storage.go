package state

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type dataEntryRecordForHashes struct {
	addr  *proto.Address
	key   []byte
	value []byte
}

func (dr *dataEntryRecordForHashes) less(other stateComponent) bool {
	dr2 := other.(*dataEntryRecordForHashes)
	val := bytes.Compare(dr.addr[:], dr2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(dr.key, dr2.key) == -1
	}
	return true
}

func (dr *dataEntryRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(dr.addr[:]); err != nil {
		return err
	}
	if _, err := w.Write(dr.key); err != nil {
		return err
	}
	if dr.value == nil {
		return nil
	}
	if _, err := w.Write(dr.value); err != nil {
		return err
	}
	return nil
}

type dataEntryRecord struct {
	value []byte
}

func (r *dataEntryRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, len(r.value))
	copy(res, r.value)
	return res, nil
}

func (r *dataEntryRecord) unmarshalBinary(data []byte) error {
	r.value = make([]byte, len(data))
	copy(r.value, data)
	return nil
}

type entryId struct {
	addr proto.Address
	key  string
}

type accountsDataStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
	hasher  *stateHasher

	addrToNumMem map[proto.Address]uint64
	addrNum      uint64

	uncertainEntries map[entryId]proto.DataEntry

	calculateHashes bool
}

func newAccountsDataStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage, calcHashes bool) *accountsDataStorage {
	return &accountsDataStorage{
		db:               db,
		dbBatch:          dbBatch,
		hs:               hs,
		hasher:           newStateHasher(),
		addrToNumMem:     make(map[proto.Address]uint64),
		uncertainEntries: make(map[entryId]proto.DataEntry),
		calculateHashes:  calcHashes,
	}
}

func (s *accountsDataStorage) getLastAddrNum() (uint64, error) {
	lastAddrNumBytes, err := s.db.Get([]byte{lastAccountsStorAddrNumKeyPrefix})
	if err == keyvalue.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(lastAddrNumBytes), nil
}

func (s *accountsDataStorage) setLastAddrNum(lastAddrNum uint64) error {
	lastAddrNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lastAddrNumBytes, lastAddrNum)
	s.dbBatch.Put([]byte{lastAccountsStorAddrNumKeyPrefix}, lastAddrNumBytes)
	return nil
}

func (s *accountsDataStorage) newestAddrToNum(addr proto.Address) (uint64, error) {
	if addrNum, ok := s.addrToNumMem[addr]; ok {
		return addrNum, nil
	}
	return s.addrToNum(addr)
}

func (s *accountsDataStorage) addrToNum(addr proto.Address) (uint64, error) {
	addrToNumKey := accountStorAddrToNumKey{addr}
	addrNumBytes, err := s.db.Get(addrToNumKey.bytes())
	if err != nil {
		return 0, err
	}
	addrNum := binary.LittleEndian.Uint64(addrNumBytes)
	return addrNum, nil
}

func (s *accountsDataStorage) appendAddr(addr proto.Address) (uint64, error) {
	if addrNum, err := s.newestAddrToNum(addr); err == nil {
		// Already there.
		return addrNum, nil
	}
	lastAddrNum, err := s.getLastAddrNum()
	if err != nil {
		return 0, err
	}
	newAddrNum := lastAddrNum + uint64(s.addrNum)
	s.addrNum++
	s.addrToNumMem[addr] = newAddrNum
	addrToNum := accountStorAddrToNumKey{addr}
	newAddrNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(newAddrNumBytes, newAddrNum)
	s.dbBatch.Put(addrToNum.bytes(), newAddrNumBytes)
	return newAddrNum, nil
}

func (s *accountsDataStorage) dropUncertain() {
	s.uncertainEntries = make(map[entryId]proto.DataEntry)
}

func (s *accountsDataStorage) commitUncertain(blockID proto.BlockID) error {
	for id, entry := range s.uncertainEntries {
		if err := s.appendEntry(id.addr, entry, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *accountsDataStorage) appendEntryUncertain(addr proto.Address, entry proto.DataEntry) {
	id := entryId{addr, entry.GetKey()}
	s.uncertainEntries[id] = entry
}

func (s *accountsDataStorage) appendEntry(addr proto.Address, entry proto.DataEntry, blockID proto.BlockID) error {
	addrNum, err := s.appendAddr(addr)
	if err != nil {
		return err
	}
	key := accountsDataStorKey{addrNum, entry.GetKey()}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	valueBytes, err := entry.MarshalValue()
	if err != nil {
		return err
	}
	record := &dataEntryRecord{valueBytes}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if s.calculateHashes {
		r := &dataEntryRecordForHashes{
			addr: &addr,
			key:  []byte(entry.GetKey()),
		}
		if entry.GetValueType() != proto.DataDelete {
			// No value should be set for deletion.
			r.value = valueBytes
		}
		if err := s.hasher.push(keyStr, r, blockID); err != nil {
			return err
		}
	}
	if err := s.hs.addNewEntry(dataEntry, keyBytes, recordBytes, blockID); err != nil {
		return err
	}
	return nil
}

func (s *accountsDataStorage) newestEntryBytes(addr proto.Address, entryKey string, filter bool) ([]byte, error) {
	addrNum, err := s.newestAddrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum, entryKey}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record dataEntryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return record.value, nil
}

func (s *accountsDataStorage) entryBytes(addr proto.Address, entryKey string, filter bool) ([]byte, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum, entryKey}
	recordBytes, err := s.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record dataEntryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return record.value, nil
}

func (s *accountsDataStorage) retrieveEntries(addr proto.Address, filter bool) ([]proto.DataEntry, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum: addrNum}
	iter, err := s.hs.newTopEntryIteratorByPrefix(key.accountPrefix(), filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err != nil {
			zap.S().Fatalf("Iterator release error: %v", err)
		}
	}()

	var entries []proto.DataEntry
	for iter.Next() {
		entryKeyBytes := keyvalue.SafeKey(iter)
		recordBytes := keyvalue.SafeValue(iter)
		var record dataEntryRecord
		if err := record.unmarshalBinary(recordBytes); err != nil {
			return nil, err
		}
		var entryKey accountsDataStorKey
		if err := entryKey.unmarshal(entryKeyBytes); err != nil {
			return nil, err
		}
		entry, err := proto.NewDataEntryFromValueBytes(record.value)
		if err != nil {
			return nil, err
		}
		entry.SetKey(entryKey.entryKey)
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *accountsDataStorage) retrieveNewestEntry(addr proto.Address, key string, filter bool) (proto.DataEntry, error) {
	id := entryId{addr, key}
	if entry, ok := s.uncertainEntries[id]; ok {
		return entry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	entry, err := proto.NewDataEntryFromValueBytes(entryBytes)
	if err != nil {
		return nil, err
	}
	entry.SetKey(key)
	return entry, nil
}

func (s *accountsDataStorage) retrieveEntry(addr proto.Address, key string, filter bool) (proto.DataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	entry, err := proto.NewDataEntryFromValueBytes(entryBytes)
	if err != nil {
		return nil, err
	}
	entry.SetKey(key)
	return entry, nil
}

func (s *accountsDataStorage) retrieveNewestIntegerEntry(addr proto.Address, key string, filter bool) (*proto.IntegerDataEntry, error) {
	id := entryId{addr, key}
	if entry, ok := s.uncertainEntries[id]; ok {
		intEntry, ok := entry.(*proto.IntegerDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to integer entry")
		}
		return intEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.IntegerDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveIntegerEntry(addr proto.Address, key string, filter bool) (*proto.IntegerDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.IntegerDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveNewestBooleanEntry(addr proto.Address, key string, filter bool) (*proto.BooleanDataEntry, error) {
	id := entryId{addr, key}
	if entry, ok := s.uncertainEntries[id]; ok {
		boolEntry, ok := entry.(*proto.BooleanDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to boolean entry")
		}
		return boolEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.BooleanDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveBooleanEntry(addr proto.Address, key string, filter bool) (*proto.BooleanDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.BooleanDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveNewestStringEntry(addr proto.Address, key string, filter bool) (*proto.StringDataEntry, error) {
	id := entryId{addr, key}
	if entry, ok := s.uncertainEntries[id]; ok {
		stringEntry, ok := entry.(*proto.StringDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to string entry")
		}
		return stringEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.StringDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveStringEntry(addr proto.Address, key string, filter bool) (*proto.StringDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.StringDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveNewestBinaryEntry(addr proto.Address, key string, filter bool) (*proto.BinaryDataEntry, error) {
	id := entryId{addr, key}
	if entry, ok := s.uncertainEntries[id]; ok {
		binaryEntry, ok := entry.(*proto.BinaryDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to binary entry")
		}
		return binaryEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.BinaryDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) retrieveBinaryEntry(addr proto.Address, key string, filter bool) (*proto.BinaryDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key, filter)
	if err != nil {
		return nil, err
	}
	var entry proto.BinaryDataEntry
	if err := entry.UnmarshalValue(entryBytes); err != nil {
		return nil, err
	}
	entry.Key = key
	return &entry, nil
}

func (s *accountsDataStorage) prepareHashes() error {
	return s.hasher.stop()
}

func (s *accountsDataStorage) flush() error {
	lastAddrNum, err := s.getLastAddrNum()
	if err != nil {
		return err
	}
	newAddrNum := lastAddrNum + uint64(s.addrNum)
	if err := s.setLastAddrNum(newAddrNum); err != nil {
		return err
	}
	return nil
}

func (s *accountsDataStorage) reset() {
	s.addrToNumMem = make(map[proto.Address]uint64)
	s.addrNum = 0
	if s.calculateHashes {
		s.hasher.reset()
	}
}

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
	addr  []byte
	key   []byte
	value []byte
}

func (dr *dataEntryRecordForHashes) less(other stateComponent) bool {
	dr2 := other.(*dataEntryRecordForHashes)
	val := bytes.Compare(dr.addr, dr2.addr)
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(dr.key, dr2.key) == -1
	}
	return true
}

func (dr *dataEntryRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(dr.addr); err != nil {
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
	addrID proto.AddressID
	key    string
}

type uncertainAccountsDataStorageEntry struct {
	addr      proto.Address
	dataEntry proto.DataEntry
}

type accountsDataStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
	hasher  *stateHasher

	addrToNumMem map[proto.AddressID]uint64
	addrNum      uint64

	uncertainEntries map[entryId]uncertainAccountsDataStorageEntry

	calculateHashes bool
}

func newAccountsDataStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage, calcHashes bool) *accountsDataStorage {
	return &accountsDataStorage{
		db:               db,
		dbBatch:          dbBatch,
		hs:               hs,
		hasher:           newStateHasher(),
		addrToNumMem:     make(map[proto.AddressID]uint64),
		uncertainEntries: make(map[entryId]uncertainAccountsDataStorageEntry),
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

// newestAddressToNum returns the number of given address. It looks up for the address in cache map first
// and if not present in state. The second result parameter is true if account's number was found cache otherwise false.
// Error can be `keyvalue.ErrNotFound` if no corresponding number found for given address.
func (s *accountsDataStorage) newestAddrToNum(addr proto.Address) (uint64, bool, error) {
	if addrNum, ok := s.addrToNumMem[addr.ID()]; ok {
		return addrNum, true, nil
	}
	addrNum, err := s.addrToNum(addr)
	return addrNum, false, err
}

func (s *accountsDataStorage) addrToNum(addr proto.Address) (uint64, error) {
	addrToNumKey := accountStorAddrToNumKey{addr.ID()}
	addrNumBytes, err := s.db.Get(addrToNumKey.bytes())
	if err != nil {
		return 0, err
	}
	addrNum := binary.LittleEndian.Uint64(addrNumBytes)
	return addrNum, nil
}

func (s *accountsDataStorage) appendAddr(addr proto.Address) (uint64, error) {
	if addrNum, _, err := s.newestAddrToNum(addr); err == nil {
		// Already there.
		return addrNum, nil
	}
	lastAddrNum, err := s.getLastAddrNum()
	if err != nil {
		return 0, err
	}
	newAddrNum := lastAddrNum + s.addrNum
	s.addrNum++
	s.addrToNumMem[addr.ID()] = newAddrNum
	addrToNum := accountStorAddrToNumKey{addr.ID()}
	newAddrNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(newAddrNumBytes, newAddrNum)
	s.dbBatch.Put(addrToNum.bytes(), newAddrNumBytes)
	return newAddrNum, nil
}

func (s *accountsDataStorage) dropUncertain() {
	s.uncertainEntries = make(map[entryId]uncertainAccountsDataStorageEntry)
}

func (s *accountsDataStorage) commitUncertain(blockID proto.BlockID) error {
	for _, entry := range s.uncertainEntries {
		if err := s.appendEntry(entry.addr, entry.dataEntry, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *accountsDataStorage) appendEntryUncertain(addr proto.Address, entry proto.DataEntry) {
	id := entryId{addr.ID(), entry.GetKey()}
	s.uncertainEntries[id] = uncertainAccountsDataStorageEntry{addr, entry}
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
			addr:  addr.Bytes(),
			key:   []byte(entry.GetKey()),
			value: valueBytes,
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

func (s *accountsDataStorage) newestEntryBytes(addr proto.Address, entryKey string) ([]byte, error) {
	addrNum, _, err := s.newestAddrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum, entryKey}
	recordBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var record dataEntryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return record.value, nil
}

func (s *accountsDataStorage) entryBytes(addr proto.Address, entryKey string) ([]byte, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum, entryKey}
	recordBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var record dataEntryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return record.value, nil
}

func (s *accountsDataStorage) retrieveEntries(addr proto.Address) ([]proto.DataEntry, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum: addrNum}
	iter, err := s.hs.newTopEntryIteratorByPrefix(key.accountPrefix())
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
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
		// Skip Delete entries, they are not returned by APIs.
		if entry.GetValueType() == proto.DataDelete {
			continue
		}
		entry.SetKey(entryKey.entryKey)
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *accountsDataStorage) newestEntryExists(addr proto.Address) (bool, error) {
	addrNum, newest, err := s.newestAddrToNum(addr)
	if err != nil {
		// If there is no number for the address, no data for this address was saved before
		if errors.Is(err, keyvalue.ErrNotFound) {
			return false, nil
		}
		return false, err // Other bloom filter errors is possible
	}
	if newest {
		return true, nil
	}
	key := accountsDataStorKey{addrNum: addrNum}
	iter, err := s.hs.newTopEntryIteratorByPrefix(key.accountPrefix())
	if err != nil {
		return false, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil && !errors.Is(iter.Error(), keyvalue.ErrNotFound) {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()
	for iter.Next() {
		return true, nil
	}
	return false, nil
}

func (s *accountsDataStorage) retrieveNewestEntry(addr proto.Address, key string) (proto.DataEntry, error) {
	id := entryId{addr.ID(), key}
	if entry, ok := s.uncertainEntries[id]; ok {
		return entry.dataEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key)
	if err != nil {
		return nil, err
	}
	entry, err := proto.NewDataEntryFromValueBytes(entryBytes)
	if err != nil {
		return nil, err
	}
	if entry.GetValueType() == proto.DataDelete {
		return nil, errors.Errorf("entry '%s' was removed", key)
	}
	entry.SetKey(key)
	return entry, nil
}

func (s *accountsDataStorage) retrieveEntry(addr proto.Address, key string) (proto.DataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key)
	if err != nil {
		return nil, err
	}
	entry, err := proto.NewDataEntryFromValueBytes(entryBytes)
	if err != nil {
		return nil, err
	}
	if entry.GetValueType() == proto.DataDelete {
		return nil, errors.Errorf("entry '%s' was removed", key)
	}
	entry.SetKey(key)
	return entry, nil
}

func (s *accountsDataStorage) retrieveNewestIntegerEntry(addr proto.Address, key string) (*proto.IntegerDataEntry, error) {
	id := entryId{addr.ID(), key}
	if entry, ok := s.uncertainEntries[id]; ok {
		intEntry, ok := entry.dataEntry.(*proto.IntegerDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to integer entry")
		}
		return intEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveIntegerEntry(addr proto.Address, key string) (*proto.IntegerDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveNewestBooleanEntry(addr proto.Address, key string) (*proto.BooleanDataEntry, error) {
	id := entryId{addr.ID(), key}
	if entry, ok := s.uncertainEntries[id]; ok {
		boolEntry, ok := entry.dataEntry.(*proto.BooleanDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to boolean entry")
		}
		return boolEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveBooleanEntry(addr proto.Address, key string) (*proto.BooleanDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveNewestStringEntry(addr proto.Address, key string) (*proto.StringDataEntry, error) {
	id := entryId{addr.ID(), key}
	if entry, ok := s.uncertainEntries[id]; ok {
		stringEntry, ok := entry.dataEntry.(*proto.StringDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to string entry")
		}
		return stringEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveStringEntry(addr proto.Address, key string) (*proto.StringDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveNewestBinaryEntry(addr proto.Address, key string) (*proto.BinaryDataEntry, error) {
	id := entryId{addr.ID(), key}
	if entry, ok := s.uncertainEntries[id]; ok {
		binaryEntry, ok := entry.dataEntry.(*proto.BinaryDataEntry)
		if !ok {
			return nil, errors.New("failed to convert to binary entry")
		}
		return binaryEntry, nil
	}
	entryBytes, err := s.newestEntryBytes(addr, key)
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

func (s *accountsDataStorage) retrieveBinaryEntry(addr proto.Address, key string) (*proto.BinaryDataEntry, error) {
	entryBytes, err := s.entryBytes(addr, key)
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
	newAddrNum := lastAddrNum + s.addrNum
	if err := s.setLastAddrNum(newAddrNum); err != nil {
		return err
	}
	return nil
}

func (s *accountsDataStorage) reset() {
	s.addrToNumMem = make(map[proto.AddressID]uint64)
	s.addrNum = 0
	if s.calculateHashes {
		s.hasher.reset()
	}
}

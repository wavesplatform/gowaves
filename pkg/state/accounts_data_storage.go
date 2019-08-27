package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type dataEntryRecord struct {
	value    []byte
	blockNum uint32
}

func (r *dataEntryRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, len(r.value)+4)
	copy(res[:len(r.value)], r.value)
	binary.BigEndian.PutUint32(res[len(r.value):len(r.value)+4], r.blockNum)
	return res, nil
}

func (r *dataEntryRecord) unmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return errors.New("invalid data size")
	}
	r.value = make([]byte, len(data)-4)
	copy(r.value, data[:len(data)-4])
	r.blockNum = binary.BigEndian.Uint32(data[len(data)-4:])
	return nil
}

type accountsDataStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
	stateDB *stateDB

	newestAddrToNum map[proto.Address]uint64
	addrNum         uint64
}

func newAccountsDataStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage, stateDB *stateDB) (*accountsDataStorage, error) {
	return &accountsDataStorage{
		db:              db,
		dbBatch:         dbBatch,
		hs:              hs,
		stateDB:         stateDB,
		newestAddrToNum: make(map[proto.Address]uint64),
	}, nil
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

func (s *accountsDataStorage) addrToNum(addr proto.Address) (uint64, error) {
	if addrNum, ok := s.newestAddrToNum[addr]; ok {
		return addrNum, nil
	}
	addrToNumKey := accountStorAddrToNumKey{addr}
	addrNumBytes, err := s.db.Get(addrToNumKey.bytes())
	if err != nil {
		return 0, err
	}
	addrNum := binary.LittleEndian.Uint64(addrNumBytes)
	return addrNum, nil
}

func (s *accountsDataStorage) appendAddr(addr proto.Address) (uint64, error) {
	if addrNum, err := s.addrToNum(addr); err == nil {
		// Already there.
		return addrNum, nil
	}
	lastAddrNum, err := s.getLastAddrNum()
	if err != nil {
		return 0, err
	}
	newAddrNum := lastAddrNum + uint64(s.addrNum)
	s.addrNum++
	s.newestAddrToNum[addr] = newAddrNum
	addrToNum := accountStorAddrToNumKey{addr}
	newAddrNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(newAddrNumBytes, newAddrNum)
	s.dbBatch.Put(addrToNum.bytes(), newAddrNumBytes)
	return newAddrNum, nil
}

func (s *accountsDataStorage) appendEntry(addr proto.Address, entry proto.DataEntry, blockID crypto.Signature) error {
	addrNum, err := s.appendAddr(addr)
	if err != nil {
		return err
	}
	key := accountsDataStorKey{addrNum, entry.GetKey()}
	blockNum, err := s.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	valueBytes, err := entry.MarshalValue()
	if err != nil {
		return err
	}
	record := &dataEntryRecord{valueBytes, blockNum}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := s.hs.set(dataEntry, key.bytes(), recordBytes); err != nil {
		return err
	}
	return nil
}

func (s *accountsDataStorage) newestEntryBytes(addr proto.Address, entryKey string, filter bool) ([]byte, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	key := accountsDataStorKey{addrNum, entryKey}
	recordBytes, err := s.hs.getFresh(key.bytes(), filter)
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
	recordBytes, err := s.hs.get(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record dataEntryRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return record.value, nil
}

func (s *accountsDataStorage) retrieveNewestEntry(addr proto.Address, key string, filter bool) (proto.DataEntry, error) {
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
	s.newestAddrToNum = make(map[proto.Address]uint64)
	s.addrNum = 0
}

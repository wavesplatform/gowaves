package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type accountsDataStorage struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	rw      *blockReadWriter
	stateDB *stateDB

	newestAddrToNum map[proto.Address]uint64
	addrNum         uint64
}

func newAccountsDataStorage(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, rw *blockReadWriter, stateDB *stateDB) (*accountsDataStorage, error) {
	return &accountsDataStorage{
		db:              db,
		dbBatch:         dbBatch,
		rw:              rw,
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

func (s *accountsDataStorage) appendEntry(addr proto.Address, entry proto.DataEntry, blockID crypto.Signature) error {
	addrNum, err := s.appendAddr(addr)
	if err != nil {
		return err
	}
	entryKey := entry.GetKey()
	blockNum, err := s.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	key := accountsDataStorKey{addrNum, entryKey, blockNum}
	valueBytes, err := entry.MarshalValue()
	if err != nil {
		return err
	}
	s.dbBatch.Put(key.bytes(), valueBytes)
	return nil
}

func (s *accountsDataStorage) isRecentValidBlock(blockNum uint32) (bool, error) {
	return isRecentValidBlock(s.rw, s.stateDB, blockNum)
}

func (s *accountsDataStorage) entryBytes(addr proto.Address, key string) ([]byte, error) {
	addrNum, err := s.addrToNum(addr)
	if err != nil {
		return nil, err
	}
	iter, err := s.db.NewKeyIterator(newAccountsDataBytePrefix(addrNum, key))
	if err != nil {
		return nil, err
	}
	correctKeyLength := properAccountDataKeyLength(key)
	maxBlockNum := int64(-1)
	var latestEntry []byte
	for iter.Next() {
		keyBytes := keyvalue.SafeKey(iter)
		if len(keyBytes) != correctKeyLength {
			// There could be some collisions between different data entries keys.
			// For example, key1 = key0 + blockNum.
			// In this case only total length differs key1's DB key from key0's DB key,
			// since key1's DB key = key0's DB key + blockNum.
			// If we find such key, we should skip it.
			continue
		}
		var key accountsDataStorKey
		if err := key.unmarshal(keyBytes); err != nil {
			return nil, err
		}
		isRecentValid, err := s.isRecentValidBlock(key.blockNum)
		if err != nil {
			return nil, err
		}
		if !isRecentValid {
			// This block is too far in the past or invalid due to rollback.
			if err := s.db.Delete(keyBytes); err != nil {
				return nil, err
			}
			continue
		}
		if int64(key.blockNum) > maxBlockNum {
			// Latest key will have maximum block number among valid blocks.
			maxBlockNum = int64(key.blockNum)
			latestEntry = keyvalue.SafeValue(iter)
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}
	if latestEntry == nil {
		return nil, errNotFound
	}
	return latestEntry, nil
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
	entry.SetKey(key)
	return entry, nil
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

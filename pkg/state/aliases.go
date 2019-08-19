package state

import (
	"encoding/binary"
	"log"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var errAliasDisabled = errors.New("alias was stolen and is now disabled")

const (
	aliasRecordSize = 1 + proto.AddressSize + 4
)

type aliasInfo struct {
	stolen bool
	addr   proto.Address
}

type aliasRecord struct {
	aliasInfo
	blockNum uint32
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, aliasRecordSize)
	proto.PutBool(res[:1], r.stolen)
	copy(res[1:1+proto.AddressSize], r.addr[:])
	binary.BigEndian.PutUint32(res[1+proto.AddressSize:], r.blockNum)
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) != aliasRecordSize {
		return errors.New("invalid data size")
	}
	var err error
	r.stolen, err = proto.Bool(data[:1])
	if err != nil {
		return err
	}
	copy(r.addr[:], data[1:1+proto.AddressSize])
	r.blockNum = binary.BigEndian.Uint32(data[1+proto.AddressSize:])
	return nil
}

type aliases struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	stateDB *stateDB
	hs      *historyStorage
}

func newAliases(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, stateDB *stateDB, hs *historyStorage) (*aliases, error) {
	return &aliases{db, dbBatch, stateDB, hs}, nil
}

func (a *aliases) createAlias(aliasStr string, info *aliasInfo, blockID crypto.Signature) error {
	key := aliasKey{aliasStr}
	blockNum, err := a.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	r := aliasRecord{*info, blockNum}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	return a.hs.set(alias, key.bytes(), recordBytes)
}

func (a *aliases) exists(aliasStr string, filter bool) bool {
	key := aliasKey{alias: aliasStr}
	if _, err := a.hs.getFresh(alias, key.bytes(), filter); err != nil {
		return false
	}
	return true
}

func (a *aliases) isDisabled(aliasStr string) (bool, error) {
	key := disabledAliasKey{alias: aliasStr}
	return a.db.Has(key.bytes())
}

func (a *aliases) newestAddrByAlias(aliasStr string, filter bool) (*proto.Address, error) {
	disabled, err := a.isDisabled(aliasStr)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	recordBytes, err := a.hs.getFresh(alias, key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.addr, nil
}

func (a *aliases) recordByAlias(key []byte, filter bool) (*aliasRecord, error) {
	recordBytes, err := a.hs.get(alias, key, filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record, nil
}

func (a *aliases) addrByAlias(aliasStr string, filter bool) (*proto.Address, error) {
	disabled, err := a.isDisabled(aliasStr)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	record, err := a.recordByAlias(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	return &record.addr, nil
}

func (a *aliases) disableStolenAliases() error {
	// TODO: this action can not be rolled back now, do we need it?
	iter, err := a.db.NewKeyIterator([]byte{aliasKeyPrefix})
	if err != nil {
		return err
	}

	for iter.Next() {
		keyBytes := iter.Key()
		record, err := a.recordByAlias(iter.Key(), true)
		if err != nil {
			return err
		}
		var key aliasKey
		if err := key.unmarshal(keyBytes); err != nil {
			return err
		}
		if record.stolen {
			log.Printf("Forbidding stolen alias %s\n", key.alias)
			disabledKey := disabledAliasKey(key)
			a.dbBatch.Put(disabledKey.bytes(), void)
		}
	}

	iter.Release()
	return iter.Error()
}

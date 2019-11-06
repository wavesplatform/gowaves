package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var errAliasDisabled = errors.New("alias was stolen and is now disabled")

const (
	aliasRecordSize = 1 + proto.AddressSize
)

type aliasInfo struct {
	stolen bool
	addr   proto.Address
}

type aliasRecord struct {
	info aliasInfo
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, aliasRecordSize)
	proto.PutBool(res[:1], r.info.stolen)
	copy(res[1:1+proto.AddressSize], r.info.addr[:])
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) != aliasRecordSize {
		return errors.New("invalid data size")
	}
	var err error
	r.info.stolen, err = proto.Bool(data[:1])
	if err != nil {
		return err
	}
	copy(r.info.addr[:], data[1:1+proto.AddressSize])
	return nil
}

type aliases struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
}

func newAliases(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage) (*aliases, error) {
	return &aliases{db, dbBatch, hs}, nil
}

func (a *aliases) createAlias(aliasStr string, info *aliasInfo, blockID crypto.Signature) error {
	key := aliasKey{aliasStr}
	r := aliasRecord{*info}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	return a.hs.addNewEntry(alias, key.bytes(), recordBytes, blockID)
}

func (a *aliases) exists(aliasStr string, filter bool) bool {
	key := aliasKey{alias: aliasStr}
	if _, err := a.hs.freshLatestEntryData(key.bytes(), filter); err != nil {
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
	recordBytes, err := a.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
	}
	return &record.info.addr, nil
}

func (a *aliases) recordByAlias(key []byte, filter bool) (*aliasRecord, error) {
	recordBytes, err := a.hs.latestEntryData(key, filter)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
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
	return &record.info.addr, nil
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
		if record.info.stolen {
			zap.S().Debugf("Forbidding stolen alias %s", key.alias)
			disabledKey := disabledAliasKey(key)
			a.dbBatch.Put(disabledKey.bytes(), void)
		}
	}

	iter.Release()
	return iter.Error()
}

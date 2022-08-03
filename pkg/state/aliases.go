package state

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var errAliasDisabled = errors.New("alias was stolen and is now disabled")

const aliasRecordSize = 1 + proto.WavesAddressSize

type aliasRecordForStateHashes struct {
	addr  *proto.WavesAddress
	alias []byte
}

func (ar *aliasRecordForStateHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(ar.addr[:]); err != nil {
		return err
	}
	if _, err := w.Write(ar.alias); err != nil {
		return err
	}
	return nil
}

func (ar *aliasRecordForStateHashes) less(other stateComponent) bool {
	ar2 := other.(*aliasRecordForStateHashes)
	val := bytes.Compare(ar.addr[:], ar2.addr[:])
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(ar.alias, ar2.alias) == -1
	}
	return true
}

type aliasInfo struct {
	stolen bool
	addr   proto.WavesAddress
}

type aliasRecord struct {
	info aliasInfo
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, aliasRecordSize)
	proto.PutBool(res[:1], r.info.stolen)
	copy(res[1:1+proto.WavesAddressSize], r.info.addr[:])
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) != aliasRecordSize {
		return errInvalidDataSize
	}
	var err error
	r.info.stolen, err = proto.Bool(data[:1])
	if err != nil {
		return err
	}
	copy(r.info.addr[:], data[1:1+proto.WavesAddressSize])
	return nil
}

type aliases struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage

	disabled map[string]bool

	calculateHashes bool
	hasher          *stateHasher
}

func newAliases(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage, calcHashes bool) *aliases {
	return &aliases{
		db:              db,
		dbBatch:         dbBatch,
		hs:              hs,
		disabled:        make(map[string]bool),
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

func (a *aliases) createAlias(aliasStr string, info *aliasInfo, blockID proto.BlockID) error {
	key := aliasKey{aliasStr}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	r := aliasRecord{*info}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	if a.calculateHashes {
		ar := &aliasRecordForStateHashes{
			addr:  &info.addr,
			alias: []byte(aliasStr),
		}
		if err := a.hasher.push(keyStr, ar, blockID); err != nil {
			return err
		}
	}
	return a.hs.addNewEntry(alias, keyBytes, recordBytes, blockID)
}

func (a *aliases) exists(aliasStr string) bool {
	key := aliasKey{alias: aliasStr}
	if _, err := a.hs.newestTopEntryData(key.bytes()); err != nil {
		return false
	}
	return true
}

func (a *aliases) newestIsDisabled(aliasStr string) (bool, error) {
	if _, ok := a.disabled[aliasStr]; ok {
		return true, nil
	}
	return a.isDisabled(aliasStr)
}

func (a *aliases) isDisabled(aliasStr string) (bool, error) {
	key := disabledAliasKey{alias: aliasStr}
	return a.db.Has(key.bytes())
}

func (a *aliases) newestAddrByAlias(aliasStr string) (*proto.WavesAddress, error) {
	disabled, err := a.newestIsDisabled(aliasStr)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	record, err := a.newestRecordByAlias(key.bytes())
	if err != nil {
		return nil, err
	}
	return &record.info.addr, nil
}

func (a *aliases) newestRecordByAlias(key []byte) (*aliasRecord, error) {
	recordBytes, err := a.hs.newestTopEntryData(key)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
	}
	return &record, nil
}

func (a *aliases) recordByAlias(key []byte) (*aliasRecord, error) {
	recordBytes, err := a.hs.topEntryData(key)
	if err != nil {
		return nil, err
	}
	var record aliasRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
	}
	return &record, nil
}

func (a *aliases) addrByAlias(aliasStr string) (*proto.WavesAddress, error) {
	disabled, err := a.isDisabled(aliasStr)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, errAliasDisabled
	}
	key := aliasKey{alias: aliasStr}
	record, err := a.recordByAlias(key.bytes())
	if err != nil {
		return nil, err
	}
	return &record.info.addr, nil
}

func (a *aliases) disableStolenAliases() error {
	// TODO: this action can not be rolled back now, do we need it?
	iter, err := a.hs.newNewestTopEntryIterator(alias)
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()
	for iter.Next() {
		keyBytes := iter.Key()
		recordBytes := iter.Value()
		var record aliasRecord
		if err := record.unmarshalBinary(recordBytes); err != nil {
			return errors.Errorf("failed to unmarshal record: %v", err)
		}
		var key aliasKey
		if err := key.unmarshal(keyBytes); err != nil {
			return err
		}
		if record.info.stolen {
			zap.S().Debugf("Forbidding stolen alias %s", key.alias)
			a.disabled[key.alias] = true
		}
	}
	return nil
}

func (a *aliases) prepareHashes() error {
	return a.hasher.stop()
}

func (a *aliases) flush() {
	for alias := range a.disabled {
		disabledKey := disabledAliasKey{alias}
		a.dbBatch.Put(disabledKey.bytes(), void)
	}
}

func (a *aliases) reset() {
	a.disabled = make(map[string]bool)
	a.hasher.reset()
}

func (a *aliases) disabledAliases() (map[string]struct{}, error) {
	iter, err := a.db.NewKeyIterator([]byte{disabledAliasKeyPrefix})
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()
	als := make(map[string]struct{})
	for iter.Next() {
		keyBytes := iter.Key()
		var key disabledAliasKey
		err := key.unmarshal(keyBytes)
		if err != nil {
			return nil, err
		}
		als[key.alias] = struct{}{}
	}
	return als, nil
}

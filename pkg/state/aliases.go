package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	aliasRecordSize = proto.AddressSize + crypto.SignatureSize
)

type aliasRecord struct {
	addr    proto.Address
	blockID crypto.Signature
}

func (r *aliasRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, proto.AddressSize+crypto.SignatureSize)
	copy(res[:proto.AddressSize], r.addr[:])
	copy(res[proto.AddressSize:], r.blockID[:])
	return res, nil
}

func (r *aliasRecord) unmarshalBinary(data []byte) error {
	if len(data) < aliasRecordSize {
		return errors.New("invalid data size")
	}
	copy(r.addr[:], data[:proto.AddressSize])
	copy(r.blockID[:], data[proto.AddressSize:])
	return nil
}

type aliases struct {
	db        keyvalue.IterableKeyVal
	dbBatch   keyvalue.Batch
	localStor map[string][]byte
	fmt       *historyFormatter
}

func newAliases(
	db keyvalue.IterableKeyVal,
	dbBatch keyvalue.Batch,
	stDb *stateDB,
	rb *recentBlocks,
) (*aliases, error) {
	fmt, err := newHistoryFormatter(aliasRecordSize, crypto.SignatureSize, stDb, rb)
	if err != nil {
		return nil, err
	}
	return &aliases{
		db:        db,
		dbBatch:   dbBatch,
		localStor: make(map[string][]byte),
		fmt:       fmt,
	}, nil
}

func (a *aliases) lastRecord(history []byte) (*aliasRecord, error) {
	last, err := a.fmt.getLatest(history)
	if err != nil {
		return nil, errors.Errorf("failed to get the last record: %v\n", err)
	}
	var record aliasRecord
	if err := record.unmarshalBinary(last); err != nil {
		return nil, errors.Errorf("failed to unmarshal history record: %v\n", err)
	}
	return &record, nil
}

func (a *aliases) createAlias(alias string, r *aliasRecord) error {
	key := aliasKey{alias: alias}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	history, _ := a.localStor[string(key.bytes())]
	history, err = a.fmt.addRecord(history, recordBytes)
	if err != nil {
		return err
	}
	a.localStor[string(key.bytes())] = history
	return nil
}

func (a *aliases) newestAddrByAlias(alias string, filter bool) (*proto.Address, error) {
	key := aliasKey{alias: alias}
	history, err := fullHistory(key.bytes(), a.db, a.localStor, a.fmt, filter)
	if err != nil {
		return nil, err
	}
	record, err := a.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return &record.addr, nil
}

func (a *aliases) addrByAlias(alias string) (*proto.Address, error) {
	key := aliasKey{alias: alias}
	history, err := a.db.Get(key.bytes())
	if err != nil {
		return nil, errors.Errorf("failed to retrieve alias history: %v\n", err)
	}
	history, err = a.fmt.normalize(history, true)
	if err != nil {
		return nil, errors.Errorf("failed to normalize alias history: %v\n", err)
	}
	record, err := a.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return &record.addr, nil
}

func (a *aliases) reset() {
	a.localStor = make(map[string][]byte)
}

func (a *aliases) flush(initialisation bool) error {
	if err := addHistoryToBatch(a.db, a.dbBatch, a.localStor, a.fmt, !initialisation); err != nil {
		return err
	}
	return nil
}

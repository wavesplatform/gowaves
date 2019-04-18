package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/history"
)

const (
	leasingRecordSize = 1 + 8 + 8 + proto.AddressSize*2 + crypto.SignatureSize
)

type leasing struct {
	isActive  bool
	leaseIn   uint64
	leaseOut  uint64
	recipient proto.Address
	sender    proto.Address
}

type leasingRecord struct {
	leasing
	blockID crypto.Signature
}

func (l *leasingRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, leasingRecordSize)
	proto.PutBool(res[0:1], l.isActive)
	binary.BigEndian.PutUint64(res[1:9], l.leaseIn)
	binary.BigEndian.PutUint64(res[9:17], l.leaseOut)
	copy(res[17:17+proto.AddressSize], l.recipient[:])
	copy(res[17+proto.AddressSize:17+proto.AddressSize*2], l.sender[:])
	copy(res[17+proto.AddressSize*2:], l.blockID[:])
	return res, nil
}

func (l *leasingRecord) unmarshalBinary(data []byte) error {
	var err error
	l.isActive, err = proto.Bool(data[0:1])
	if err != nil {
		return err
	}
	l.leaseIn = binary.BigEndian.Uint64(data[1:9])
	l.leaseOut = binary.BigEndian.Uint64(data[9:17])
	copy(l.recipient[:], data[17:17+proto.AddressSize])
	copy(l.sender[:], data[17+proto.AddressSize:17+proto.AddressSize*2])
	copy(l.blockID[:], data[17+proto.AddressSize*2:])
	return nil
}

type leases struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch

	stor map[string][]byte
	fmt  *history.HistoryFormatter
}

func newLeases(
	db keyvalue.KeyValue,
	dbBatch keyvalue.Batch,
	hInfo heightInfo,
	bInfo blockInfo,
) (*leases, error) {
	fmt, err := history.NewHistoryFormatter(leasingRecordSize, crypto.SignatureSize, hInfo, bInfo)
	if err != nil {
		return nil, err
	}
	return &leases{
		db:      db,
		dbBatch: dbBatch,
		stor:    make(map[string][]byte),
		fmt:     fmt,
	}, nil
}

func (l *leases) lastRecord(history []byte) (*leasingRecord, error) {
	last, err := l.fmt.GetLatest(history)
	if err != nil {
		return nil, errors.Errorf("failed to get the last record: %v\n", err)
	}
	var record leasingRecord
	if err := record.unmarshalBinary(last); err != nil {
		return nil, errors.Errorf("failed to unmarshal history record: %v\n", err)
	}
	return &record, nil
}

// Leasing info from DB or local storage.
func (l *leases) newestLeasingInfo(id crypto.Digest) (*leasing, error) {
	key := leaseKey{leaseID: id}
	history, err := fullHistory(key.bytes(), l.db, l.stor, l.fmt)
	if err != nil {
		return nil, err
	}
	record, err := l.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return &record.leasing, nil
}

// Stable leasing info from DB.
func (l *leases) leasingInfo(id crypto.Digest) (*leasing, error) {
	key := leaseKey{leaseID: id}
	history, err := l.db.Get(key.bytes())
	if err != nil {
		return nil, errors.Errorf("failed to retrieve lease history: %v\n", err)
	}
	history, err = l.fmt.Normalize(history)
	if err != nil {
		return nil, errors.Errorf("failed to normalize history: %v\n", err)
	}
	record, err := l.lastRecord(history)
	if err != nil {
		return nil, err
	}
	return &record.leasing, nil
}

func (l *leases) addLeasing(id crypto.Digest, r *leasingRecord) error {
	key := leaseKey{leaseID: id}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	history, _ := l.stor[string(key.bytes())]
	history, err = l.fmt.AddRecord(history, recordBytes)
	if err != nil {
		return errors.Errorf("failed to add leasing record to history: %v\n", err)
	}
	l.stor[string(key.bytes())] = history
	return nil
}

func (l *leases) cancelLeasing(id crypto.Digest, blockID crypto.Signature) error {
	leasing, err := l.newestLeasingInfo(id)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v\n", err)
	}
	leasing.isActive = false
	record := &leasingRecord{*leasing, blockID}
	return l.addLeasing(id, record)
}

func (l *leases) reset() {
	l.stor = make(map[string][]byte)
}

func (l *leases) flush() error {
	if err := addHistoryToBatch(l.db, l.dbBatch, l.stor, l.fmt); err != nil {
		return err
	}
	return nil
}

package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	leasingRecordSize = 1 + 8 + proto.AddressSize*2
)

type leasing struct {
	isActive    bool
	leaseAmount uint64
	recipient   proto.Address
	sender      proto.Address
}

type leasingRecord struct {
	leasing
}

func (l *leasingRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, leasingRecordSize)
	proto.PutBool(res[0:1], l.isActive)
	binary.BigEndian.PutUint64(res[1:9], l.leaseAmount)
	copy(res[9:9+proto.AddressSize], l.recipient[:])
	copy(res[9+proto.AddressSize:9+proto.AddressSize*2], l.sender[:])
	return res, nil
}

func (l *leasingRecord) unmarshalBinary(data []byte) error {
	if len(data) != leasingRecordSize {
		return errors.New("invalid data size")
	}
	var err error
	l.isActive, err = proto.Bool(data[0:1])
	if err != nil {
		return err
	}
	l.leaseAmount = binary.BigEndian.Uint64(data[1:9])
	copy(l.recipient[:], data[9:9+proto.AddressSize])
	copy(l.sender[:], data[9+proto.AddressSize:9+proto.AddressSize*2])
	return nil
}

type leases struct {
	db keyvalue.IterableKeyVal
	hs *historyStorage
}

func newLeases(db keyvalue.IterableKeyVal, hs *historyStorage) (*leases, error) {
	return &leases{db, hs}, nil
}

func (l *leases) cancelLeases(bySenders map[proto.Address]struct{}, blockID crypto.Signature) error {
	leaseIter, err := l.db.NewKeyIterator([]byte{leaseKeyPrefix})
	if err != nil {
		return errors.Errorf("failed to create key iterator to cancel leases: %v", err)
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	// Iterate all the leases.
	zap.S().Info("Started to cancel leases")
	for leaseIter.Next() {
		key := keyvalue.SafeKey(leaseIter)
		leaseBytes, err := l.hs.latestEntryData(key, true)
		if err != nil {
			return err
		}
		var leaseRecord leasingRecord
		if err := leaseRecord.unmarshalBinary(leaseBytes); err != nil {
			return errors.Errorf("failed to unmarshal lease: %v", err)
		}
		toCancel := true
		if bySenders != nil {
			_, toCancel = bySenders[leaseRecord.sender]
		}
		if leaseRecord.isActive && toCancel {
			// Cancel lease.
			var k leaseKey
			if err := k.unmarshal(key); err != nil {
				return errors.Errorf("failed to unmarshal lease key: %v", err)
			}
			zap.S().Infof("State: cancelling lease %s", k.leaseID.String())
			leaseRecord.isActive = false
			leaseBytes, err := leaseRecord.marshalBinary()
			if err != nil {
				return errors.Errorf("failed to marshal lease: %v", err)
			}
			if err := l.hs.addNewEntry(lease, key, leaseBytes, blockID); err != nil {
				return errors.Errorf("failed to save lease to storage: %v", err)
			}
		}
	}
	zap.S().Info("Finished to cancel leases")
	return nil
}

func (l *leases) validLeaseIns() (map[proto.Address]int64, error) {
	leaseIter, err := l.db.NewKeyIterator([]byte{leaseKeyPrefix})
	if err != nil {
		return nil, errors.Errorf("failed to create key iterator to cancel leases: %v", err)
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	leaseIns := make(map[proto.Address]int64)
	// Iterate all the leases.
	zap.S().Info("Started collecting leases")
	for leaseIter.Next() {
		leaseBytes, err := l.hs.latestEntryData(leaseIter.Key(), true)
		if err != nil {
			return nil, err
		}
		var lease leasingRecord
		if err := lease.unmarshalBinary(leaseBytes); err != nil {
			return nil, errors.Errorf("failed to unmarshal lease: %v", err)
		}
		if lease.isActive {
			leaseIns[lease.recipient] += int64(lease.leaseAmount)
		}
	}
	zap.S().Info("Finished collecting leases")
	return leaseIns, nil
}

// Leasing info from DB or local storage.
func (l *leases) newestLeasingInfo(id crypto.Digest, filter bool) (*leasing, error) {
	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record leasingRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
	}
	return &record.leasing, nil
}

// Stable leasing info from DB.
func (l *leases) leasingInfo(id crypto.Digest, filter bool) (*leasing, error) {
	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record leasingRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v", err)
	}
	return &record.leasing, nil
}

func (l *leases) addLeasing(id crypto.Digest, leasing *leasing, blockID crypto.Signature) error {
	key := leaseKey{leaseID: id}
	r := &leasingRecord{*leasing}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v", err)
	}
	if err := l.hs.addNewEntry(lease, key.bytes(), recordBytes, blockID); err != nil {
		return err
	}
	return nil
}

func (l *leases) cancelLeasing(id crypto.Digest, blockID crypto.Signature, filter bool) error {
	leasing, err := l.newestLeasingInfo(id, filter)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v", err)
	}
	leasing.isActive = false
	return l.addLeasing(id, leasing, blockID)
}

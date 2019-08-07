package state

import (
	"encoding/binary"
	"log"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	leasingRecordSize = 1 + 8 + proto.AddressSize*2 + 4
)

type leasing struct {
	isActive    bool
	leaseAmount uint64
	recipient   proto.Address
	sender      proto.Address
}

type leasingRecord struct {
	leasing
	blockNum uint32
}

func (l *leasingRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, leasingRecordSize)
	proto.PutBool(res[0:1], l.isActive)
	binary.BigEndian.PutUint64(res[1:9], l.leaseAmount)
	copy(res[9:9+proto.AddressSize], l.recipient[:])
	copy(res[9+proto.AddressSize:9+proto.AddressSize*2], l.sender[:])
	binary.BigEndian.PutUint32(res[9+proto.AddressSize*2:], l.blockNum)
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
	l.blockNum = binary.BigEndian.Uint32(data[9+proto.AddressSize*2:])
	return nil
}

type leases struct {
	db      keyvalue.IterableKeyVal
	stateDB *stateDB
	hs      *historyStorage
}

func newLeases(db keyvalue.IterableKeyVal, stateDB *stateDB, hs *historyStorage) (*leases, error) {
	return &leases{db, stateDB, hs}, nil
}

func (l *leases) cancelLeases(bySenders map[proto.Address]struct{}) error {
	// TODO: this action can not be rolled back now, do we need it?
	leaseIter, err := l.db.NewKeyIterator([]byte{leaseKeyPrefix})
	if err != nil {
		return errors.Errorf("failed to create key iterator to cancel leases: %v\n", err)
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	// Iterate all the leases.
	log.Printf("Started to cancel leases\n")
	for leaseIter.Next() {
		key := keyvalue.SafeKey(leaseIter)
		leaseBytes, err := l.hs.get(lease, key, true)
		if err != nil {
			return err
		}
		var leaseRecord leasingRecord
		if err := leaseRecord.unmarshalBinary(leaseBytes); err != nil {
			return errors.Errorf("failed to unmarshal lease: %v\n", err)
		}
		toCancel := true
		if bySenders != nil {
			_, toCancel = bySenders[leaseRecord.sender]
		}
		if leaseRecord.isActive && toCancel {
			// Cancel lease.
			var k leaseKey
			if err := k.unmarshal(key); err != nil {
				return errors.Errorf("failed to unmarshal lease key: %v\n", err)
			}
			log.Printf("State: cancelling lease %s", k.leaseID.String())
			leaseRecord.isActive = false
			leaseBytes, err := leaseRecord.marshalBinary()
			if err != nil {
				return errors.Errorf("failed to marshal lease: %v\n", err)
			}
			if err := l.hs.set(lease, key, leaseBytes); err != nil {
				return errors.Errorf("failed to save lease to storage: %v\n", err)
			}
		}
	}
	log.Printf("Finished to cancel leases\n")
	return nil
}

func (l *leases) validLeaseIns() (map[proto.Address]int64, error) {
	leaseIter, err := l.db.NewKeyIterator([]byte{leaseKeyPrefix})
	if err != nil {
		return nil, errors.Errorf("failed to create key iterator to cancel leases: %v\n", err)
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			log.Fatalf("Iterator error: %v", err)
		}
	}()

	leaseIns := make(map[proto.Address]int64)
	// Iterate all the leases.
	log.Printf("Started collecting leases\n")
	for leaseIter.Next() {
		leaseBytes, err := l.hs.get(lease, leaseIter.Key(), true)
		if err != nil {
			return nil, err
		}
		var lease leasingRecord
		if err := lease.unmarshalBinary(leaseBytes); err != nil {
			return nil, errors.Errorf("failed to unmarshal lease: %v\n", err)
		}
		if lease.isActive {
			leaseIns[lease.recipient] += int64(lease.leaseAmount)
		}
	}
	log.Printf("Finished collecting leases\n")
	return leaseIns, nil
}

// Leasing info from DB or local storage.
func (l *leases) newestLeasingInfo(id crypto.Digest, filter bool) (*leasing, error) {
	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.getFresh(lease, key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record leasingRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.leasing, nil
}

// Stable leasing info from DB.
func (l *leases) leasingInfo(id crypto.Digest, filter bool) (*leasing, error) {
	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.get(lease, key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record leasingRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %v\n", err)
	}
	return &record.leasing, nil
}

func (l *leases) addLeasing(id crypto.Digest, leasing *leasing, blockID crypto.Signature) error {
	key := leaseKey{leaseID: id}
	blockNum, err := l.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	r := &leasingRecord{*leasing, blockNum}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return errors.Errorf("failed to marshal record: %v\n", err)
	}
	if err := l.hs.set(lease, key.bytes(), recordBytes); err != nil {
		return err
	}
	return nil
}

func (l *leases) cancelLeasing(id crypto.Digest, blockID crypto.Signature, filter bool) error {
	leasing, err := l.newestLeasingInfo(id, filter)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v\n", err)
	}
	leasing.isActive = false
	return l.addLeasing(id, leasing, blockID)
}

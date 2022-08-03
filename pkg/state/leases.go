package state

import (
	"bytes"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type LeaseStatus byte

const (
	LeaseActive LeaseStatus = iota
	LeaseCanceled
	//TODO: LeaseExpired (for future use)
)

type leaseRecordForStateHashes struct {
	id     *crypto.Digest
	active byte
}

func (lr *leaseRecordForStateHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(lr.id[:]); err != nil {
		return err
	}
	if _, err := w.Write([]byte{lr.active}); err != nil {
		return err
	}
	return nil
}

func (lr *leaseRecordForStateHashes) less(other stateComponent) bool {
	lr2 := other.(*leaseRecordForStateHashes)
	return bytes.Compare(lr.id[:], lr2.id[:]) == -1
}

type leasing struct {
	Sender              proto.WavesAddress `cbor:"0,keyasint"`
	Recipient           proto.WavesAddress `cbor:"1,keyasint"`
	Amount              uint64             `cbor:"2,keyasint"`
	Height              uint64             `cbor:"3,keyasint"`
	Status              LeaseStatus        `cbor:"4,keyasint"`
	OriginTransactionID *crypto.Digest     `cbor:"5,keyasint,omitempty"`
	RecipientAlias      *proto.Alias       `cbor:"6,keyasint,omitempty"`
	CancelHeight        uint64             `cbor:"7,keyasint,omitempty"`
	CancelTransactionID *crypto.Digest     `cbor:"8,keyasint,omitempty"`
}

func (l leasing) isActive() bool {
	return l.Status == LeaseActive
}

type leases struct {
	hs *historyStorage

	uncertainLeases map[crypto.Digest]*leasing

	calculateHashes bool
	hasher          *stateHasher
}

func newLeases(hs *historyStorage, calcHashes bool) *leases {
	return &leases{
		hs:              hs,
		uncertainLeases: make(map[crypto.Digest]*leasing),
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

func (l *leases) cancelLeases(bySenders map[proto.WavesAddress]struct{}, blockID proto.BlockID) error {
	leaseIter, err := l.hs.newNewestTopEntryIterator(lease)
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
		leaseBytes := keyvalue.SafeValue(leaseIter)
		record := new(leasing)
		if err := cbor.Unmarshal(leaseBytes, record); err != nil {
			return errors.Wrap(err, "failed to unmarshal lease")
		}
		toCancel := true
		if bySenders != nil {
			_, toCancel = bySenders[record.Sender]
		}
		if record.isActive() && toCancel {
			// Cancel lease.
			var k leaseKey
			if err := k.unmarshal(key); err != nil {
				return errors.Wrap(err, "failed to unmarshal lease key")
			}
			zap.S().Infof("State: cancelling lease %s", k.leaseID.String())
			record.Status = LeaseCanceled
			if err := l.addLeasing(k.leaseID, record, blockID); err != nil {
				return errors.Wrap(err, "failed to save lease to storage")
			}
		}
	}
	zap.S().Info("Finished to cancel leases")
	return nil
}

func (l *leases) cancelLeasesToAliases(aliases map[string]struct{}, blockID proto.BlockID) (map[proto.WavesAddress]balanceDiff, error) {
	leaseIter, err := l.hs.newNewestTopEntryIterator(lease)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create key iterator to cancel leases to stolen aliases")
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	// Iterate all the leases.
	zap.S().Info("Started cancelling leases to disabled aliases")
	changes := make(map[proto.WavesAddress]balanceDiff)
	for leaseIter.Next() {
		keyBytes := keyvalue.SafeKey(leaseIter)
		var key leaseKey
		if err := key.unmarshal(keyBytes); err != nil {
			return nil, errors.Wrap(err, "failed ot unmarshal leasing key")
		}
		leaseBytes := keyvalue.SafeValue(leaseIter)
		record := new(leasing)
		if err := cbor.Unmarshal(leaseBytes, record); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal lease")
		}
		if record.isActive() && record.RecipientAlias != nil {
			if _, ok := aliases[record.RecipientAlias.Alias]; ok {
				zap.S().Infof("State: canceling lease %s", key.leaseID.String())
				record.Status = LeaseCanceled
				if err := l.addLeasing(key.leaseID, record, blockID); err != nil {
					return nil, errors.Wrap(err, "failed to save lease to storage")
				}
				if diff, ok := changes[record.Sender]; ok {
					diff.leaseOut += -int64(record.Amount)
					changes[record.Sender] = diff
				} else {
					changes[record.Sender] = newBalanceDiff(0, 0, -int64(record.Amount), false)
				}
				if diff, ok := changes[record.Recipient]; ok {
					diff.leaseIn += -int64(record.Amount)
					changes[record.Recipient] = diff
				} else {
					changes[record.Recipient] = newBalanceDiff(0, -int64(record.Amount), 0, false)
				}
			}
		}
	}
	zap.S().Info("Finished cancelling leases to disabled aliases")
	return changes, nil
}

func (l *leases) validLeaseIns() (map[proto.WavesAddress]int64, error) {
	leaseIter, err := l.hs.newNewestTopEntryIterator(lease)
	if err != nil {
		return nil, errors.Errorf("failed to create key iterator to cancel leases: %v", err)
	}
	defer func() {
		leaseIter.Release()
		if err := leaseIter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	leaseIns := make(map[proto.WavesAddress]int64)
	// Iterate all the leases.
	zap.S().Info("Started collecting leases")
	for leaseIter.Next() {
		leaseBytes := keyvalue.SafeValue(leaseIter)
		record := new(leasing)
		if err := cbor.Unmarshal(leaseBytes, record); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal lease")
		}
		if record.isActive() {
			leaseIns[record.Recipient] += int64(record.Amount)
		}
	}
	zap.S().Info("Finished collecting leases")
	return leaseIns, nil
}

// Leasing info from DB or local storage.
func (l *leases) newestLeasingInfo(id crypto.Digest) (*leasing, error) {
	if leasing, ok := l.uncertainLeases[id]; ok {
		return leasing, nil
	}

	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	record := new(leasing)
	if err := cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal record")
	}
	if record.OriginTransactionID == nil {
		record.OriginTransactionID = &id
	}
	return record, nil
}

// Stable leasing info from DB.
func (l *leases) leasingInfo(id crypto.Digest) (*leasing, error) {
	key := leaseKey{leaseID: id}
	recordBytes, err := l.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	record := new(leasing)
	if err := cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal record")
	}
	if record.OriginTransactionID == nil {
		record.OriginTransactionID = &id
	}
	return record, nil
}

func (l *leases) isActive(id crypto.Digest) (bool, error) {
	info, err := l.leasingInfo(id)
	if err != nil {
		return false, err
	}
	return info.isActive(), nil
}

func (l *leases) addLeasing(id crypto.Digest, leasing *leasing, blockID proto.BlockID) error {
	key := leaseKey{leaseID: id}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	recordBytes, err := cbor.Marshal(leasing)
	if err != nil {
		return errors.Wrap(err, "failed to marshal record")
	}
	if l.calculateHashes {
		active := byte(0)
		if leasing.isActive() {
			active = byte(1)
		}
		lr := &leaseRecordForStateHashes{
			id:     &id,
			active: active,
		}
		if err := l.hasher.push(keyStr, lr, blockID); err != nil {
			return err
		}
	}
	if err := l.hs.addNewEntry(lease, keyBytes, recordBytes, blockID); err != nil {
		return err
	}
	return nil
}

func (l *leases) addLeasingUncertain(id crypto.Digest, leasing *leasing) {
	l.uncertainLeases[id] = leasing
}

func (l *leases) cancelLeasing(id crypto.Digest, blockID proto.BlockID, height uint64, txID *crypto.Digest) error {
	leasing, err := l.newestLeasingInfo(id)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v", err)
	}
	leasing.Status = LeaseCanceled
	leasing.CancelHeight = height
	leasing.CancelTransactionID = txID
	return l.addLeasing(id, leasing, blockID)
}

func (l *leases) cancelLeasingUncertain(id crypto.Digest, height uint64, txID *crypto.Digest) error {
	leasing, err := l.newestLeasingInfo(id)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v", err)
	}
	leasing.Status = LeaseCanceled
	leasing.CancelTransactionID = txID
	leasing.CancelHeight = height
	l.addLeasingUncertain(id, leasing)
	return nil
}

func (l *leases) prepareHashes() error {
	return l.hasher.stop()
}

func (l *leases) reset() {
	l.hasher.reset()
}

func (l *leases) commitUncertain(blockID proto.BlockID) error {
	for id, leasing := range l.uncertainLeases {
		if err := l.addLeasing(id, leasing, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (l *leases) dropUncertain() {
	l.uncertainLeases = make(map[crypto.Digest]*leasing)
}

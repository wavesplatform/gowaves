package state

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/internal"
)

type LeaseStatus byte

const (
	LeaseActive LeaseStatus = iota
	LeaseCancelled
	//TODO: LeaseExpired (for future use)
)

type leaseRecordForStateHashes struct {
	id       crypto.Digest
	isActive bool
}

func (lr *leaseRecordForStateHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(lr.id[:]); err != nil {
		return err
	}
	active := byte(0)
	if lr.isActive {
		active = byte(1)
	}
	_, err := w.Write([]byte{active})
	return err
}

func (lr *leaseRecordForStateHashes) less(other stateComponent) bool {
	lr2 := other.(*leaseRecordForStateHashes)
	return bytes.Compare(lr.id[:], lr2.id[:]) == -1
}

type leasing struct {
	SenderPK            crypto.PublicKey   `cbor:"0,keyasint"`
	RecipientAddr       proto.WavesAddress `cbor:"1,keyasint"`
	Amount              uint64             `cbor:"2,keyasint"`
	OriginHeight        uint64             `cbor:"3,keyasint,omitempty"`
	Status              LeaseStatus        `cbor:"4,keyasint"`
	OriginTransactionID *crypto.Digest     `cbor:"5,keyasint,omitempty"`
	CancelHeight        uint64             `cbor:"7,keyasint,omitempty"`
	CancelTransactionID *crypto.Digest     `cbor:"8,keyasint,omitempty"`
}

func (l *leasing) isActive() bool {
	return l.Status == LeaseActive
}

func (l *leasing) marshalBinary() ([]byte, error) {
	return cbor.Marshal(l)
}

func (l *leasing) unmarshalBinary(data []byte) error {
	return cbor.Unmarshal(data, l)
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

func (l *leases) generateCancelledLeaseSnapshots(
	scheme proto.Scheme,
	bySenders map[proto.WavesAddress]struct{},
) ([]proto.CancelledLeaseSnapshot, error) {
	leaseIter, err := l.hs.newNewestTopEntryIterator(lease)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create key iterator to cancel leases")
	}
	defer func() {
		leaseIter.Release()
		if liErr := leaseIter.Error(); liErr != nil {
			slog.Error("Iterator error", logging.Error(liErr))
			panic(liErr)
		}
	}()

	var leasesToCancel []proto.CancelledLeaseSnapshot
	// Iterate all the leases.
	slog.Info("Started to cancel leases")
	for leaseIter.Next() {
		key := keyvalue.SafeKey(leaseIter)
		leaseBytes := keyvalue.SafeValue(leaseIter)
		record := new(leasing)
		if err = record.unmarshalBinary(leaseBytes); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal lease")
		}
		toCancel := true
		if len(bySenders) != 0 { // if is not empty, we need to check if the sender is in the set
			sender, addrErr := proto.NewAddressFromPublicKey(scheme, record.SenderPK)
			if addrErr != nil {
				return nil, errors.Wrapf(addrErr, "failed to build address from PK %q", record.SenderPK)
			}
			_, toCancel = bySenders[sender]
		}
		if record.isActive() && toCancel {
			// Cancel lease.
			var k leaseKey
			if err := k.unmarshal(key); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal lease key")
			}
			slog.Info("State: cancelling lease", "ID", k.leaseID.String())
			leasesToCancel = append(leasesToCancel, proto.CancelledLeaseSnapshot{
				LeaseID: k.leaseID,
			})
		}
	}
	slog.Info("Finished to cancel leases")
	return leasesToCancel, nil
}

func (l *leases) cancelLeasesToDisabledAliases(
	scheme proto.Scheme,
) ([]proto.CancelledLeaseSnapshot, map[proto.WavesAddress]balanceDiff, error) {
	if scheme != proto.MainNetScheme { // no-op
		return nil, nil, nil
	}
	slog.Info("Started cancelling leases to disabled aliases")
	leasesToCancelMainnet := leasesToDisabledAliasesMainnet()
	cancelledLeasesSnapshots := make([]proto.CancelledLeaseSnapshot, 0, len(leasesToCancelMainnet))
	changes := make(map[proto.WavesAddress]balanceDiff, len(leasesToCancelMainnet))
	for _, leaseID := range leasesToCancelMainnet {
		record, err := l.newestLeasingInfo(leaseID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get newest leasing info by id %q", leaseID.String())
		}
		slog.Info("State: canceling lease", "ID", leaseID)
		cancelledLeasesSnapshots = append(cancelledLeasesSnapshots, proto.CancelledLeaseSnapshot{
			LeaseID: leaseID,
		})
		// calculate balance changes
		senderAddr, err := proto.NewAddressFromPublicKey(scheme, record.SenderPK)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to build address for PK %q", record.SenderPK)
		}
		if diff, ok := changes[senderAddr]; ok {
			newLeaseOut, loErr := diff.leaseOut.Add(internal.NewIntChange(-int64(record.Amount)))
			if loErr != nil {
				return nil, nil, errors.Wrapf(loErr, "failed to add leaseOut change for address %q",
					senderAddr.String(),
				)
			}
			diff.leaseOut = newLeaseOut
			changes[senderAddr] = diff
		} else {
			changes[senderAddr] = newBalanceDiff(0, 0, -int64(record.Amount), false)
		}
		if diff, ok := changes[record.RecipientAddr]; ok {
			newLeaseIn, liErr := diff.leaseIn.Add(internal.NewIntChange(-int64(record.Amount)))
			if liErr != nil {
				return nil, nil, errors.Wrapf(liErr, "failed to add leaseIn change for address %q",
					record.RecipientAddr.String(),
				)
			}
			diff.leaseIn = newLeaseIn
			changes[record.RecipientAddr] = diff
		} else {
			changes[record.RecipientAddr] = newBalanceDiff(0, -int64(record.Amount), 0, false)
		}
	}
	slog.Info("Finished cancelling leases to disabled aliases")
	return cancelledLeasesSnapshots, changes, nil
}

// validLeaseIns returns a map of active leases by recipient address with correct leaseIn values.
// This function should not generate any fix snapshots, it just returns the map with valid leaseIn values.
func (l *leases) validLeaseIns() (map[proto.WavesAddress]int64, error) {
	leaseIter, err := l.hs.newNewestTopEntryIterator(lease)
	if err != nil {
		return nil, errors.Errorf("failed to create key iterator to cancel leases: %v", err)
	}
	defer func() {
		leaseIter.Release()
		if liErr := leaseIter.Error(); liErr != nil {
			slog.Error("Iterator error", logging.Error(liErr))
			panic(liErr)
		}
	}()

	leaseIns := make(map[proto.WavesAddress]int64)
	// Iterate all the leases.
	slog.Info("Started collecting leases")
	for leaseIter.Next() {
		leaseBytes := keyvalue.SafeValue(leaseIter)
		record := new(leasing)
		if err = record.unmarshalBinary(leaseBytes); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal lease")
		}
		if record.isActive() {
			leaseIns[record.RecipientAddr] += int64(record.Amount)
		}
	}
	slog.Info("Finished collecting leases")
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
	if err = record.unmarshalBinary(recordBytes); err != nil {
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
	if err = record.unmarshalBinary(recordBytes); err != nil {
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
	recordBytes, err := leasing.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal record")
	}
	if l.calculateHashes {
		lr := &leaseRecordForStateHashes{
			id:       id,
			isActive: leasing.isActive(),
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

func (l *leases) pushStateHash(leaseID crypto.Digest, isActive bool, blockID proto.BlockID) error {
	if !l.calculateHashes {
		return nil
	}
	key := leaseKey{leaseID: leaseID}
	lr := &leaseRecordForStateHashes{
		id:       leaseID,
		isActive: isActive,
	}
	return l.hasher.push(string(key.bytes()), lr, blockID)
}

func (l *leases) rawWriteLeasing(id crypto.Digest, leasing *leasing, blockID proto.BlockID) error {
	key := leaseKey{leaseID: id}
	keyBytes := key.bytes()
	recordBytes, err := leasing.marshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal record")
	}
	return l.hs.addNewEntry(lease, keyBytes, recordBytes, blockID)
}

func (l *leases) addLeasingUncertain(id crypto.Digest, leasing *leasing) {
	l.uncertainLeases[id] = leasing
}

func (l *leases) cancelLeasing(id crypto.Digest, blockID proto.BlockID, height uint64, txID *crypto.Digest) error {
	leasing, err := l.newestLeasingInfo(id)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v", err)
	}
	leasing.Status = LeaseCancelled
	leasing.CancelHeight = height
	leasing.CancelTransactionID = txID
	return l.addLeasing(id, leasing, blockID)
}

func (l *leases) cancelLeasingUncertain(id crypto.Digest, height uint64, txID *crypto.Digest) error {
	leasing, err := l.newestLeasingInfo(id)
	if err != nil {
		return errors.Errorf("failed to get leasing info: %v", err)
	}
	leasing.Status = LeaseCancelled
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

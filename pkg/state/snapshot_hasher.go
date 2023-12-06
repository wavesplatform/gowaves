package state

import (
	"bytes"
	"encoding/binary"
	"sort"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	byteSize   = 1 // must be equal sizeof(byte)
	boolSize   = 1 // must be equal sizeof(bool)
	uint32Size = 4 // must be equal sizeof(uint32)
	uint64Size = 8 // must be equal sizeof(uint64)
)

type hashEntry struct {
	_    struct{}
	data []byte
}

type txSnapshotHasher struct {
	hashEntries   []hashEntry
	blockHeight   proto.Height
	transactionID crypto.Digest
}

var _ = proto.SnapshotApplier((*txSnapshotHasher)(nil)) // use the same interface for applying and hashing

var _ = newTxSnapshotHasher // only for linter

func newTxSnapshotHasher(blockHeight proto.Height, transactionID crypto.Digest) txSnapshotHasher {
	return txSnapshotHasher{
		hashEntries:   nil,
		blockHeight:   blockHeight,
		transactionID: transactionID,
	}
}

func writeUint32BigEndian(w *bytes.Buffer, v uint32) error {
	var buf [uint32Size]byte
	binary.BigEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeUint64BigEndian(w *bytes.Buffer, v uint64) error {
	var buf [uint64Size]byte
	binary.BigEndian.PutUint64(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeBool(w *bytes.Buffer, v bool) error {
	var b byte
	if v {
		b = 1
	}
	return w.WriteByte(b)
}

func (h *txSnapshotHasher) Release() {
	// no-op for now
}

func (h *txSnapshotHasher) CalculateHash(prevHash crypto.Digest) (crypto.Digest, error) {
	// scala node uses stable sort, thought it's unnecessary to use stable sort because:
	// - every byte sequence is unique for each snapshot
	// - if two byte sequences are equal then they are indistinguishable and order doesn't matter
	sort.Slice(h.hashEntries, func(i, j int) bool {
		return bytes.Compare(h.hashEntries[i].data, h.hashEntries[j].data) == -1
	})

	fh, errH := crypto.NewFastHash() // TODO: make hasher txSnapshotHasher field?
	if errH != nil {
		return crypto.Digest{}, errors.Wrap(errH, "failed to create new fast blake2b hasher")
	}

	for i, entry := range h.hashEntries {
		if _, err := fh.Write(entry.data); err != nil {
			return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher %d-th hash entry", i)
		}
	}
	var txSnapshotsDigest crypto.Digest
	fh.Sum(txSnapshotsDigest[:0])

	fh.Reset() // reuse the same hasher
	if _, err := fh.Write(prevHash[:]); err != nil {
		return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher previous tx state snapshot hash")
	}
	if _, err := fh.Write(txSnapshotsDigest[:]); err != nil {
		return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher current tx snapshots hash")
	}
	var txStateSnapshotDigest crypto.Digest
	fh.Sum(txStateSnapshotDigest[:0])

	return txStateSnapshotDigest, nil
}

func (h *txSnapshotHasher) ApplyWavesBalance(snapshot proto.WavesBalanceSnapshot) error {
	const size = len(snapshot.Address) + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Waves balances: address || balance
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.Balance); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyLeaseBalance(snapshot proto.LeaseBalanceSnapshot) error {
	const size = len(snapshot.Address) + uint64Size + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Lease balance: address || lease_in || lease_out
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.LeaseIn); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.LeaseOut); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetBalance(snapshot proto.AssetBalanceSnapshot) error {
	const size = len(snapshot.Address) + len(snapshot.AssetID) + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Asset balances: address || asset_id || balance
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.Balance); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAlias(snapshot proto.AliasSnapshot) error {
	size := len(snapshot.Address) + len(snapshot.Alias.Alias)
	var buf bytes.Buffer
	buf.Grow(size)

	// Alias: address || alias
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if _, err := buf.WriteString(snapshot.Alias.Alias); err != nil { // we assume that string is valid UTF-8
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyNewAsset(snapshot proto.NewAssetSnapshot) error {
	const size = len(snapshot.AssetID) + len(snapshot.IssuerPublicKey) + byteSize + boolSize
	var buf bytes.Buffer
	buf.Grow(size)

	// Static asset info: asset_id || issuer || decimals || is_nft
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.IssuerPublicKey[:]); err != nil {
		return err
	}
	if err := buf.WriteByte(snapshot.Decimals); err != nil {
		return err
	}
	if err := writeBool(&buf, snapshot.IsNFT); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetDescription(snapshot proto.AssetDescriptionSnapshot) error {
	size := len(snapshot.AssetID) + len(snapshot.AssetName) + len(snapshot.AssetDescription) + uint32Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Asset name and description: asset_id || name || description || change_height
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if _, err := buf.WriteString(snapshot.AssetName); err != nil { // we assume that string is valid UTF-8
		return err
	}
	if _, err := buf.WriteString(snapshot.AssetDescription); err != nil { // we assume that string is valid UTF-8
		return err
	}
	// in scala node height is hashed as 4 byte integer
	if err := writeUint32BigEndian(&buf, uint32(h.blockHeight)); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetVolume(snapshot proto.AssetVolumeSnapshot) error {
	totalQuantityBytes := snapshot.TotalQuantity.Bytes() // here the number is represented in big-endian form
	size := len(snapshot.AssetID) + boolSize + len(totalQuantityBytes)
	var buf bytes.Buffer
	buf.Grow(size)

	// Asset reissuability: asset_id || is_reissuable || total_quantity
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeBool(&buf, snapshot.IsReissuable); err != nil {
		return err
	}
	if _, err := buf.Write(totalQuantityBytes); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetScript(snapshot proto.AssetScriptSnapshot) error {
	size := len(snapshot.AssetID) + len(snapshot.Script)
	var buf bytes.Buffer
	buf.Grow(size)

	// Asset script: asset_id || script
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.Script); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplySponsorship(snapshot proto.SponsorshipSnapshot) error {
	const size = len(snapshot.AssetID) + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Sponsorship: asset_id || min_sponsored_fee
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.MinSponsoredFee); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyAccountScript(snapshot proto.AccountScriptSnapshot) error {
	if snapshot.Script.IsEmpty() {
		var buf bytes.Buffer
		const size = len(snapshot.SenderPublicKey)
		buf.Grow(size)

		// Emtpy account script: sender_public_key

		if _, err := buf.Write(snapshot.SenderPublicKey[:]); err != nil {
			return err
		}
		h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
		return nil
	}

	var buf bytes.Buffer
	size := len(snapshot.SenderPublicKey) + len(snapshot.Script) + uint64Size
	buf.Grow(size)

	// Not emtpy account script: sender_public_key || script || verifier_complexity
	if _, err := buf.Write(snapshot.SenderPublicKey[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.Script); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.VerifierComplexity); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyFilledVolumeAndFee(snapshot proto.FilledVolumeFeeSnapshot) error {
	const size = len(snapshot.OrderID) + uint64Size + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Filled volume and fee: order_id || filled_volume || filled_fee
	if _, err := buf.Write(snapshot.OrderID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.FilledVolume); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.FilledFee); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyDataEntries(snapshot proto.DataEntriesSnapshot) error {
	for i, entry := range snapshot.DataEntries {
		entryValue, marshalErr := entry.MarshalValue() // TODO: use writer methods
		if marshalErr != nil {
			return errors.Wrapf(marshalErr, "failed to marshal (%d) data entry (%T) for addr %s", i, entry,
				snapshot.Address,
			)
		}
		entryKey := entry.GetKey()

		size := len(snapshot.Address) + len(entryKey) + len(entryValue)
		var buf bytes.Buffer
		buf.Grow(size)

		// Data entries: address || key || data_entry
		if _, err := buf.Write(snapshot.Address[:]); err != nil {
			return err
		}
		if _, err := buf.WriteString(entryKey); err != nil { // we assume that string is valid UTF-8
			return err
		}
		if _, err := buf.Write(entryValue); err != nil {
			return err
		}

		h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	}
	return nil
}

func (h *txSnapshotHasher) applyLeaseStatusHashEntry(leaseID crypto.Digest, isActive bool) error {
	const size = len(leaseID) + boolSize
	var buf bytes.Buffer
	buf.Grow(size)

	// Lease details: lease_id || is_active
	if _, err := buf.Write(leaseID[:]); err != nil {
		return err
	}
	if err := writeBool(&buf, isActive); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

func (h *txSnapshotHasher) ApplyNewLease(snapshot proto.NewLeaseSnapshot) error {
	const size = len(snapshot.LeaseID) + len(snapshot.SenderPK) + len(snapshot.RecipientAddr) + uint64Size
	var buf bytes.Buffer
	buf.Grow(size)

	// Lease details: lease_id || sender_public_key || recipient || amount
	if _, err := buf.Write(snapshot.LeaseID[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.SenderPK[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.RecipientAddr[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(&buf, snapshot.Amount); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return h.applyLeaseStatusHashEntry(snapshot.LeaseID, true)
}

func (h *txSnapshotHasher) ApplyCancelledLease(snapshot proto.CancelledLeaseSnapshot) error {
	return h.applyLeaseStatusHashEntry(snapshot.LeaseID, false)
}

func (h *txSnapshotHasher) ApplyTransactionsStatus(snapshot proto.TransactionStatusSnapshot) error {
	// Application status is one byte, either 0x01 (script execution failed) or 0x02 (elided).
	var applicationStatus byte
	switch v := snapshot.Status; v {
	case proto.TransactionSucceeded:
		return nil // don't hash transaction status snapshot in case of successful transaction
	case proto.TransactionFailed:
		applicationStatus = 1
	case proto.TransactionElided:
		applicationStatus = 2
	default:
		return errors.Errorf("invalid status value (%d) of TransactionStatus snapshot", v)
	}

	const size = len(h.transactionID) + byteSize
	var buf bytes.Buffer
	buf.Grow(size)

	// Non-successful transaction application status: tx_id || application_status
	if _, err := buf.Write(h.transactionID[:]); err != nil {
		return err
	}
	if err := buf.WriteByte(applicationStatus); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf.Bytes()})
	return nil
}

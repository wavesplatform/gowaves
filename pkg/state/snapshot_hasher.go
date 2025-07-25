package state

import (
	"bytes"
	"encoding/binary"
	"hash"
	"sort"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	uint32Size = 4 // must be equal sizeof(uint32)
	uint64Size = 8 // must be equal sizeof(uint64)
)

func CalculateSnapshotStateHash(
	scheme proto.Scheme,
	height proto.Height,
	initSh crypto.Digest,
	txs []proto.Transaction,
	txSnapshots [][]proto.AtomicSnapshot,
) (crypto.Digest, error) {
	if len(txs) != len(txSnapshots) { // sanity check
		return crypto.Digest{}, errors.Errorf("different number of transactions (%d) and tx snapshots (%d)",
			len(txs), len(txSnapshots),
		)
	}
	hasher, err := newTxSnapshotHasherDefault()
	if err != nil {
		return crypto.Digest{}, errors.Wrapf(err, "failed to create tx snapshot default hasher, block height is %d", height)
	}
	defer hasher.Release()
	curSh := initSh
	for i, ts := range txSnapshots {
		id, errID := txs[i].GetID(scheme)
		if errID != nil {
			return crypto.Digest{}, errors.Wrapf(errID, "failed to get transaction ID")
		}
		txSh, shErr := calculateTxSnapshotStateHash(hasher, id, height, curSh, ts)
		if shErr != nil {
			return crypto.Digest{}, errors.Wrapf(shErr, "failed to calculate tx snapshot hash for txID %q at height %d",
				base58.Encode(id), height,
			)
		}
		curSh = txSh
	}
	return curSh, nil
}

type hashEntry struct {
	_    struct{}
	data *bytebufferpool.ByteBuffer
}

func (e *hashEntry) Release() {
	if e.data != nil {
		bytebufferpool.Put(e.data)
		e.data = nil
	}
}

type hashEntries []hashEntry

func (h hashEntries) Len() int { return len(h) }

func (h hashEntries) Less(i, j int) bool { return bytes.Compare(h[i].data.B, h[j].data.B) == -1 }

func (h hashEntries) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

type txSnapshotHasher struct {
	fastHasher    hash.Hash
	hashEntries   hashEntries
	blockHeight   proto.Height
	transactionID []byte
}

var _ = proto.SnapshotApplier((*txSnapshotHasher)(nil)) // use the same interface for applying and hashing

func newTxSnapshotHasherDefault() (*txSnapshotHasher, error) {
	return newTxSnapshotHasher(0, nil)
}

func newTxSnapshotHasher(blockHeight proto.Height, transactionID []byte) (*txSnapshotHasher, error) {
	fastHasher, err := crypto.NewFastHash()
	if err != nil {
		return nil, err
	}
	return &txSnapshotHasher{
		fastHasher:    fastHasher,
		hashEntries:   nil,
		blockHeight:   blockHeight,
		transactionID: transactionID,
	}, nil
}

func calculateTxSnapshotStateHash(
	h *txSnapshotHasher,
	txID []byte,
	blockHeight proto.Height,
	prevHash crypto.Digest,
	txSnapshot []proto.AtomicSnapshot,
) (crypto.Digest, error) {
	h.Reset(blockHeight, txID) // reset hasher before using

	for i, snapshot := range txSnapshot {
		if err := snapshot.Apply(h); err != nil {
			return crypto.Digest{}, errors.Wrapf(err, "failed to apply to hasher %d-th snapshot (%T)",
				i+1, snapshot,
			)
		}
	}
	return h.CalculateHash(prevHash)
}

func writeUint32BigEndian(w *bytebufferpool.ByteBuffer, v uint32) error {
	var buf [uint32Size]byte
	binary.BigEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeUint64BigEndian(w *bytebufferpool.ByteBuffer, v uint64) error {
	var buf [uint64Size]byte
	binary.BigEndian.PutUint64(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeBool(w *bytebufferpool.ByteBuffer, v bool) error {
	var b byte
	if v {
		b = 1
	}
	return w.WriteByte(b)
}

// Release releases the hasher and sets its state to default.
func (h *txSnapshotHasher) Release() {
	for _, e := range h.hashEntries {
		e.Release()
	}
	h.hashEntries = h.hashEntries[:0]
	h.blockHeight = 0
	h.transactionID = nil
	h.fastHasher.Reset()
}

// Reset releases the hasher and sets a new state.
func (h *txSnapshotHasher) Reset(blockHeight proto.Height, transactionID []byte) {
	h.Release()
	h.blockHeight = blockHeight
	h.transactionID = transactionID
}

func (h *txSnapshotHasher) CalculateHash(prevHash crypto.Digest) (crypto.Digest, error) {
	defer h.fastHasher.Reset() // reset saved hasher
	// scala node uses stable sort, thought it's unnecessary to use stable sort because:
	// - every byte sequence is unique for each snapshot
	// - if two byte sequences are equal then they are indistinguishable and order doesn't matter
	sort.Sort(h.hashEntries)

	for i, entry := range h.hashEntries {
		if _, err := h.fastHasher.Write(entry.data.Bytes()); err != nil {
			return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher %d-th hash entry", i)
		}
	}
	var txSnapshotsDigest crypto.Digest
	h.fastHasher.Sum(txSnapshotsDigest[:0])

	h.fastHasher.Reset() // reuse the same hasher
	if _, err := h.fastHasher.Write(prevHash[:]); err != nil {
		return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher previous tx state snapshot hash")
	}
	if _, err := h.fastHasher.Write(txSnapshotsDigest[:]); err != nil {
		return crypto.Digest{}, errors.Wrapf(err, "failed to write to hasher current tx snapshots hash")
	}
	var newHash crypto.Digest
	h.fastHasher.Sum(newHash[:0])

	return newHash, nil
}

func (h *txSnapshotHasher) ApplyWavesBalance(snapshot proto.WavesBalanceSnapshot) error {
	buf := bytebufferpool.Get()

	// Waves balances: address || balance
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.Balance); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyLeaseBalance(snapshot proto.LeaseBalanceSnapshot) error {
	buf := bytebufferpool.Get()

	// Lease balance: address || lease_in || lease_out
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.LeaseIn); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.LeaseOut); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetBalance(snapshot proto.AssetBalanceSnapshot) error {
	buf := bytebufferpool.Get()

	// Asset balances: address || asset_id || balance
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.Balance); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAlias(snapshot proto.AliasSnapshot) error {
	buf := bytebufferpool.Get()

	// Alias: address || alias
	if _, err := buf.Write(snapshot.Address[:]); err != nil {
		return err
	}
	if _, err := buf.WriteString(snapshot.Alias); err != nil { // we assume that string is valid UTF-8
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyNewAsset(snapshot proto.NewAssetSnapshot) error {
	buf := bytebufferpool.Get()

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
	if err := writeBool(buf, snapshot.IsNFT); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetDescription(snapshot proto.AssetDescriptionSnapshot) error {
	if h.blockHeight == 0 { // sanity check
		return errors.New("failed to apply asset description snapshot: block height is not set")
	}
	buf := bytebufferpool.Get()

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
	if err := writeUint32BigEndian(buf, uint32(h.blockHeight)); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetVolume(snapshot proto.AssetVolumeSnapshot) error {
	totalQuantityBytes := common.Encode2CBigInt(&snapshot.TotalQuantity)
	buf := bytebufferpool.Get()

	// Asset reissuability: asset_id || is_reissuable || total_quantity
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeBool(buf, snapshot.IsReissuable); err != nil {
		return err
	}
	if _, err := buf.Write(totalQuantityBytes); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAssetScript(snapshot proto.AssetScriptSnapshot) error {
	buf := bytebufferpool.Get()

	// Asset script: asset_id || script
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.Script); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplySponsorship(snapshot proto.SponsorshipSnapshot) error {
	buf := bytebufferpool.Get()

	// Sponsorship: asset_id || min_sponsored_fee
	if _, err := buf.Write(snapshot.AssetID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.MinSponsoredFee); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyAccountScript(snapshot proto.AccountScriptSnapshot) error {
	if snapshot.Script.IsEmpty() {
		buf := bytebufferpool.Get()

		// Empty account script: sender_public_key

		if _, err := buf.Write(snapshot.SenderPublicKey[:]); err != nil {
			return err
		}
		h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
		return nil
	}

	buf := bytebufferpool.Get()

	// Not empty account script: sender_public_key || script || verifier_complexity
	if _, err := buf.Write(snapshot.SenderPublicKey[:]); err != nil {
		return err
	}
	if _, err := buf.Write(snapshot.Script); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.VerifierComplexity); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyFilledVolumeAndFee(snapshot proto.FilledVolumeFeeSnapshot) error {
	buf := bytebufferpool.Get()

	// Filled volume and fee: order_id || filled_volume || filled_fee
	if _, err := buf.Write(snapshot.OrderID[:]); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.FilledVolume); err != nil {
		return err
	}
	if err := writeUint64BigEndian(buf, snapshot.FilledFee); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyDataEntries(snapshot proto.DataEntriesSnapshot) error {
	for _, entry := range snapshot.DataEntries {
		entryKey := entry.GetKey()

		buf := bytebufferpool.Get()

		// Data entries: address || key || data_entry
		if _, err := buf.Write(snapshot.Address[:]); err != nil {
			return err
		}
		if _, err := buf.WriteString(entryKey); err != nil { // we assume that string is valid UTF-8
			return err
		}
		if err := entry.WriteValueTo(buf); err != nil {
			return err
		}

		h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	}
	return nil
}

func (h *txSnapshotHasher) applyLeaseStatusHashEntry(leaseID crypto.Digest, isActive bool) error {
	buf := bytebufferpool.Get()

	// Lease details: lease_id || is_active
	if _, err := buf.Write(leaseID[:]); err != nil {
		return err
	}
	if err := writeBool(buf, isActive); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

func (h *txSnapshotHasher) ApplyNewLease(snapshot proto.NewLeaseSnapshot) error {
	buf := bytebufferpool.Get()

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
	if err := writeUint64BigEndian(buf, snapshot.Amount); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return h.applyLeaseStatusHashEntry(snapshot.LeaseID, true)
}

func (h *txSnapshotHasher) ApplyCancelledLease(snapshot proto.CancelledLeaseSnapshot) error {
	return h.applyLeaseStatusHashEntry(snapshot.LeaseID, false)
}

func (h *txSnapshotHasher) ApplyTransactionsStatus(snapshot proto.TransactionStatusSnapshot) error {
	if len(h.transactionID) == 0 { // sanity check
		return errors.New("failed to apply transaction status snapshot: transaction ID is not set")
	}
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

	buf := bytebufferpool.Get()

	// Non-successful transaction application status: tx_id || application_status
	if _, err := buf.Write(h.transactionID); err != nil {
		return err
	}
	if err := buf.WriteByte(applicationStatus); err != nil {
		return err
	}

	h.hashEntries = append(h.hashEntries, hashEntry{data: buf})
	return nil
}

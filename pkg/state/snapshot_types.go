package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type TransactionSnapshot []AtomicSnapshot

func SplitSnapshots(atomicSnapshots []AtomicSnapshot) ([]AtomicSnapshot, []AtomicSnapshot) {
	var snapshots []AtomicSnapshot
	var internalSnapshots []AtomicSnapshot
	for _, snapshot := range atomicSnapshots {
		if !snapshot.IsInternal() {
			snapshots = append(snapshots, snapshot)
		} else {
			internalSnapshots = append(internalSnapshots, snapshot)
		}
	}
	return snapshots, internalSnapshots
}

func (ts TransactionSnapshot) Apply(a SnapshotApplier) error {
	mainSnapshots, internalSnapshots := SplitSnapshots(ts)
	// internal snapshots must be applied at the end
	for _, mainSnapshot := range mainSnapshots {
		if !mainSnapshot.IsGeneratedByTxDiff() {
			err := mainSnapshot.Apply(a)
			if err != nil {
				return errors.Wrap(err, "failed to apply main transaction snapshot")
			}
		}
	}

	for _, internalSnapshot := range internalSnapshots {
		if !internalSnapshot.IsGeneratedByTxDiff() {
			err := internalSnapshot.Apply(a)
			if err != nil {
				return errors.Wrap(err, "failed to apply internal transaction snapshot")
			}
		}
	}
	return nil
}

type AtomicSnapshot interface {
	Apply(SnapshotApplier) error
	/* is temporarily used to mark snapshots generated by tx diff that shouldn't be applied,
	because balances diffs are applied later in the block. */
	IsGeneratedByTxDiff() bool
	IsInternal() bool
}

type WavesBalanceSnapshot struct {
	Address proto.WavesAddress
	Balance uint64
}

func (s WavesBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s WavesBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyWavesBalance(s) }

type AssetBalanceSnapshot struct {
	Address proto.WavesAddress
	AssetID crypto.Digest
	Balance uint64
}

func (s WavesBalanceSnapshot) IsInternal() bool {
	return false
}

func (s AssetBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s AssetBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetBalance(s) }

func (s AssetBalanceSnapshot) IsInternal() bool {
	return false
}

type DataEntriesSnapshot struct { // AccountData in pb
	Address     proto.WavesAddress
	DataEntries []proto.DataEntry
}

func (s DataEntriesSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s DataEntriesSnapshot) Apply(a SnapshotApplier) error { return a.ApplyDataEntries(s) }

func (s DataEntriesSnapshot) IsInternal() bool {
	return false
}

type AccountScriptSnapshot struct {
	SenderPublicKey    crypto.PublicKey
	Script             proto.Script
	VerifierComplexity uint64
}

func (s AccountScriptSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AccountScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAccountScript(s) }

func (s AccountScriptSnapshot) IsInternal() bool {
	return false
}

type AssetScriptSnapshot struct {
	AssetID            crypto.Digest
	Script             proto.Script
	SenderPK           crypto.PublicKey // should be removed later
	VerifierComplexity uint64
}

func (s AssetScriptSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetScript(s) }

func (s AssetScriptSnapshot) IsInternal() bool {
	return false
}

type LeaseBalanceSnapshot struct {
	Address  proto.WavesAddress
	LeaseIn  uint64
	LeaseOut uint64
}

func (s LeaseBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s LeaseBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseBalance(s) }

func (s LeaseBalanceSnapshot) IsInternal() bool {
	return false
}

type LeaseStateStatus struct {
	Value               LeaseStatus // can be only LeaseActive or LeaseCanceled
	CancelHeight        proto.Height
	CancelTransactionID *crypto.Digest
}

type LeaseStateSnapshot struct {
	LeaseID             crypto.Digest
	Status              LeaseStateStatus
	Amount              uint64
	Sender              proto.WavesAddress
	Recipient           proto.WavesAddress
	OriginTransactionID *crypto.Digest
	Height              proto.Height
}

func (s LeaseStateSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s LeaseStateSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseState(s) }

func (s LeaseStateSnapshot) IsInternal() bool {
	return false
}

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest
	MinSponsoredFee uint64
}

func (s SponsorshipSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s SponsorshipSnapshot) Apply(a SnapshotApplier) error { return a.ApplySponsorship(s) }

func (s SponsorshipSnapshot) IsInternal() bool {
	return false
}

type AliasSnapshot struct {
	Address proto.WavesAddress
	Alias   proto.Alias
}

func (s AliasSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AliasSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAlias(s) }

func (s AliasSnapshot) IsInternal() bool {
	return false
}

// FilledVolumeFeeSnapshot Filled Volume and Fee.
type FilledVolumeFeeSnapshot struct { // OrderFill
	OrderID      crypto.Digest
	FilledVolume uint64
	FilledFee    uint64
}

func (s FilledVolumeFeeSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s FilledVolumeFeeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyFilledVolumeAndFee(s) }

func (s FilledVolumeFeeSnapshot) IsInternal() bool {
	return false
}

type StaticAssetInfoSnapshot struct {
	AssetID             crypto.Digest
	SourceTransactionID crypto.Digest
	IssuerPublicKey     crypto.PublicKey
	Decimals            uint8
	IsNFT               bool
}

func (s StaticAssetInfoSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s StaticAssetInfoSnapshot) Apply(a SnapshotApplier) error { return a.ApplyStaticAssetInfo(s) }

func (s StaticAssetInfoSnapshot) IsInternal() bool {
	return false
}

type AssetVolumeSnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest
	TotalQuantity big.Int // volume in protobuf
	IsReissuable  bool
}

func (s AssetVolumeSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetVolumeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetVolume(s) }

func (s AssetVolumeSnapshot) IsInternal() bool {
	return false
}

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest
	AssetName        string
	AssetDescription string
	ChangeHeight     proto.Height // last_updated in pb
}

func (s AssetDescriptionSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetDescriptionSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetDescription(s) }

func (s AssetDescriptionSnapshot) IsInternal() bool {
	return false
}

/*
Below are internal snapshots only.
They are not necessary and used for optimization, initialized in the full node mode only.
*/
type internalDAppComplexitySnapshot struct {
	scriptAddress proto.WavesAddress
	estimation    ride.TreeEstimation
	update        bool
}

func (s internalDAppComplexitySnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s internalDAppComplexitySnapshot) Apply(a SnapshotApplier) error {
	return a.applyInternalDAppComplexitySnapshot(s)
}

func (s internalDAppComplexitySnapshot) IsInternal() bool {
	return true
}

type SnapshotApplier interface {
	ApplyWavesBalance(snapshot WavesBalanceSnapshot) error
	ApplyLeaseBalance(snapshot LeaseBalanceSnapshot) error
	ApplyAssetBalance(snapshot AssetBalanceSnapshot) error
	ApplyAlias(snapshot AliasSnapshot) error
	ApplyStaticAssetInfo(snapshot StaticAssetInfoSnapshot) error
	ApplyAssetDescription(snapshot AssetDescriptionSnapshot) error
	ApplyAssetVolume(snapshot AssetVolumeSnapshot) error
	ApplyAssetScript(snapshot AssetScriptSnapshot) error
	ApplySponsorship(snapshot SponsorshipSnapshot) error
	ApplyAccountScript(snapshot AccountScriptSnapshot) error
	ApplyFilledVolumeAndFee(snapshot FilledVolumeFeeSnapshot) error
	ApplyDataEntries(snapshot DataEntriesSnapshot) error
	ApplyLeaseState(snapshot LeaseStateSnapshot) error

	/* Internal snapshots. Applied only in the full node mode */
	applyInternalDAppComplexitySnapshot(internalSnapshot internalDAppComplexitySnapshot) error
}

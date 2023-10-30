package proto

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AtomicSnapshot interface {
	Apply(SnapshotApplier) error
	/* is temporarily used to mark snapshots generated by tx diff that shouldn't be applied,
	because balances diffs are applied later in the block. */
	IsGeneratedByTxDiff() bool
}

type WavesBalanceSnapshot struct {
	Address WavesAddress
	Balance uint64
}

func (s WavesBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s WavesBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyWavesBalance(s) }

type AssetBalanceSnapshot struct {
	Address WavesAddress
	AssetID crypto.Digest
	Balance uint64
}

func (s AssetBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s AssetBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetBalance(s) }

type DataEntriesSnapshot struct { // AccountData in pb
	Address     WavesAddress
	DataEntries []DataEntry
}

func (s DataEntriesSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s DataEntriesSnapshot) Apply(a SnapshotApplier) error { return a.ApplyDataEntries(s) }

type AccountScriptSnapshot struct {
	SenderPublicKey    crypto.PublicKey
	Script             Script
	VerifierComplexity uint64
}

func (s AccountScriptSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AccountScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAccountScript(s) }

type AssetScriptSnapshot struct {
	AssetID crypto.Digest
	Script  Script
}

func (s AssetScriptSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetScriptSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetScript(s) }

type LeaseBalanceSnapshot struct {
	Address  WavesAddress
	LeaseIn  uint64
	LeaseOut uint64
}

func (s LeaseBalanceSnapshot) IsGeneratedByTxDiff() bool {
	return true
}

func (s LeaseBalanceSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseBalance(s) }

type LeaseStateStatus struct {
	Value               LeaseStatus // can be only LeaseActive or LeaseCanceled
	CancelHeight        Height
	CancelTransactionID *crypto.Digest
}

type LeaseStateSnapshot struct {
	LeaseID             crypto.Digest
	Status              LeaseStateStatus
	Amount              uint64
	Sender              WavesAddress
	Recipient           WavesAddress
	OriginTransactionID *crypto.Digest
	Height              Height
}

func (s LeaseStateSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s LeaseStateSnapshot) Apply(a SnapshotApplier) error { return a.ApplyLeaseState(s) }

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest
	MinSponsoredFee uint64
}

func (s SponsorshipSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s SponsorshipSnapshot) Apply(a SnapshotApplier) error { return a.ApplySponsorship(s) }

type AliasSnapshot struct {
	Address WavesAddress
	Alias   Alias
}

func (s AliasSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AliasSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAlias(s) }

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

type AssetVolumeSnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest
	TotalQuantity big.Int // volume in protobuf
	IsReissuable  bool
}

func (s AssetVolumeSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetVolumeSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetVolume(s) }

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest
	AssetName        string
	AssetDescription string
	ChangeHeight     Height // last_updated in pb
}

func (s AssetDescriptionSnapshot) IsGeneratedByTxDiff() bool {
	return false
}

func (s AssetDescriptionSnapshot) Apply(a SnapshotApplier) error { return a.ApplyAssetDescription(s) }

type TransactionStatusSnapshot struct {
	TransactionID crypto.Digest
	Status        TransactionStatus
}

func (s TransactionStatusSnapshot) Apply(a SnapshotApplier) error {
	return a.ApplyTransactionsStatus(s)
}

func (s TransactionStatusSnapshot) IsGeneratedByTxDiff() bool {
	return false
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
	ApplyTransactionsStatus(snapshot TransactionStatusSnapshot) error
}

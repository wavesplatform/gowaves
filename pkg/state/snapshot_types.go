package state

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionSnapshot []AtomicSnapshot

type AtomicSnapshot interface{ Apply(SnapshotManager) error }

type WavesBalanceSnapshot struct {
	Address proto.WavesAddress
	Balance uint64
}

func (s WavesBalanceSnapshot) Apply(m SnapshotManager) error { return m.ApplyWavesBalance(s) }

type AssetBalanceSnapshot struct {
	Address proto.WavesAddress
	AssetID crypto.Digest
	Balance uint64
}

func (s AssetBalanceSnapshot) Apply(m SnapshotManager) error { return m.ApplyAssetBalance(s) }

type DataEntriesSnapshot struct { // AccountData in pb
	Address     proto.WavesAddress
	DataEntries []proto.DataEntry
}

func (s DataEntriesSnapshot) Apply(m SnapshotManager) error { return m.ApplyDataEntries(s) }

type AccountScriptSnapshot struct {
	SenderPublicKey    crypto.PublicKey
	Script             proto.Script
	VerifierComplexity uint64
}

func (s AccountScriptSnapshot) Apply(m SnapshotManager) error { return m.ApplyAccountScript(s) }

type AssetScriptSnapshot struct {
	AssetID    crypto.Digest
	Script     proto.Script
	Complexity uint64
}

func (s AssetScriptSnapshot) Apply(m SnapshotManager) error { return m.ApplyAssetScript(s) }

type LeaseBalanceSnapshot struct {
	Address  proto.WavesAddress
	LeaseIn  uint64
	LeaseOut uint64
}

func (s LeaseBalanceSnapshot) Apply(m SnapshotManager) error { return m.ApplyLeaseBalance(s) }

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

func (s LeaseStateSnapshot) Apply(m SnapshotManager) error { return m.ApplyLeaseState(s) }

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest
	MinSponsoredFee uint64
}

func (s SponsorshipSnapshot) Apply(m SnapshotManager) error { return m.ApplySponsorship(s) }

type AliasSnapshot struct {
	Address proto.WavesAddress
	Alias   proto.Alias
}

func (s AliasSnapshot) Apply(m SnapshotManager) error { return m.ApplyAlias(s) }

// FilledVolumeFeeSnapshot Filled Volume and Fee
type FilledVolumeFeeSnapshot struct { // OrderFill
	OrderID      crypto.Digest
	FilledVolume uint64
	FilledFee    uint64
}

func (s FilledVolumeFeeSnapshot) Apply(m SnapshotManager) error { return m.ApplyFilledVolumeAndFee(s) }

type StaticAssetInfoSnapshot struct {
	AssetID             crypto.Digest
	SourceTransactionID crypto.Digest
	IssuerPublicKey     crypto.PublicKey
	Decimals            uint8
	IsNFT               bool
}

func (s StaticAssetInfoSnapshot) Apply(m SnapshotManager) error { return m.ApplyStaticAssetInfo(s) }

type AssetVolumeSnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest
	TotalQuantity big.Int // volume in protobuf
	IsReissuable  bool
}

func (s AssetVolumeSnapshot) Apply(m SnapshotManager) error { return m.ApplyAssetVolume(s) }

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest
	AssetName        string
	AssetDescription string
	ChangeHeight     proto.Height // last_updated in pb
}

func (s AssetDescriptionSnapshot) Apply(m SnapshotManager) error { return m.ApplyAssetDescription(s) }

type SnapshotManager interface {
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
}

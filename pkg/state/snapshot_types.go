package state

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionSnapshot []AtomicSnapshot

type AtomicSnapshot interface {
	atomicSnapshotMarker()
	// TODO: add all necessary methods here
}

type WavesBalanceSnapshot struct {
	Address proto.WavesAddress
	Balance uint64
}

func (*WavesBalanceSnapshot) atomicSnapshotMarker() {}

// What is address || asset_id?
type AssetBalanceSnapshot struct {
	Address proto.WavesAddress
	AssetID crypto.Digest
	Balance uint64
}

func (*AssetBalanceSnapshot) atomicSnapshotMarker() {}

type DataEntriesSnapshot struct { // AccountData in pb
	Address     proto.WavesAddress
	DataEntries []proto.DataEntry
}

func (*DataEntriesSnapshot) atomicSnapshotMarker() {}

type AccountScriptSnapshot struct {
	SenderPublicKey    crypto.PublicKey
	Script             proto.Script
	VerifierComplexity uint64
}

func (*AccountScriptSnapshot) atomicSnapshotMarker() {}

type AssetScriptSnapshot struct {
	AssetID    crypto.Digest
	Script     proto.Script
	Complexity uint64
}

func (*AssetScriptSnapshot) atomicSnapshotMarker() {}

type LeaseBalanceSnapshot struct {
	Address  proto.WavesAddress
	LeaseIn  uint64
	LeaseOut uint64
}

func (*LeaseBalanceSnapshot) atomicSnapshotMarker() {}

type LeaseStateSnapshot struct {
	LeaseID             crypto.Digest
	Status              LeaseStatus // TODO(nickeskov): add cancelHeight and cancelTxID info for canceled leases
	Amount              uint64
	Sender              proto.WavesAddress
	Recipient           proto.WavesAddress
	OriginTransactionID crypto.Digest
	Height              proto.Height
}

func (*LeaseStateSnapshot) atomicSnapshotMarker() {}

type SponsorshipSnapshot struct {
	AssetID         crypto.Digest
	MinSponsoredFee uint64
}

func (*SponsorshipSnapshot) atomicSnapshotMarker() {}

type AliasSnapshot struct {
	Address proto.WavesAddress
	Alias   proto.Alias
}

func (*AliasSnapshot) atomicSnapshotMarker() {}

// FilledVolumeFeeSnapshot Filled Volume and Fee
type FilledVolumeFeeSnapshot struct { // OrderFill
	OrderID      crypto.Digest
	FilledVolume uint64
	FilledFee    uint64
}

func (*FilledVolumeFeeSnapshot) atomicSnapshotMarker() {}

type StaticAssetInfoSnapshot struct {
	AssetID             crypto.Digest
	SourceTransactionID crypto.Digest
	IssuerPublicKey     crypto.PublicKey
	Decimals            uint8
	IsNFT               bool
}

func (*StaticAssetInfoSnapshot) atomicSnapshotMarker() {}

type AssetReissuabilitySnapshot struct { // AssetVolume in pb
	AssetID       crypto.Digest
	TotalQuantity big.Int // volume in protobuf
	IsReissuable  bool
}

func (*AssetReissuabilitySnapshot) atomicSnapshotMarker() {}

type AssetDescriptionSnapshot struct { // AssetNameAndDescription in pb
	AssetID          crypto.Digest
	AssetName        string
	AssetDescription string
	ChangeHeight     proto.Height // last_updated in pb
}

func (*AssetDescriptionSnapshot) atomicSnapshotMarker() {}

type SnapshotManager interface {
	// TODO: add all necessary methods here
}

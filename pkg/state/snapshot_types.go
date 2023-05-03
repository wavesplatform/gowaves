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

type balanceWaves struct {
	address proto.WavesAddress
	balance uint64
}

type WavesBalancesSnapshot struct {
	wavesBalances []balanceWaves
}

func (*WavesBalancesSnapshot) atomicSnapshotMarker() {}

// What is address || asset_id?
type balanceAsset struct {
	address proto.WavesAddress
	assetID crypto.Digest
	balance uint64
}

type AssetBalancesSnapshot struct {
	assetBalances []balanceAsset
}

func (*AssetBalancesSnapshot) atomicSnapshotMarker() {}

type DataEntriesSnapshot struct {
	address     proto.WavesAddress
	dataEntries []proto.DataEntry
}

func (*DataEntriesSnapshot) atomicSnapshotMarker() {}

type AccountScriptSnapshot struct {
	address proto.WavesAddress
	script  proto.Script
}

func (*AccountScriptSnapshot) atomicSnapshotMarker() {}

type AssetScriptSnapshot struct {
	assetID crypto.Digest
	script  proto.Script
}

func (*AssetScriptSnapshot) atomicSnapshotMarker() {}

type LeaseBalanceSnapshot struct {
	address  proto.WavesAddress
	leaseIn  int64
	leaseOut int64
}

func (*LeaseBalanceSnapshot) atomicSnapshotMarker() {}

type LeaseStatusSnapshot struct {
	leaseID  crypto.Digest
	isActive bool
}

func (*LeaseStatusSnapshot) atomicSnapshotMarker() {}

type SponsorshipSnapshot struct {
	assetID         crypto.Digest
	minSponsoredFee uint64
}

func (*SponsorshipSnapshot) atomicSnapshotMarker() {}

type AliasSnapshot struct {
	alias   proto.Alias
	address proto.WavesAddress
}

func (*AliasSnapshot) atomicSnapshotMarker() {}

// FilledVolumeFee Filled Volume and Fee
type FilledVolumeFeeSnapshot struct {
	orderID      []byte
	filledVolume uint64
	filledFee    uint64
}

func (*FilledVolumeFeeSnapshot) atomicSnapshotMarker() {}

type StaticAssetInfoSnapshot struct {
	assetID  crypto.Digest
	issuer   proto.WavesAddress
	decimals int8
	isNFT    bool
}

func (*StaticAssetInfoSnapshot) atomicSnapshotMarker() {}

type AssetReissuabilitySnapshot struct {
	assetID       crypto.Digest
	totalQuantity big.Int
	isReissuable  bool
}

func (*AssetReissuabilitySnapshot) atomicSnapshotMarker() {}

type AssetDescriptionSnapshot struct {
	assetID          crypto.Digest
	assetName        string
	assetDescription string
	changeHeight     uint64
}

func (*AssetDescriptionSnapshot) atomicSnapshotMarker() {}

type SnapshotManager interface {
	// TODO: add all necessary methods here
}

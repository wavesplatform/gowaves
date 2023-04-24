package state

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionSnapshot []AtomicSnapshot

type AtomicSnapshot interface {
	dummy() error
}

type balanceWaves struct {
	address proto.Address
	balance uint64
}

type WavesBalancesSnapshot struct {
	wavesBalances []balanceWaves
}

func (s *WavesBalancesSnapshot) dummy() error {
	return nil
}

// What is address || asset_id?
type balanceAsset struct {
	address proto.Address
	assetID proto.AssetID
	balance uint64
}

type AssetBalancesSnapshot struct {
	assetBalances []balanceAsset
}

func (s *AssetBalancesSnapshot) dummy() error {
	return nil
}

type DataEntriesSnapshot struct {
	address     proto.Address
	dataEntries []proto.DataEntry
}

func (s *DataEntriesSnapshot) dummy() error {
	return nil
}

type AccountScriptSnapshot struct {
	address proto.Address
	script  proto.Script
}

func (s *AccountScriptSnapshot) dummy() error {
	return nil
}

type AssetScriptSnapshot struct {
	assetID proto.AssetID
	script  proto.Script
}

func (s *AssetScriptSnapshot) dummy() error {
	return nil
}

type LeaseBalanceSnapshot struct {
	address  proto.Address
	leaseIn  uint64
	leaseOut uint64
}

func (s *LeaseBalanceSnapshot) dummy() error {
	return nil
}

type LeaseStatusSnapshot struct {
	leaseID  uint64
	isActive bool
}

func (s *LeaseStatusSnapshot) dummy() error {
	return nil
}

type SponsorshipSnapshot struct {
	assetID         proto.AssetID
	minSponsoredFee uint64
}

func (s *SponsorshipSnapshot) dummy() error {
	return nil
}

type AliasSnapshot struct {
	alias   *proto.Alias
	address *proto.Address
}

func (s *AliasSnapshot) dummy() error {
	return nil
}

// FilledVolumeFee Filled Volume and Fee
type FilledVolumeFeeSnapshot struct {
	orderID      uint64
	filledVolume uint64
	filledFee    uint64
}

func (s *FilledVolumeFeeSnapshot) dummy() error {
	return nil
}

type StaticAssetInfoSnapshot struct {
	assetID  proto.AssetID
	issuer   proto.Address
	decimals int8
	isNFT    bool
}

func (s *StaticAssetInfoSnapshot) dummy() error {
	return nil
}

type AssetReissuabilitySnapshot struct {
	assetID       proto.AssetID
	totalQuantity big.Int
	isReissuable  bool
}

func (s *AssetReissuabilitySnapshot) dummy() error {
	return nil
}

type AssetDescriptionSnapshot struct {
	assetID          proto.AssetID
	assetName        string
	assetDescription string
	changeHeight     uint64
}

func (s *AssetDescriptionSnapshot) dummy() error {
	return nil
}

type SnapshotManager struct {
	stor   *blockchainEntitiesStorage
	scheme proto.Scheme
}

func NewSnapshotManager(stor *blockchainEntitiesStorage) *SnapshotManager {
	return &SnapshotManager{stor: stor}
}

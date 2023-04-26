package state

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionSnapshot []AtomicSnapshot

type AtomicSnapshot interface {
	dummy() error
}

type balanceWaves struct {
	address proto.WavesAddress
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
	address proto.WavesAddress
	assetID crypto.Digest
	balance uint64
}

type AssetBalancesSnapshot struct {
	assetBalances []balanceAsset
}

func (s *AssetBalancesSnapshot) dummy() error {
	return nil
}

type DataEntriesSnapshot struct {
	address     proto.WavesAddress
	dataEntries []proto.DataEntry
}

func (s *DataEntriesSnapshot) dummy() error {
	return nil
}

type AccountScriptSnapshot struct {
	address proto.WavesAddress
	script  proto.Script
}

func (s *AccountScriptSnapshot) dummy() error {
	return nil
}

type AssetScriptSnapshot struct {
	assetID crypto.Digest
	script  proto.Script
}

func (s *AssetScriptSnapshot) dummy() error {
	return nil
}

type LeaseBalanceSnapshot struct {
	address  proto.WavesAddress
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
	assetID         crypto.Digest
	minSponsoredFee uint64
}

func (s *SponsorshipSnapshot) dummy() error {
	return nil
}

type AliasSnapshot struct {
	alias   *proto.Alias
	address *proto.WavesAddress
}

func (s *AliasSnapshot) dummy() error {
	return nil
}

// FilledVolumeFee Filled Volume and Fee
type FilledVolumeFeeSnapshot struct {
	orderID      []byte
	filledVolume uint64
	filledFee    uint64
}

func (s *FilledVolumeFeeSnapshot) dummy() error {
	return nil
}

type StaticAssetInfoSnapshot struct {
	assetID  crypto.Digest
	issuer   proto.WavesAddress
	decimals int8
	isNFT    bool
}

func (s *StaticAssetInfoSnapshot) dummy() error {
	return nil
}

type AssetReissuabilitySnapshot struct {
	assetID       crypto.Digest
	totalQuantity big.Int
	isReissuable  bool
}

func (s *AssetReissuabilitySnapshot) dummy() error {
	return nil
}

type AssetDescriptionSnapshot struct {
	assetID          crypto.Digest
	assetName        string
	assetDescription string
	changeHeight     uint64
}

func (s *AssetDescriptionSnapshot) dummy() error {
	return nil
}

type SnapshotManager interface {
	// TODO: add all necessary methods here
}

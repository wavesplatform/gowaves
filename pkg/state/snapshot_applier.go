package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

type SnapshotApplierInfo interface {
	BlockID() proto.BlockID
	Height() proto.Height
	EstimatorVersion() int
	Scheme() proto.Scheme
	StateActionsCounter() *proto.StateActionsCounter
}

type SnapshotApplier interface {
	ApplyWavesBalance(info SnapshotApplierInfo, snapshot WavesBalanceSnapshot) error
	ApplyLeaseBalance(info SnapshotApplierInfo, snapshot LeaseBalanceSnapshot) error
	ApplyAssetBalance(info SnapshotApplierInfo, snapshot AssetBalanceSnapshot) error
	ApplyAlias(info SnapshotApplierInfo, snapshot AliasSnapshot) error
	ApplyStaticAssetInfo(info SnapshotApplierInfo, snapshot StaticAssetInfoSnapshot) error
	ApplyAssetDescription(info SnapshotApplierInfo, snapshot AssetDescriptionSnapshot) error
	ApplyAssetVolume(info SnapshotApplierInfo, snapshot AssetVolumeSnapshot) error
	ApplyAssetScript(info SnapshotApplierInfo, snapshot AssetScriptSnapshot) error
	ApplySponsorship(info SnapshotApplierInfo, snapshot SponsorshipSnapshot) error
	ApplyAccountScript(info SnapshotApplierInfo, snapshot AccountScriptSnapshot) error
	ApplyFilledVolumeAndFee(info SnapshotApplierInfo, snapshot FilledVolumeFeeSnapshot) error
	ApplyDataEntry(info SnapshotApplierInfo, snapshot DataEntriesSnapshot) error
	ApplyLeaseState(info SnapshotApplierInfo, snapshot LeaseStateSnapshot) error
}

type snapshotApplier struct {
	balances          *balances
	aliases           *aliases
	assets            *assets
	scriptsStorage    scriptStorageState
	scriptsComplexity *scriptsComplexity
	sponsoredAssets   *sponsoredAssets
	ordersVolumes     *ordersVolumes
	accountsDataStor  *accountsDataStorage
	leases            *leases
}

var _ = SnapshotApplier((*snapshotApplier)(nil))

type snapshotApplierInfo struct {
	ci                  *checkerInfo
	scheme              proto.Scheme
	stateActionsCounter *proto.StateActionsCounter
}

var _ = SnapshotApplierInfo(snapshotApplierInfo{})

func (s snapshotApplierInfo) BlockID() proto.BlockID {
	return s.ci.blockID
}

func (s snapshotApplierInfo) Height() proto.Height {
	return s.ci.height
}

func (s snapshotApplierInfo) EstimatorVersion() int {
	return s.ci.estimatorVersion()
}

func (s snapshotApplierInfo) Scheme() proto.Scheme {
	return s.scheme
}

func (s snapshotApplierInfo) StateActionsCounter() *proto.StateActionsCounter {
	return s.stateActionsCounter
}

func (a *snapshotApplier) ApplyWavesBalance(info SnapshotApplierInfo, snapshot WavesBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.balance = snapshot.Balance
	value := newWavesValue(profile, newProfile)
	if err := a.balances.setWavesBalance(addrID, value, info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *snapshotApplier) ApplyLeaseBalance(info SnapshotApplierInfo, snapshot LeaseBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.leaseIn = int64(snapshot.LeaseIn)
	newProfile.leaseOut = int64(snapshot.LeaseOut)
	value := newWavesValue(profile, newProfile)
	if err := a.balances.setWavesBalance(addrID, value, info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *snapshotApplier) ApplyAssetBalance(info SnapshotApplierInfo, snapshot AssetBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	return a.balances.setAssetBalance(addrID, assetID, snapshot.Balance, info.BlockID())
}

func (a *snapshotApplier) ApplyAlias(info SnapshotApplierInfo, snapshot AliasSnapshot) error {
	return a.aliases.createAlias(snapshot.Alias.Alias, snapshot.Address, info.BlockID())
}

func (a *snapshotApplier) ApplyStaticAssetInfo(info SnapshotApplierInfo, snapshot StaticAssetInfoSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(snapshot.AssetID),
			issuer:               snapshot.IssuerPublicKey,
			decimals:             snapshot.Decimals,
			issueHeight:          info.Height(),
			issueSequenceInBlock: info.StateActionsCounter().NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{},
	}
	return a.assets.issueAsset(assetID, assetFullInfo, info.BlockID())
}

func (a *snapshotApplier) ApplyAssetDescription(info SnapshotApplierInfo, snapshot AssetDescriptionSnapshot) error {
	change := &assetInfoChange{
		newName:        snapshot.AssetName,
		newDescription: snapshot.AssetDescription,
		newHeight:      snapshot.ChangeHeight,
	}
	return a.assets.updateAssetInfo(snapshot.AssetID, change, info.BlockID())
}

func (a *snapshotApplier) ApplyAssetVolume(info SnapshotApplierInfo, snapshot AssetVolumeSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo, err := a.assets.newestAssetInfo(assetID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest asset info for asset %q", snapshot.AssetID.String())
	}
	assetFullInfo.assetChangeableInfo.reissuable = snapshot.IsReissuable
	assetFullInfo.assetChangeableInfo.quantity = snapshot.TotalQuantity
	return a.assets.storeAssetInfo(assetID, assetFullInfo, info.BlockID())
}

func (a *snapshotApplier) ApplyAssetScript(info SnapshotApplierInfo, snapshot AssetScriptSnapshot) error {
	estimation := ride.TreeEstimation{ // TODO: use uint in TreeEstimation
		Estimation: int(snapshot.Complexity),
		Verifier:   int(snapshot.Complexity),
		Functions:  nil,
	}
	if err := a.scriptsComplexity.saveComplexitiesForAsset(snapshot.AssetID, estimation, info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to store asset script estimation for asset %q", snapshot.AssetID.String())
	}
	constInfo, err := a.assets.newestConstInfo(proto.AssetIDFromDigest(snapshot.AssetID)) // only issuer can set new asset script
	if err != nil {
		return errors.Wrapf(err, "failed to get const asset info for asset %q", snapshot.AssetID.String())
	}
	return a.scriptsStorage.setAssetScript(snapshot.AssetID, snapshot.Script, constInfo.issuer, info.BlockID())
}

func (a *snapshotApplier) ApplySponsorship(info SnapshotApplierInfo, snapshot SponsorshipSnapshot) error {
	return a.sponsoredAssets.sponsorAsset(snapshot.AssetID, snapshot.MinSponsoredFee, info.BlockID())
}

func (a *snapshotApplier) ApplyAccountScript(info SnapshotApplierInfo, snapshot AccountScriptSnapshot) error {
	addr, err := proto.NewAddressFromPublicKey(info.Scheme(), snapshot.SenderPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create address from scheme %d and PK %q",
			info.Scheme(), snapshot.SenderPublicKey.String())
	}
	var estimations treeEstimations
	if !snapshot.Script.IsEmpty() {
		tree, err := serialization.Parse(snapshot.Script)
		if err != nil {
			return errors.Wrapf(err, "failed to parse script from account script snapshot for addr %q", addr.String())
		}
		estimations, err = makeRideEstimations(tree, info.EstimatorVersion(), true)
		if err != nil {
			return errors.Wrapf(err, "failed to make account script estimations for addr %q", addr.String())
		}
	}
	if err := a.scriptsComplexity.saveComplexitiesForAddr(addr, estimations, info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to store account script estimation for addr %q", addr.String())
	}
	return a.scriptsStorage.setAccountScript(addr, snapshot.Script, snapshot.SenderPublicKey, info.BlockID())
}

func (a *snapshotApplier) ApplyFilledVolumeAndFee(info SnapshotApplierInfo, snapshot FilledVolumeFeeSnapshot) error {
	return a.ordersVolumes.increaseFilled(snapshot.OrderID.Bytes(), snapshot.FilledVolume, snapshot.FilledFee, info.BlockID())
}

func (a *snapshotApplier) ApplyDataEntry(info SnapshotApplierInfo, snapshot DataEntriesSnapshot) error {
	blockID := info.BlockID()
	for _, entry := range snapshot.DataEntries {
		if err := a.accountsDataStor.appendEntry(snapshot.Address, entry, blockID); err != nil {
			return errors.Wrapf(err, "failed to add entry (%T) for address %q", entry, snapshot.Address)
		}
	}
	return nil
}

func (a *snapshotApplier) ApplyLeaseState(info SnapshotApplierInfo, snapshot LeaseStateSnapshot) error {
	l := &leasing{
		Sender:              snapshot.Sender,
		Recipient:           snapshot.Recipient,
		Amount:              snapshot.Amount,
		Height:              snapshot.Height,
		Status:              snapshot.Status.Value,
		OriginTransactionID: snapshot.OriginTransactionID,
		CancelHeight:        snapshot.Status.CancelHeight,
		CancelTransactionID: snapshot.Status.CancelTransactionID,
	}
	return a.leases.addLeasing(snapshot.LeaseID, l, info.BlockID())
}

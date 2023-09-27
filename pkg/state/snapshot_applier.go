package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type blockSnapshotsApplier struct {
	info blockSnapshotsApplierInfo
	stor snapshotApplierStorages
}

var _ = newBlockSnapshotsApplier // TODO: only for linter, will be removed later

func newBlockSnapshotsApplier(info blockSnapshotsApplierInfo, stor snapshotApplierStorages) blockSnapshotsApplier {
	return blockSnapshotsApplier{info: info, stor: stor}
}

type snapshotApplierStorages struct {
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

var _ = newSnapshotApplierStorages // TODO: only for linter, will be removed later

func newSnapshotApplierStorages(stor *blockchainEntitiesStorage) snapshotApplierStorages {
	return snapshotApplierStorages{
		balances:          stor.balances,
		aliases:           stor.aliases,
		assets:            stor.assets,
		scriptsStorage:    stor.scriptsStorage,
		scriptsComplexity: stor.scriptsComplexity,
		sponsoredAssets:   stor.sponsoredAssets,
		ordersVolumes:     stor.ordersVolumes,
		accountsDataStor:  stor.accountsDataStor,
		leases:            stor.leases,
	}
}

var _ = SnapshotApplier((*blockSnapshotsApplier)(nil))

type blockSnapshotsApplierInfo struct {
	ci                  *checkerInfo
	scheme              proto.Scheme
	stateActionsCounter *proto.StateActionsCounter
}

var _ = newBlockSnapshotsApplierInfo

func newBlockSnapshotsApplierInfo(ci *checkerInfo, scheme proto.Scheme,
	cnt *proto.StateActionsCounter) blockSnapshotsApplierInfo {
	return blockSnapshotsApplierInfo{
		ci:                  ci,
		scheme:              scheme,
		stateActionsCounter: cnt,
	}
}

func (s blockSnapshotsApplierInfo) BlockID() proto.BlockID {
	return s.ci.blockID
}

func (s blockSnapshotsApplierInfo) Height() proto.Height {
	return s.ci.height
}

func (s blockSnapshotsApplierInfo) EstimatorVersion() int {
	return s.ci.estimatorVersion()
}

func (s blockSnapshotsApplierInfo) Scheme() proto.Scheme {
	return s.scheme
}

func (s blockSnapshotsApplierInfo) StateActionsCounter() *proto.StateActionsCounter {
	return s.stateActionsCounter
}

func (a *blockSnapshotsApplier) ApplyWavesBalance(snapshot WavesBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.stor.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.balance = snapshot.Balance
	value := newWavesValue(profile, newProfile)
	if err = a.stor.balances.setWavesBalance(addrID, value, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseBalance(snapshot LeaseBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	var err error
	profile, err := a.stor.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.leaseIn = int64(snapshot.LeaseIn)
	newProfile.leaseOut = int64(snapshot.LeaseOut)
	value := newWavesValue(profile, newProfile)
	if err = a.stor.balances.setWavesBalance(addrID, value, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyAssetBalance(snapshot AssetBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	return a.stor.balances.setAssetBalance(addrID, assetID, snapshot.Balance, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAlias(snapshot AliasSnapshot) error {
	return a.stor.aliases.createAlias(snapshot.Alias.Alias, snapshot.Address, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyStaticAssetInfo(snapshot StaticAssetInfoSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(snapshot.AssetID),
			issuer:               snapshot.IssuerPublicKey,
			decimals:             snapshot.Decimals,
			issueHeight:          a.info.Height(),
			issueSequenceInBlock: a.info.StateActionsCounter().NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{},
	}
	return a.stor.assets.issueAsset(assetID, assetFullInfo, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetDescription(snapshot AssetDescriptionSnapshot) error {
	change := &assetInfoChange{
		newName:        snapshot.AssetName,
		newDescription: snapshot.AssetDescription,
		newHeight:      snapshot.ChangeHeight,
	}
	return a.stor.assets.updateAssetInfo(snapshot.AssetID, change, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetVolume(snapshot AssetVolumeSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo, err := a.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest asset info for asset %q", snapshot.AssetID.String())
	}
	assetFullInfo.assetChangeableInfo.reissuable = snapshot.IsReissuable
	assetFullInfo.assetChangeableInfo.quantity = snapshot.TotalQuantity
	return a.stor.assets.storeAssetInfo(assetID, assetFullInfo, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetScript(snapshot AssetScriptSnapshot) error {
	treeEstimation := ride.TreeEstimation{
		Estimation: int(snapshot.VerifierComplexity),
		Verifier:   int(snapshot.VerifierComplexity),
		Functions:  nil,
	}
	// TODO the empty script should not be in snapshots, so it needs to be removed in the future
	if snapshot.Script.IsEmpty() {
		if err := a.stor.scriptsStorage.setAssetScript(snapshot.AssetID, proto.Script{},
			snapshot.SenderPK, a.info.BlockID()); err != nil {
			return err
		}
	}
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: !snapshot.Script.IsEmpty(),
		estimation:    treeEstimation}
	if err := a.stor.scriptsComplexity.saveComplexitiesForAsset(
		snapshot.AssetID, scriptEstimation, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to store asset script estimation for asset %q",
			snapshot.AssetID.String())
	}
	return a.stor.scriptsStorage.setAssetScript(snapshot.AssetID, snapshot.Script, snapshot.SenderPK, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplySponsorship(snapshot SponsorshipSnapshot) error {
	return a.stor.sponsoredAssets.sponsorAsset(snapshot.AssetID, snapshot.MinSponsoredFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAccountScript(snapshot AccountScriptSnapshot) error {
	addr, err := proto.NewAddressFromPublicKey(a.info.Scheme(), snapshot.SenderPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create address from scheme %d and PK %q",
			a.info.Scheme(), snapshot.SenderPublicKey.String())
	}
	treeEstimation := ride.TreeEstimation{
		Estimation: int(snapshot.VerifierComplexity),
		Verifier:   int(snapshot.VerifierComplexity),
		Functions:  nil,
	}
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: !snapshot.Script.IsEmpty(),
		estimation:    treeEstimation}
	if cmplErr := a.stor.scriptsComplexity.saveComplexitiesForAddr(
		addr, scriptEstimation, a.info.BlockID()); cmplErr != nil {
		return errors.Wrapf(cmplErr, "failed to store account script estimation for addr %q",
			addr.String())
	}
	return a.stor.scriptsStorage.setAccountScript(addr, snapshot.Script, snapshot.SenderPublicKey, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyFilledVolumeAndFee(snapshot FilledVolumeFeeSnapshot) error {
	return a.stor.ordersVolumes.storeFilled(snapshot.OrderID.Bytes(),
		snapshot.FilledVolume, snapshot.FilledFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyDataEntries(snapshot DataEntriesSnapshot) error {
	blockID := a.info.BlockID()
	for _, entry := range snapshot.DataEntries {
		if err := a.stor.accountsDataStor.appendEntry(snapshot.Address, entry, blockID); err != nil {
			return errors.Wrapf(err, "failed to add entry (%T) for address %q", entry, snapshot.Address)
		}
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseState(snapshot LeaseStateSnapshot) error {
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
	return a.stor.leases.addLeasing(snapshot.LeaseID, l, a.info.BlockID())
}

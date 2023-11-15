package state

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type blockSnapshotsApplier struct {
	info *blockSnapshotsApplierInfo
	stor snapshotApplierStorages
}

func newBlockSnapshotsApplier(info *blockSnapshotsApplierInfo, stor snapshotApplierStorages) blockSnapshotsApplier {
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

type blockSnapshotsApplierInfo struct {
	ci                  *checkerInfo
	scheme              proto.Scheme
	stateActionsCounter *proto.StateActionsCounter
}

func newBlockSnapshotsApplierInfo(ci *checkerInfo, scheme proto.Scheme,
	counter *proto.StateActionsCounter) *blockSnapshotsApplierInfo {
	return &blockSnapshotsApplierInfo{
		ci:                  ci,
		scheme:              scheme,
		stateActionsCounter: counter,
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

func (a *blockSnapshotsApplier) SetApplierInfo(info *blockSnapshotsApplierInfo) {
	a.info = info
}

func (a *blockSnapshotsApplier) ApplyWavesBalance(snapshot proto.WavesBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.stor.balances.newestWavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.balance = snapshot.Balance
	value := newWavesValue(profile, newProfile)
	if err = a.stor.balances.setWavesBalance(addrID, value, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseBalance(snapshot proto.LeaseBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	var err error
	profile, err := a.stor.balances.newestWavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest waves balance profile for address %q", snapshot.Address.String())
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

func (a *blockSnapshotsApplier) ApplyAssetBalance(snapshot proto.AssetBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	return a.stor.balances.setAssetBalance(addrID, assetID, snapshot.Balance, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAlias(snapshot proto.AliasSnapshot) error {
	return a.stor.aliases.createAlias(snapshot.Alias.Alias, snapshot.Address, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyStaticAssetInfo(snapshot proto.StaticAssetInfoSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	height := a.info.Height() + 1

	assetFullInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(snapshot.AssetID),
			issuer:               snapshot.IssuerPublicKey,
			decimals:             snapshot.Decimals,
			issueHeight:          height,
			issueSequenceInBlock: a.info.stateActionsCounter.NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{},
	}
	return a.stor.assets.issueAsset(assetID, assetFullInfo, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetDescription(snapshot proto.AssetDescriptionSnapshot) error {
	change := &assetInfoChange{
		newName:        snapshot.AssetName,
		newDescription: snapshot.AssetDescription,
		newHeight:      snapshot.ChangeHeight,
	}
	return a.stor.assets.updateAssetInfo(snapshot.AssetID, change, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetVolume(snapshot proto.AssetVolumeSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo, err := a.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest asset info for asset %q", snapshot.AssetID.String())
	}
	assetFullInfo.assetChangeableInfo.reissuable = snapshot.IsReissuable
	assetFullInfo.assetChangeableInfo.quantity = snapshot.TotalQuantity
	return a.stor.assets.storeAssetInfo(assetID, assetFullInfo, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetScript(snapshot proto.AssetScriptSnapshot) error {
	return a.stor.scriptsStorage.setAssetScript(snapshot.AssetID, snapshot.Script, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplySponsorship(snapshot proto.SponsorshipSnapshot) error {
	return a.stor.sponsoredAssets.sponsorAsset(snapshot.AssetID, snapshot.MinSponsoredFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAccountScript(snapshot proto.AccountScriptSnapshot) error {
	addr, err := proto.NewAddressFromPublicKey(a.info.Scheme(), snapshot.SenderPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create address from scheme %d and PK %q",
			a.info.Scheme(), snapshot.SenderPublicKey.String())
	}
	// In case of verifier, there are no functions. If it is a full DApp,
	// the complexity 'functions' will be stored through the internal snapshot InternalDAppComplexitySnapshot.
	treeEstimation := ride.TreeEstimation{
		Estimation: int(snapshot.VerifierComplexity),
		Verifier:   int(snapshot.VerifierComplexity),
		Functions:  nil,
	}
	setErr := a.stor.scriptsStorage.setAccountScript(addr, snapshot.Script, snapshot.SenderPublicKey, a.info.BlockID())
	if setErr != nil {
		return setErr
	}
	se := scriptEstimation{
		currentEstimatorVersion: 0, // 0 means unknown estimator version, script will be re-estimated in full node mode
		scriptIsEmpty:           snapshot.Script.IsEmpty(),
		estimation:              treeEstimation,
	}
	if cmplErr := a.stor.scriptsComplexity.saveComplexitiesForAddr(addr, se, a.info.BlockID()); cmplErr != nil {
		return errors.Wrapf(cmplErr, "failed to store account script estimation for addr %q",
			addr.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyFilledVolumeAndFee(snapshot proto.FilledVolumeFeeSnapshot) error {
	return a.stor.ordersVolumes.storeFilled(snapshot.OrderID.Bytes(),
		snapshot.FilledVolume, snapshot.FilledFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyDataEntries(snapshot proto.DataEntriesSnapshot) error {
	blockID := a.info.BlockID()
	for _, entry := range snapshot.DataEntries {
		if err := a.stor.accountsDataStor.appendEntry(snapshot.Address, entry, blockID); err != nil {
			return errors.Wrapf(err, "failed to add entry (%T) for address %q", entry, snapshot.Address)
		}
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseState(snapshot proto.LeaseStateSnapshot) error {
	switch status := snapshot.Status.(type) {
	case *proto.LeaseStateStatusActive:
		l := &leasing{
			Sender:    status.Sender,
			Recipient: status.Recipient,
			Amount:    status.Amount,
			Status:    LeaseActive,
		}
		return a.stor.leases.addLeasing(snapshot.LeaseID, l, a.info.BlockID())
	case *proto.LeaseStatusCancelled:
		l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
		if err != nil {
			return errors.Wrapf(err, "failed to get leasing info by id '%s' for cancelling", snapshot.LeaseID)
		}
		l.Status = LeaseCancelled
		return a.stor.leases.addLeasing(snapshot.LeaseID, l, a.info.BlockID())
	default:
		return errors.Errorf("invalid lease state snapshot status (%T)", status)
	}
}

func (a *blockSnapshotsApplier) ApplyTransactionsStatus(_ proto.TransactionStatusSnapshot) error {
	return nil // no-op
}

func (a *blockSnapshotsApplier) ApplyDAppComplexity(snapshot InternalDAppComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Save full complexity of both callable and verifier when the script is set first time
	if setErr := a.stor.scriptsComplexity.saveComplexitiesForAddr(snapshot.ScriptAddress,
		scriptEstimation, a.info.BlockID()); setErr != nil {
		return errors.Wrapf(setErr, "failed to save script complexities for addr %q",
			snapshot.ScriptAddress.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyDAppUpdateComplexity(snapshot InternalDAppUpdateComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Update full complexity of both callable and verifier when the script is set first time
	if scErr := a.stor.scriptsComplexity.updateCallableComplexitiesForAddr(
		snapshot.ScriptAddress,
		scriptEstimation, a.info.BlockID()); scErr != nil {
		return errors.Wrapf(scErr, "failed to save complexity for addr %q",
			snapshot.ScriptAddress,
		)
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyAssetScriptComplexity(snapshot InternalAssetScriptComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Save complexity of verifier when the script is set first time
	if setErr := a.stor.scriptsComplexity.saveComplexitiesForAsset(snapshot.AssetID,
		scriptEstimation, a.info.BlockID()); setErr != nil {
		return errors.Wrapf(setErr, "failed to save script complexities for asset ID %q",
			snapshot.AssetID.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseStateActiveInfo(snapshot InternalLeaseStateActiveInfoSnapshot) error {
	l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get leasing info by id '%s' for adding active info", snapshot.LeaseID)
	}
	l.OriginHeight = snapshot.OriginHeight
	l.OriginTransactionID = snapshot.OriginTransactionID
	return a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyLeaseStateCancelInfo(snapshot InternalLeaseStateCancelInfoSnapshot) error {
	l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get leasing info by id '%s' for adding cancel info", snapshot.LeaseID)
	}
	l.CancelHeight = snapshot.CancelHeight
	l.CancelTransactionID = snapshot.CancelTransactionID
	return a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
}

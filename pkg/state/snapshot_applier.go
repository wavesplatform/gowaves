package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type snapshotApplier struct {
	balances          *balances
	aliases           *aliases
	assets            *assets
	scriptsStorage    scriptStorageState
	scriptsComplexity *scriptsComplexity
}

var _ = (&snapshotApplier{}).applyWavesBalance // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyWavesBalance(blockID proto.BlockID, snapshot WavesBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.balance = snapshot.Balance
	value := newWavesValue(profile, newProfile)
	if err := a.balances.setWavesBalance(addrID, value, blockID); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

var _ = (&snapshotApplier{}).applyLeaseBalance // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyLeaseBalance(blockID proto.BlockID, snapshot LeaseBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	profile, err := a.balances.wavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.leaseIn = int64(snapshot.LeaseIn)
	newProfile.leaseOut = int64(snapshot.LeaseOut)
	value := newWavesValue(profile, newProfile)
	if err := a.balances.setWavesBalance(addrID, value, blockID); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

var _ = (&snapshotApplier{}).applyAssetBalance // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyAssetBalance(blockID proto.BlockID, snapshot AssetBalanceSnapshot) error {
	addrID := snapshot.Address.ID()
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	return a.balances.setAssetBalance(addrID, assetID, snapshot.Balance, blockID)
}

var _ = (&snapshotApplier{}).applyAlias // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyAlias(blockID proto.BlockID, snapshot AliasSnapshot) error {
	return a.aliases.createAlias(snapshot.Alias.Alias, snapshot.Address, blockID)
}

var _ = (&snapshotApplier{}).applyStaticAssetInfo // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyStaticAssetInfo(blockID proto.BlockID, snapshot StaticAssetInfoSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	info := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(snapshot.AssetID),
			issuer:               snapshot.IssuerPublicKey,
			decimals:             snapshot.Decimals,
			issueHeight:          0, // TODO: add info?
			issueSequenceInBlock: 0, // TODO: add info?
		},
		assetChangeableInfo: assetChangeableInfo{}, // TODO: add info?
	}
	return a.assets.issueAsset(assetID, info, blockID)
}

var _ = (&snapshotApplier{}).applyAssetDescription // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyAssetDescription(blockID proto.BlockID, snapshot AssetDescriptionSnapshot) error {
	change := &assetInfoChange{
		newName:        snapshot.AssetName,
		newDescription: snapshot.AssetDescription,
		newHeight:      snapshot.ChangeHeight,
	}
	return a.assets.updateAssetInfo(snapshot.AssetID, change, blockID)
}

var _ = (&snapshotApplier{}).applyAssetVolume // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyAssetVolume(blockID proto.BlockID, snapshot AssetVolumeSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	info, err := a.assets.newestAssetInfo(assetID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest asset info for asset %q", snapshot.AssetID.String())
	}
	info.assetChangeableInfo.reissuable = snapshot.IsReissuable
	info.assetChangeableInfo.quantity = snapshot.TotalQuantity
	return a.assets.storeAssetInfo(assetID, info, blockID)
}

var _ = (&snapshotApplier{}).applyAssetScript // TODO: remove it, need for linter for now

func (a *snapshotApplier) applyAssetScript(blockID proto.BlockID, snapshot AssetScriptSnapshot) error {
	estimation := ride.TreeEstimation{ // TODO: use uint in TreeEstimation
		Estimation: int(snapshot.Complexity),
		Verifier:   int(snapshot.Complexity),
		Functions:  nil,
	}
	if err := a.scriptsComplexity.saveComplexitiesForAsset(snapshot.AssetID, estimation, blockID); err != nil {
		return errors.Wrapf(err, "failed to store asset script estimation for asset %q", snapshot.AssetID.String())
	}
	info, err := a.assets.newestConstInfo(proto.AssetIDFromDigest(snapshot.AssetID)) // only issuer can set new asset script
	if err != nil {
		return errors.Wrapf(err, "failed to get const asset info for asset %q", snapshot.AssetID.String())
	}
	return a.scriptsStorage.setAssetScript(snapshot.AssetID, snapshot.Script, info.issuer, blockID)
}

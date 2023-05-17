package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotApplier struct {
	balances *balances
	aliases  *aliases
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

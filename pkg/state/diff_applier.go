package state

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
)

type diffApplier struct {
	balances *balances
}

func newDiffApplier(balances *balances) (*diffApplier, error) {
	return &diffApplier{balances}, nil
}

func newWavesValueFromProfile(p balanceProfile) *wavesValue {
	val := &wavesValue{profile: p}
	if p.leaseIn != 0 || p.leaseOut != 0 {
		val.leaseChange = true
	}
	if p.balance != 0 {
		val.balanceChange = true
	}
	return val
}

func newWavesValue(prevProf, newProf balanceProfile) *wavesValue {
	val := &wavesValue{profile: newProf}
	if prevProf.balance != newProf.balance {
		val.balanceChange = true
	}
	if prevProf.leaseIn != newProf.leaseIn || prevProf.leaseOut != newProf.leaseOut {
		val.leaseChange = true
	}
	return val
}

func (a *diffApplier) applyWavesBalanceChanges(change *balanceChanges, filter, validateOnly bool) error {
	var k wavesBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal waves balance key: %v\n", err)
	}
	profile, err := a.balances.wavesBalance(k.address, filter)
	if err != nil {
		return errors.Errorf("failed to retrieve waves balance: %v\n", err)
	}
	prevProfile := *profile
	for _, diff := range change.balanceDiffs {
		// Check for negative balance.
		newProfile, err := diff.applyTo(profile)
		if err != nil {
			return errs.NewAccountBalanceError(fmt.Sprintf("failed to apply waves balance change for addr %s: %v\n", k.address.String(), err))
		}
		if validateOnly {
			continue
		}
		val := newWavesValue(prevProfile, *newProfile)
		if err := a.balances.setWavesBalance(k.address, val, diff.blockID); err != nil {
			return errors.Errorf("failed to set account balance: %v\n", err)
		}
		prevProfile = *newProfile
	}
	return nil
}

func (a *diffApplier) applyAssetBalanceChanges(change *balanceChanges, filter, validateOnly bool) error {
	var k assetBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal asset balance key: %v\n", err)
	}
	balance, err := a.balances.assetBalance(k.address, k.asset, filter)
	if err != nil {
		return errors.Errorf("failed to retrieve asset balance: %v\n", err)
	}
	prevBalance := balance
	for _, diff := range change.balanceDiffs {
		newBalance, err := diff.applyToAssetBalance(balance)
		if err != nil {
			return errs.NewAccountBalanceError(fmt.Sprintf("validation failed: negative asset balance: %v\n", err))
		}
		if validateOnly {
			continue
		}
		if newBalance == prevBalance {
			// Nothing has changed.
			continue
		}
		if err := a.balances.setAssetBalance(k.address, k.asset, newBalance, diff.blockID); err != nil {
			return errors.Errorf("failed to set asset balance: %v\n", err)
		}
		prevBalance = newBalance
	}
	return nil
}

func (a *diffApplier) applyBalanceChanges(changes *balanceChanges, filter, validateOnly bool) error {
	if len(changes.key) > wavesBalanceKeySize {
		// Is asset change.
		if err := a.applyAssetBalanceChanges(changes, filter, validateOnly); err != nil {
			return err
		}
	} else {
		// Is Waves change, need to take leasing into account.
		if err := a.applyWavesBalanceChanges(changes, filter, validateOnly); err != nil {
			return err
		}
	}
	return nil
}

type changesByKey []balanceChanges

func (k changesByKey) Len() int {
	return len(k)
}
func (k changesByKey) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
func (k changesByKey) Less(i, j int) bool {
	return bytes.Compare(k[i].key, k[j].key) == -1
}

func (a *diffApplier) applyBalancesChangesImpl(changes []balanceChanges, filter, validateOnly bool) error {
	// Sort all changes by addresses they do modify.
	// LevelDB stores data sorted by keys, and the idea is to read in sorted order.
	// We save a lot of time on disk's seek time for hdd, and some time for ssd too (by reducing amount of reads).
	// TODO: if DB supported MultiGet() operation, this would probably be even faster.
	sort.Sort(changesByKey(changes))
	for _, changes := range changes {
		if err := a.applyBalanceChanges(&changes, filter, validateOnly); err != nil {
			return err
		}
	}
	return nil
}

func (a *diffApplier) applyBalancesChanges(changes []balanceChanges, filter bool) error {
	return a.applyBalancesChangesImpl(changes, filter, false)
}

func (a *diffApplier) validateBalancesChanges(changes []balanceChanges, filter bool) error {
	return a.applyBalancesChangesImpl(changes, filter, true)
}

func (a *diffApplier) validateTxDiff(diff txDiff, stor *diffStorage, filter bool) error {
	changes, err := stor.changesByTxDiff(diff)
	if err != nil {
		return err
	}
	return a.validateBalancesChanges(changes, filter)
}

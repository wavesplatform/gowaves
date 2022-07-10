package state

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/wavesplatform/gowaves/pkg/proto"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
)

type diffApplier struct {
	balances *balances
	scheme   proto.Scheme
}

func newDiffApplier(balances *balances, scheme proto.Scheme) (*diffApplier, error) {
	return &diffApplier{balances, scheme}, nil
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

func (a *diffApplier) applyWavesBalanceChanges(change *balanceChanges, validateOnly bool) error {
	var k wavesBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal waves balance key: %v\n", err)
	}
	profile, err := a.balances.newestWavesBalance(k.address)
	if err != nil {
		return errors.Errorf("failed to retrieve waves balance: %v\n", err)
	}
	prevProfile := *profile
	for _, diff := range change.balanceDiffs {
		// Check for negative balance.
		newProfile, err := diff.applyTo(profile)
		if err != nil {
			addr, convertErr := k.address.ToWavesAddress(a.scheme)
			if convertErr != nil {
				return errs.NewAccountBalanceError(fmt.Sprintf(
					"failed to convert AddressID to WavesAddress (%v) and apply waves balance change: %v", convertErr, err,
				))
			}
			return errs.NewAccountBalanceError(fmt.Sprintf(
				"failed to apply waves balance change for addr %s: %v", addr.String(), err,
			))
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

func (a *diffApplier) applyAssetBalanceChanges(change *balanceChanges, validateOnly bool) error {
	var k assetBalanceKey
	if err := k.unmarshal(change.key); err != nil {
		return errors.Errorf("failed to unmarshal asset balance key: %v\n", err)
	}
	balance, err := a.balances.newestAssetBalance(k.address, k.asset)
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

func (a *diffApplier) applyBalanceChanges(changes *balanceChanges, validateOnly bool) error {
	if len(changes.key) > wavesBalanceKeySize {
		// Is asset change.
		if err := a.applyAssetBalanceChanges(changes, validateOnly); err != nil {
			return err
		}
	} else {
		// Is Waves change, need to take leasing into account.
		if err := a.applyWavesBalanceChanges(changes, validateOnly); err != nil {
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

func (a *diffApplier) applyBalancesChangesImpl(changesPack []balanceChanges, validateOnly bool) error {
	// Sort all changes by addresses they do modify.
	// LevelDB stores data sorted by keys, and the idea is to read in sorted order.
	// We save a lot of time on disk's seek time for hdd, and some time for ssd too (by reducing amount of reads).
	// TODO: if DB supported MultiGet() operation, this would probably be even faster.
	sort.Sort(changesByKey(changesPack))
	for i := range changesPack {
		changes := &changesPack[i] // prevent implicit memory aliasing in for loop
		if err := a.applyBalanceChanges(changes, validateOnly); err != nil {
			return err
		}
	}
	return nil
}

func (a *diffApplier) applyBalancesChanges(changes []balanceChanges) error {
	return a.applyBalancesChangesImpl(changes, false)
}

func (a *diffApplier) validateBalancesChanges(changes []balanceChanges) error {
	return a.applyBalancesChangesImpl(changes, true)
}

func (a *diffApplier) validateTxDiff(diff txDiff, stor *diffStorage) error {
	changes, err := stor.changesByTxDiff(diff)
	if err != nil {
		return err
	}
	return a.validateBalancesChanges(changes)
}

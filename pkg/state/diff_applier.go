package state

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type diffApplier struct {
	balances *balances
}

func newDiffApplier(balances *balances) (*diffApplier, error) {
	return &diffApplier{balances}, nil
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
	// Check for negative balance.
	if _, err := change.minBalanceDiff.applyTo(profile); err != nil {
		return errors.Errorf("minimum balance diff for %s produces invalid result: %v\n", k.address.String(), err)
	}
	for _, diff := range change.balanceDiffs {
		// Check for negative balance.
		newProfile, err := diff.applyTo(profile)
		if err != nil {
			return errors.Errorf("failed to apply waves balance change: %v\n", err)
		}
		if validateOnly {
			continue
		}
		r := &wavesBalanceRecord{*newProfile, diff.blockID}
		if err := a.balances.setWavesBalance(k.address, r); err != nil {
			return errors.Errorf("failed to set account balance: %v\n", err)
		}
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
	// Check for negative balance.
	minBalance, err := util.AddInt64(int64(balance), change.minBalanceDiff.balance)
	if err != nil {
		return errors.Errorf("failed to add balances: %v\n", err)
	}
	if minBalance < 0 {
		return errors.New("validation failed: negative asset balance")
	}
	for _, diff := range change.balanceDiffs {
		newBalance, err := util.AddInt64(int64(balance), diff.balance)
		if err != nil {
			return errors.Errorf("failed to add balances: %v\n", err)
		}
		if newBalance < 0 {
			return errors.New("validation failed: negative asset balance")
		}
		if validateOnly {
			continue
		}
		r := &assetBalanceRecord{uint64(newBalance), diff.blockID}
		if err := a.balances.setAssetBalance(k.address, k.asset, r); err != nil {
			return errors.Errorf("failed to set asset balance: %v\n", err)
		}
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

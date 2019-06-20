package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type diffApplierTestObjects struct {
	stor     *storageObjects
	balances *balances
	applier  *diffApplier
}

func createDiffApplierTestObjects(t *testing.T) (*diffApplierTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	balances, err := newBalances(stor.db, stor.hs)
	assert.NoError(t, err, "newBalances() failed")
	applier, err := newDiffApplier(balances)
	assert.NoError(t, err, "newDiffApplier() failed")
	return &diffApplierTestObjects{stor, balances, applier}, path
}

func TestDiffApplierWithWaves(t *testing.T) {
	to, path := createDiffApplierTestObjects(t)

	defer func() {
		err := to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Test applying valid change.
	diff := balanceDiff{balance: 100, blockID: blockID0}
	changes := []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err := to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	profile, err := to.balances.wavesBalance(testGlobal.senderInfo.addr, true)
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.balance, int64(profile.balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
	// Test applying invalid leasing change.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with leasing change leading to negative balance")
	// Valid leasing change.
	diff = balanceDiff{leaseIn: 10, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.flush(t)
	profile, err = to.balances.wavesBalance(testGlobal.senderInfo.addr, true)
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.leaseIn, int64(profile.leaseIn))
	// Test that leasing leased money leads to error.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail when spending leased money")
	// Spending leased money leads to error.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail when spending leased money")
}

func TestDiffApplierWithAssets(t *testing.T) {
	to, path := createDiffApplierTestObjects(t)

	defer func() {
		err := to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Test applying valid change.
	diff := balanceDiff{balance: 100, blockID: blockID0}
	changes := []balanceChanges{
		{[]byte(testGlobal.senderInfo.assetKey), []balanceDiff{diff}, diff},
	}
	err := to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	balance, err := to.balances.assetBalance(testGlobal.senderInfo.addr, testGlobal.assetID, true)
	assert.NoError(t, err, "assetBalance() failed")
	assert.Equal(t, diff.balance, int64(balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.assetKey), []balanceDiff{diff}, diff},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
}

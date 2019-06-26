package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type diffApplierTestObjects struct {
	stor     *storageObjects
	entities *blockchainEntitiesStorage
	applier  *diffApplier
	td       *transactionDiffer
}

func createDiffApplierTestObjects(t *testing.T) (*diffApplierTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	applier, err := newDiffApplier(entities.balances)
	assert.NoError(t, err, "newDiffApplier() failed")
	td, err := newTransactionDiffer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	return &diffApplierTestObjects{stor, entities, applier, td}, path
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
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err := to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	profile, err := to.entities.balances.wavesBalance(testGlobal.senderInfo.addr, true)
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.balance, int64(profile.balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
	// Test applying invalid leasing change.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with leasing change leading to negative balance")
	// Valid leasing change.
	diff = balanceDiff{leaseIn: 10, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.flush(t)
	profile, err = to.entities.balances.wavesBalance(testGlobal.senderInfo.addr, true)
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.leaseIn, int64(profile.leaseIn))
	// Test that leasing leased money leads to error.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail when spending leased money")
	// Spending leased money leads to error.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
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
		{[]byte(testGlobal.senderInfo.assetKey), []balanceDiff{diff}},
	}
	err := to.applier.applyBalancesChanges(changes, true)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	balance, err := to.entities.balances.assetBalance(testGlobal.senderInfo.addr, testGlobal.asset0.assetID, true)
	assert.NoError(t, err, "assetBalance() failed")
	assert.Equal(t, diff.balance, int64(balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.assetKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes, true)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
}

// Check that intermediate balance in Transfer can not be negative.
func TestTransferOverspend(t *testing.T) {
	to, path := createDiffApplierTestObjects(t)

	defer func() {
		err := to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create overspend transfer to self.
	tx := createTransferV1(t)
	info := defaultDifferInfo(t)
	info.blockTime = settings.MainNetSettings.CheckTempNegativeAfterTime - 1
	tx.Timestamp = info.blockTime
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	// Set balance equal to tx Fee.
	to.stor.addBlock(t, blockID0)
	r := &assetBalanceRecord{tx.Fee, blockID0}
	err := to.entities.balances.setAssetBalance(testGlobal.senderInfo.addr, testGlobal.asset0.assetID, r)
	assert.NoError(t, err, "setAssetBalacne() failed")
	to.stor.flush(t)

	// Sending to self more than possess before settings.MainNetSettings.CheckTempNegativeAfterTime is fine.
	diff, err := to.td.createDiffTransferV1(tx, info)
	assert.NoError(t, err, "createDiffTransferV1() failed")
	err = to.applier.validateBalancesChanges(diff.balancesChanges(), true)
	assert.NoError(t, err, "validateBalancesChanges() failed with overspend when it is allowed")
	// Sending to self more than possess after settings.MainNetSettings.CheckTempNegativeAfterTime must lead to error.
	info.blockTime = settings.MainNetSettings.CheckTempNegativeAfterTime
	tx.Timestamp = info.blockTime
	diff, err = to.td.createDiffTransferV1(tx, info)
	assert.NoError(t, err, "createDiffTransferV1() failed")
	err = to.applier.validateBalancesChanges(diff.balancesChanges(), true)
	assert.Error(t, err, "validateBalancesChanges() did not fail with overspend when it is not allowed")
}

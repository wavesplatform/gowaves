package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type diffApplierTestObjects struct {
	stor    *testStorageObjects
	applier *diffApplier
	td      *transactionDiffer
}

func createDiffApplierTestObjects(t *testing.T) *diffApplierTestObjects {
	stor := createStorageObjects(t, true)
	applier, err := newDiffApplier(stor.entities.balances, proto.TestNetScheme)
	require.NoError(t, err, "newDiffApplier() failed")
	td, err := newTransactionDiffer(stor.entities, settings.MainNetSettings)
	require.NoError(t, err, "newTransactionDiffer() failed")
	return &diffApplierTestObjects{stor, applier, td}
}

func TestDiffApplierWithWaves(t *testing.T) {
	to := createDiffApplierTestObjects(t)

	to.stor.addBlock(t, blockID0)
	// Test applying valid change.
	diff := balanceDiff{balance: 100, blockID: blockID0}
	changes := []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err := to.applier.applyBalancesChanges(changes)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.flush(t)
	profile, err := to.stor.entities.balances.wavesBalance(testGlobal.senderInfo.addr.ID())
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.balance, int64(profile.balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
	// Test applying invalid leasing change.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.Error(t, err, "applyBalancesChanges() did not fail with leasing change leading to negative balance")
	// Valid leasing change.
	diff = balanceDiff{leaseIn: 10, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.flush(t)
	profile, err = to.stor.entities.balances.wavesBalance(testGlobal.senderInfo.addr.ID())
	assert.NoError(t, err, "wavesBalance() failed")
	assert.Equal(t, diff.leaseIn, profile.leaseIn)
	// Test that leasing leased money leads to error.
	diff = balanceDiff{leaseOut: 101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.Error(t, err, "applyBalancesChanges() did not fail when spending leased money")
	// Spending leased money leads to error.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.wavesKey), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.Error(t, err, "applyBalancesChanges() did not fail when spending leased money")
}

func TestDiffApplierWithAssets(t *testing.T) {
	to := createDiffApplierTestObjects(t)

	to.stor.addBlock(t, blockID0)
	// Test applying valid change.
	diff := balanceDiff{balance: 100, blockID: blockID0}
	changes := []balanceChanges{
		{[]byte(testGlobal.senderInfo.assetKeys[0]), []balanceDiff{diff}},
	}
	err := to.applier.applyBalancesChanges(changes)
	assert.NoError(t, err, "applyBalancesChanges() failed")
	to.stor.flush(t)
	balance, err := to.stor.entities.balances.assetBalance(
		testGlobal.senderInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset0.assetID),
	)
	assert.NoError(t, err, "assetBalance() failed")
	assert.Equal(t, diff.balance, int64(balance))
	// Test applying invalid balance change.
	diff = balanceDiff{balance: -101, blockID: blockID0}
	changes = []balanceChanges{
		{[]byte(testGlobal.senderInfo.assetKeys[0]), []balanceDiff{diff}},
	}
	err = to.applier.applyBalancesChanges(changes)
	assert.Error(t, err, "applyBalancesChanges() did not fail with balance change leading to negative balance")
}

// Check that intermediate balance in Transfer can not be negative.
func TestTransferOverspend(t *testing.T) {
	to := createDiffApplierTestObjects(t)

	to.stor.addBlock(t, blockID0)
	// Create overspend transfer to self.
	tx := createTransferWithSig(t)
	info := defaultDifferInfo()
	info.blockInfo.Timestamp = settings.MainNetSettings.CheckTempNegativeAfterTime - 1
	tx.Timestamp = info.blockInfo.Timestamp
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	// Set balance equal to tx Fee.
	err := to.stor.entities.balances.setAssetBalance(
		testGlobal.senderInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset0.assetID),
		tx.Fee,
		blockID0,
	)
	assert.NoError(t, err, "setAssetBalance() failed")
	to.stor.flush(t)

	// Sending to self more than possess before settings.MainNetSettings.CheckTempNegativeAfterTime is fine.
	txChanges, err := to.td.createDiffTransferWithSig(tx, info)
	assert.NoError(t, err, "createDiffTransferWithSig() failed")
	err = to.applier.validateBalancesChanges(txChanges.diff.balancesChanges())
	assert.NoError(t, err, "validateBalancesChanges() failed with overspend when it is allowed")
	// Sending to self more than possess after settings.MainNetSettings.CheckTempNegativeAfterTime must lead to error.
	info.blockInfo.Timestamp = settings.MainNetSettings.CheckTempNegativeAfterTime
	tx.Timestamp = info.blockInfo.Timestamp
	txChanges, err = to.td.createDiffTransferWithSig(tx, info)
	assert.NoError(t, err, "createDiffTransferWithSig() failed")
	err = to.applier.validateBalancesChanges(txChanges.diff.balancesChanges())
	assert.Error(t, err, "validateBalancesChanges() did not fail with overspend when it is not allowed")
	assert.EqualError(t, err, "validation failed: negative asset balance: negative intermediate asset balance (Attempt to transfer unavailable funds)\n")
}

package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func TestCheckMinFeeWaves(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Burn.
	tx := createBurnV1(t)
	err = checkMinFeeWaves(tx)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")

	tx.Fee = 1
	err = checkMinFeeWaves(tx)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")

	// MassTransfer special case.
	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx1 := createMassTransferV1(t, entries)
	tx1.Fee = FeeUnit * 34
	err = checkMinFeeWaves(tx1)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid MassTransfer fee")

	tx1.Fee -= 1
	err = checkMinFeeWaves(tx1)
	assert.Error(t, err, "checkMinFeeWaves did not fail with invalid MassTransfer fee")

	// Data transaction special case.
	tx2 := createDataV1(t, 100)
	tx2.Fee = FeeUnit * 2
	err = checkMinFeeWaves(tx2)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Data transaction fee")

	tx2.Fee -= 1
	err = checkMinFeeWaves(tx2)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Data transaction fee")
}

func TestCheckMinFeeAsset(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)

	to.stor.addBlock(t, blockID0)
	assetCost := uint64(4)
	err = to.sponsoredAssets.sponsorAsset(tx.FeeAsset.ID, assetCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	to.stor.flush(t)

	tx.Fee = 1 * assetCost
	err = checkMinFeeAsset(to.sponsoredAssets, tx, tx.FeeAsset.ID)
	assert.NoError(t, err, "checkMinFeeAsset() failed with valid Transfer transaction fee in asset")

	tx.Fee -= 1
	err = checkMinFeeAsset(to.sponsoredAssets, tx, tx.FeeAsset.ID)
	assert.Error(t, err, "checkMinFeeAsset() did not fail with invalid Transfer transaction fee in asset")
}

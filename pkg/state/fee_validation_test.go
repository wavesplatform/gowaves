package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func TestScriptExtraFee(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.stor.entities.accountsScripts.setScript(addr, proto.Script(testGlobal.scriptBytes), blockID0)
	assert.NoError(t, err)

	// Burn.
	tx := createBurnV1(t)
	tx.Fee = 1 * FeeUnit
	params := &feeValidationParams{to.stor.entities, settings.MainNetSettings, false}
	err = checkMinFeeWaves(tx, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")
	tx.Fee += scriptExtraFee
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")
}

func TestCheckMinFeeWaves(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Burn.
	tx := createBurnV1(t)
	params := &feeValidationParams{to.stor.entities, settings.MainNetSettings, false}
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")

	tx.Fee = 1
	err = checkMinFeeWaves(tx, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")

	// MassTransfer special case.
	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx1 := createMassTransferV1(t, entries)
	tx1.Fee = FeeUnit * 34
	err = checkMinFeeWaves(tx1, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid MassTransfer fee")

	tx1.Fee -= 1
	err = checkMinFeeWaves(tx1, params)
	assert.Error(t, err, "checkMinFeeWaves did not fail with invalid MassTransfer fee")

	// Data transaction special case.
	tx2 := createDataV1(t, 100)
	tx2.Fee = FeeUnit * 2
	err = checkMinFeeWaves(tx2, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Data transaction fee")

	tx2.Fee -= 1
	err = checkMinFeeWaves(tx2, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Data transaction fee")
}

func TestCheckMinFeeAsset(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)
	params := &feeValidationParams{to.stor.entities, settings.MainNetSettings, false}

	to.stor.addBlock(t, blockID0)
	assetCost := uint64(4)
	err = to.sponsoredAssets.sponsorAsset(tx.FeeAsset.ID, assetCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	to.stor.flush(t)

	tx.Fee = 1 * assetCost
	err = checkMinFeeAsset(tx, tx.FeeAsset.ID, params)
	assert.NoError(t, err, "checkMinFeeAsset() failed with valid Transfer transaction fee in asset")

	tx.Fee -= 1
	err = checkMinFeeAsset(tx, tx.FeeAsset.ID, params)
	assert.Error(t, err, "checkMinFeeAsset() did not fail with invalid Transfer transaction fee in asset")
}

func TestNFTMinFee(t *testing.T) {
	storage, path, err := createStorageObjects()
	require.NoError(t, err)

	defer func() {
		storage.close(t)
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	params := &feeValidationParams{storage.entities, settings.MainNetSettings, false}

	issueA1 := createIssueV1(t, 500)
	issueA2 := createIssueV2(t, 500)
	issueB1 := createIssueV1(t, 1000)
	issueB2 := createIssueV2(t, 1000)
	nftA1 := createNFTIssueV1(t)
	nftA2 := createNFTIssueV2(t)

	require.Error(t, checkMinFeeWaves(issueA1, params))
	require.Error(t, checkMinFeeWaves(issueA2, params))
	require.NoError(t, checkMinFeeWaves(issueB1, params))
	require.NoError(t, checkMinFeeWaves(issueB2, params))

	require.Error(t, checkMinFeeWaves(nftA1, params))
	require.Error(t, checkMinFeeWaves(nftA2, params))

	storage.activateFeature(t, int16(settings.ReduceNFTFee))

	require.Error(t, checkMinFeeWaves(issueA1, params))
	require.Error(t, checkMinFeeWaves(issueA2, params))
	require.NoError(t, checkMinFeeWaves(issueB1, params))
	require.NoError(t, checkMinFeeWaves(issueB2, params))

	require.NoError(t, checkMinFeeWaves(nftA1, params))
	require.NoError(t, checkMinFeeWaves(nftA2, params))
}

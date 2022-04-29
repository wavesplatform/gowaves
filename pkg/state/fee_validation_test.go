package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func TestAssetScriptExtraFee(t *testing.T) {
	to, path, err := createSponsoredAssets(true)
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err)

	// Burn.
	tx := createBurnWithSig(t)

	to.stor.createSmartAsset(t, tx.AssetID)

	// This fee would be valid for simple Smart Account (without Smart asset).
	tx.Fee = 1*FeeUnit + scriptExtraFee
	params := &feeValidationParams{
		stor:           to.stor.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: []crypto.Digest{tx.AssetID}},
	}
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion) // it doesn't matter for these tests what version estimator is
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")
	// One more extra fee for asset script must be added.
	tx.Fee += scriptExtraFee
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")
}

func TestAccountScriptExtraFee(t *testing.T) {
	to, path, err := createSponsoredAssets(true)
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err)

	// Burn.
	tx := createBurnWithSig(t)
	tx.Fee = 1 * FeeUnit
	params := &feeValidationParams{
		stor:           to.stor.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")
	tx.Fee += scriptExtraFee
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")
}

func TestCheckMinFeeWaves(t *testing.T) {
	to, path, err := createSponsoredAssets(true)
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Burn.
	tx := createBurnWithSig(t)
	params := &feeValidationParams{
		stor:           to.stor.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")

	tx.Fee = 1
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")

	// MassTransfer special case.
	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx1 := createMassTransferWithProofs(t, entries)
	tx1.Fee = FeeUnit * 34
	err = checkMinFeeWaves(tx1, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid MassTransfer fee")

	tx1.Fee -= 1
	err = checkMinFeeWaves(tx1, params, false, maxEstimatorVersion)
	assert.Error(t, err, "checkMinFeeWaves did not fail with invalid MassTransfer fee")

	// Data transaction special case.
	tx2 := createDataWithProofs(t, 100)
	tx2.Fee = FeeUnit * 2
	err = checkMinFeeWaves(tx2, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Data transaction fee")

	tx2.Fee -= 1
	err = checkMinFeeWaves(tx2, params, false, maxEstimatorVersion)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Data transaction fee")
}

func TestCheckMinFeeAsset(t *testing.T) {
	to, path, err := createSponsoredAssets(true)
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferWithSig(t)
	params := &feeValidationParams{
		stor:           to.stor.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}

	to.stor.addBlock(t, blockID0)
	assetCost := uint64(4)
	err = to.sponsoredAssets.sponsorAsset(tx.FeeAsset.ID, assetCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	to.stor.flush(t)

	tx.Fee = 1 * assetCost
	err = checkMinFeeAsset(tx, tx.FeeAsset.ID, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeAsset() failed with valid Transfer transaction fee in asset")

	tx.Fee -= 1
	err = checkMinFeeAsset(tx, tx.FeeAsset.ID, params, false, maxEstimatorVersion)
	assert.Error(t, err, "checkMinFeeAsset() did not fail with invalid Transfer transaction fee in asset")
}

func TestNFTMinFee(t *testing.T) {
	storage, path, err := createStorageObjects()
	require.NoError(t, err)

	defer func() {
		storage.close(t)
		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	params := &feeValidationParams{
		stor:           storage.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}

	issueA1 := createIssueWithSig(t, 500)
	issueA2 := createIssueWithProofs(t, 500)
	issueB1 := createIssueWithSig(t, 1000)
	issueB2 := createIssueWithProofs(t, 1000)
	nftA1 := createNFTIssueWithSig(t)
	nftA2 := createNFTIssueWithProofs(t)

	require.Error(t, checkMinFeeWaves(issueA1, params, false, maxEstimatorVersion))
	require.Error(t, checkMinFeeWaves(issueA2, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(issueB1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(issueB2, params, false, maxEstimatorVersion))

	require.Error(t, checkMinFeeWaves(nftA1, params, false, maxEstimatorVersion))
	require.Error(t, checkMinFeeWaves(nftA2, params, false, maxEstimatorVersion))

	storage.activateFeature(t, int16(settings.ReduceNFTFee))

	require.Error(t, checkMinFeeWaves(issueA1, params, false, maxEstimatorVersion))
	require.Error(t, checkMinFeeWaves(issueA2, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(issueB1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(issueB2, params, false, maxEstimatorVersion))

	require.NoError(t, checkMinFeeWaves(nftA1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(nftA2, params, false, maxEstimatorVersion))
}

func TestReissueFeeReduction(t *testing.T) {
	storage, path, err := createStorageObjects()
	require.NoError(t, err)

	defer func() {
		storage.close(t)
		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	params := &feeValidationParams{
		stor:           storage.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}

	reissueA1 := createReissueWithSig(t, 1)
	reissueA2 := createReissueWithProofs(t, 1)
	reissueB1 := createReissueWithSig(t, 1000)
	reissueB2 := createReissueWithProofs(t, 1000)

	require.Error(t, checkMinFeeWaves(reissueA1, params, false, maxEstimatorVersion))
	require.Error(t, checkMinFeeWaves(reissueA2, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(reissueB1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(reissueB2, params, false, maxEstimatorVersion))

	storage.activateFeature(t, int16(settings.BlockV5))

	require.NoError(t, checkMinFeeWaves(reissueA1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(reissueA2, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(reissueB1, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(reissueB2, params, false, maxEstimatorVersion))
}

func TestSponsorshipFeeReduction(t *testing.T) {
	storage, path, err := createStorageObjects()
	require.NoError(t, err)

	defer func() {
		storage.close(t)
		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	params := &feeValidationParams{
		stor:           storage.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}

	sponsorshipA := createSponsorshipWithProofs(t, 1)
	sponsorshipB := createSponsorshipWithProofs(t, 1000)

	require.Error(t, checkMinFeeWaves(sponsorshipA, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(sponsorshipB, params, false, maxEstimatorVersion))

	storage.activateFeature(t, int16(settings.BlockV5))

	require.NoError(t, checkMinFeeWaves(sponsorshipA, params, false, maxEstimatorVersion))
	require.NoError(t, checkMinFeeWaves(sponsorshipB, params, false, maxEstimatorVersion))
}

func randomScript(size uint64) (proto.Script, error) {
	var s proto.Script = make([]byte, size)
	_, err := rand.Read(s[:])
	if err != nil {
		return nil, err
	}
	return s, nil
}

func TestSetScriptTransactionDynamicFee(t *testing.T) {
	to, path, err := createSponsoredAssets(true)
	assert.NoError(t, err, "createSponsoredAssets() failed")
	to.stor.activateFeature(t, int16(settings.RideV6))
	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetScriptWithProofs(t)
	params := &feeValidationParams{
		stor:           to.stor.entities,
		settings:       settings.MainNetSettings,
		initialisation: false,
		txAssets:       &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
	}

	tx.Script, err = randomScript(2 * 1024)
	assert.NoError(t, err)

	// Validation failed with min fee
	tx.Fee = FeeUnit * 1
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.Error(t, err)

	// Validation ok
	tx.Fee = FeeUnit * 4
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid SetScriptTx fee")

	// Validation with zero size script
	tx.Script = proto.Script{}

	tx.Fee = FeeUnit * 1
	err = checkMinFeeWaves(tx, params, false, maxEstimatorVersion)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid SetScriptTx fee")
}

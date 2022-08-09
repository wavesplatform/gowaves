package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestAssetScriptExtraFee(t *testing.T) {
	to := createSponsoredAssets(t, true)

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err := to.stor.entities.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err)

	// Burn.
	tx := createBurnWithSig(t)

	to.stor.createSmartAsset(t, tx.AssetID)

	// This fee would be valid for simple Smart Account (without Smart asset).
	tx.Fee = 1*FeeUnit + scriptExtraFee
	params := &feeValidationParams{
		stor:             to.stor.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves(), smartAssets: []crypto.Digest{tx.AssetID}},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}
	err = checkMinFeeWaves(tx, params) // it doesn't matter for these tests what version estimator is
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")
	// One more extra fee for asset script must be added.
	tx.Fee += scriptExtraFee
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")
}

/*
	Negative test

The account script is set on blockID2, then rollback returns storage to the blockID1.
The account must not have a verifier anymore. However, the filter is false, so invalid data (verifier) will be returned\
*/
func TestAccountHasVerifierAfterRollbackFilterFalse(t *testing.T) {
	to := createCheckerTestObjects(t)
	to.stor.hs.amend = false

	tx := createSetScriptWithProofs(t)
	info := defaultCheckerInfo()

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	to.stor.addBlock(t, blockID1)
	to.stor.addBlock(t, blockID2)
	_, err := to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	address, err := proto.NewAddressFromPublicKey(to.tc.settings.AddressSchemeCharacter, tx.SenderPK)
	assert.NoError(t, err, "failed to receive an address from public key")

	txPerformerInfo := &performerInfo{blockID: blockID2}
	err = to.tp.performSetScriptWithProofs(tx, txPerformerInfo)
	assert.NoError(t, err, "performSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	hasVerifier, err := to.tp.stor.scriptsStorage.newestAccountHasVerifier(address)
	assert.NoError(t, err, "failed to check whether script has a verifier")
	assert.True(t, hasVerifier, "a script must have a verifier after setting script")

	to.stor.fullRollbackBlockClearCache(t, blockID1)

	hasVerifier, err = to.tp.stor.scriptsStorage.newestAccountHasVerifier(address)
	assert.NoError(t, err, "failed to check whether script has a verifier")
	assert.True(t, hasVerifier, "a script must have not a verifier after rollback") // the filter is false, so the script will be returned
}

// Positive test
// the account script is set on blockID2, then blockID3 is added, then rollback returns storage to the blockID1.
// The account must not have a verifier anymore. Filter is true, so everything must be valid
func TestAccountDoesNotHaveScriptAfterRollbackFilterTrue(t *testing.T) {
	to := createCheckerTestObjects(t)
	to.stor.hs.amend = true

	tx := createSetScriptWithProofs(t)

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	to.stor.addBlock(t, blockID1)
	to.stor.addBlock(t, blockID2)

	address, err := proto.NewAddressFromPublicKey(to.tc.settings.AddressSchemeCharacter, tx.SenderPK)
	assert.NoError(t, err, "failed to receive an address from public key")

	txPerformerInfo := &performerInfo{blockID: blockID2}
	err = to.tp.performSetScriptWithProofs(tx, txPerformerInfo)
	assert.NoError(t, err, "performSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	hasVerifier, err := to.tp.stor.scriptsStorage.newestAccountHasVerifier(address)
	assert.NoError(t, err, "failed to check whether script has a verifier")
	assert.True(t, hasVerifier, "a script must have a verifier after setting script")

	to.stor.addBlock(t, blockID3)

	to.stor.fullRollbackBlockClearCache(t, blockID1)

	hasVerifier, err = to.tp.stor.scriptsStorage.newestAccountHasVerifier(address) // if cache is cleared, the script must have not been found
	assert.NoError(t, err, "failed to check whether script has a verifier")
	assert.False(t, hasVerifier, "a script must have not a verifier after rollback")
}

func TestAccountScriptExtraFee(t *testing.T) {
	to := createSponsoredAssets(t, true)

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err := to.stor.entities.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err)

	// Burn.
	tx := createBurnWithSig(t)
	tx.Fee = 1 * FeeUnit
	params := &feeValidationParams{
		stor:             to.stor.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}
	err = checkMinFeeWaves(tx, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")
	tx.Fee += scriptExtraFee
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")
}

func TestCheckMinFeeWaves(t *testing.T) {
	to := createSponsoredAssets(t, true)

	// Burn.
	tx := createBurnWithSig(t)
	params := &feeValidationParams{
		stor:             to.stor.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}
	err := checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Burn fee")

	tx.Fee = 1
	err = checkMinFeeWaves(tx, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Burn fee")

	// MassTransfer special case.
	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx1 := createMassTransferWithProofs(t, entries)
	tx1.Fee = FeeUnit * 34
	err = checkMinFeeWaves(tx1, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid MassTransfer fee")

	tx1.Fee -= 1
	err = checkMinFeeWaves(tx1, params)
	assert.Error(t, err, "checkMinFeeWaves did not fail with invalid MassTransfer fee")

	// Data transaction special case.
	tx2 := createDataWithProofs(t, 100)
	tx2.Fee = FeeUnit * 2
	err = checkMinFeeWaves(tx2, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid Data transaction fee")

	tx2.Fee -= 1
	err = checkMinFeeWaves(tx2, params)
	assert.Error(t, err, "checkMinFeeWaves() did not fail with invalid Data transaction fee")
}

func TestCheckMinFeeAsset(t *testing.T) {
	to := createSponsoredAssets(t, true)

	tx := createTransferWithSig(t)
	params := &feeValidationParams{
		stor:             to.stor.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}

	to.stor.addBlock(t, blockID0)
	assetCost := uint64(4)
	err := to.sponsoredAssets.sponsorAsset(tx.FeeAsset.ID, assetCost, blockID0)
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
	storage := createStorageObjects(t, true)
	params := &feeValidationParams{
		stor:             storage.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}

	issueA1 := createIssueWithSig(t, 500)
	issueA2 := createIssueWithProofs(t, 500)
	issueB1 := createIssueWithSig(t, 1000)
	issueB2 := createIssueWithProofs(t, 1000)
	nftA1 := createNFTIssueWithSig(t)
	nftA2 := createNFTIssueWithProofs(t)

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

func TestReissueFeeReduction(t *testing.T) {
	storage := createStorageObjects(t, true)
	params := &feeValidationParams{
		stor:             storage.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}

	reissueA1 := createReissueWithSig(t, 1)
	reissueA2 := createReissueWithProofs(t, 1)
	reissueB1 := createReissueWithSig(t, 1000)
	reissueB2 := createReissueWithProofs(t, 1000)

	require.Error(t, checkMinFeeWaves(reissueA1, params))
	require.Error(t, checkMinFeeWaves(reissueA2, params))
	require.NoError(t, checkMinFeeWaves(reissueB1, params))
	require.NoError(t, checkMinFeeWaves(reissueB2, params))

	storage.activateFeature(t, int16(settings.BlockV5))

	require.NoError(t, checkMinFeeWaves(reissueA1, params))
	require.NoError(t, checkMinFeeWaves(reissueA2, params))
	require.NoError(t, checkMinFeeWaves(reissueB1, params))
	require.NoError(t, checkMinFeeWaves(reissueB2, params))
}

func TestSponsorshipFeeReduction(t *testing.T) {
	storage := createStorageObjects(t, true)
	params := &feeValidationParams{
		stor:             storage.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}

	sponsorshipA := createSponsorshipWithProofs(t, 1)
	sponsorshipB := createSponsorshipWithProofs(t, 1000)

	require.Error(t, checkMinFeeWaves(sponsorshipA, params))
	require.NoError(t, checkMinFeeWaves(sponsorshipB, params))

	storage.activateFeature(t, int16(settings.BlockV5))

	require.NoError(t, checkMinFeeWaves(sponsorshipA, params))
	require.NoError(t, checkMinFeeWaves(sponsorshipB, params))
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
	to := createSponsoredAssets(t, true)
	to.stor.activateFeature(t, int16(settings.RideV6))
	tx := createSetScriptWithProofs(t)
	params := &feeValidationParams{
		stor:             to.stor.entities,
		settings:         settings.MainNetSettings,
		txAssets:         &txAssets{feeAsset: proto.NewOptionalAssetWaves()},
		rideV5Activated:  false,
		estimatorVersion: maxEstimatorVersion,
	}

	script, err := randomScript(2 * 1024)
	assert.NoError(t, err)
	tx.Script = script

	// Validation failed with min fee
	tx.Fee = FeeUnit * 1
	err = checkMinFeeWaves(tx, params)
	assert.Error(t, err)

	// Validation ok
	tx.Fee = FeeUnit * 4
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid SetScriptTx fee")

	// Validation with zero size script
	tx.Script = proto.Script{}

	tx.Fee = FeeUnit * 1
	err = checkMinFeeWaves(tx, params)
	assert.NoError(t, err, "checkMinFeeWaves() failed with valid SetScriptTx fee")
}

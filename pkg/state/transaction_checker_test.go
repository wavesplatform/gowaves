package state

import (
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type checkerTestObjects struct {
	stor *testStorageObjects
	tc   *transactionChecker
	tp   *transactionPerformer
}

func createCheckerTestObjects(t *testing.T) (*checkerTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	tc, err := newTransactionChecker(crypto.MustSignatureFromBase58(genesisSignature), stor.entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionChecker() failed")
	tp, err := newTransactionPerformer(stor.entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerormer() failed")
	return &checkerTestObjects{stor, tc, tp}, path
}

func defaultCheckerInfo(t *testing.T) *checkerInfo {
	return &checkerInfo{false, defaultTimestamp, defaultTimestamp - settings.MainNetSettings.MaxTxTimeBackOffset/2, blockID0, 1, 100500}
}

func TestCheckGenesis(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkGenesis(tx, info)
	info.blockID = crypto.MustSignatureFromBase58(genesisSignature)
	assert.Error(t, err, "checkGenesis accepted genesis tx in non-initialisation mode")
	info.initialisation = true
	_, err = to.tc.checkGenesis(tx, info)
	assert.NoError(t, err, "checkGenesis failed with valid genesis tx")
	info.blockID = blockID0
	_, err = to.tc.checkGenesis(tx, info)
	assert.Error(t, err, "checkGenesis accepted genesis tx in non-genesis block")
}

func TestCheckPayment(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	info := defaultCheckerInfo(t)
	info.height = settings.MainNetSettings.BlockVersion3AfterHeight
	_, err := to.tc.checkPayment(tx, info)
	assert.Error(t, err, "checkPayment accepted payment tx after Block v3 height")
	info.height = 10
	_, err = to.tc.checkPayment(tx, info)
	assert.NoError(t, err, "checkPayment failed with valid payment tx")

	tx.Timestamp = 0
	_, err = to.tc.checkPayment(tx, info)
	assert.Error(t, err, "checkPayment did not fail with invalid timestamp")
}

func TestCheckTransferWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferWithSig(t)
	info := defaultCheckerInfo(t)

	assetId := tx.FeeAsset.ID

	_, err := to.tc.checkTransferWithSig(tx, info)
	assert.Error(t, err, "checkTransferWithSig did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.NoError(t, err, "checkTransferWithSig failed with valid transfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AmountAsset.ID)
	smartAssets, err := to.tc.checkTransferWithSig(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AmountAsset.ID, smartAssets[0])

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.Error(t, err, "checkTransferWithSig did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.NoError(t, err, "checkTransferWithSig failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.Error(t, err, "checkTransferWithSig did not fail with invalid timestamp")
}

func TestCheckTransferWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferWithProofs(t)
	info := defaultCheckerInfo(t)

	assetId := tx.FeeAsset.ID

	_, err := to.tc.checkTransferWithProofs(tx, info)
	assert.Error(t, err, "checkTransferWithProofs did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)

	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.Error(t, err, "checkTransferWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	to.stor.createAsset(t, assetId)
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.NoError(t, err, "checkTransferWithProofs failed with valid transfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AmountAsset.ID)
	smartAssets, err := to.tc.checkTransferWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AmountAsset.ID, smartAssets[0])

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.Error(t, err, "checkTransferWithProofs did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.NoError(t, err, "checkTransferWithProofs failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.Error(t, err, "checkTransferWithProofs did not fail with invalid timestamp")
}

func TestCheckIsValidUtf8(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	err := to.tc.isValidUtf8("just a normal string")
	assert.NoError(t, err)

	err = to.tc.isValidUtf8("более странная ひも")
	assert.NoError(t, err)

	invalid := string([]byte{0xff, 0xfe, 0xfd})
	err = to.tc.isValidUtf8(invalid)
	assert.Error(t, err)

	valid := string([]byte{0xc3, 0x87})
	err = to.tc.isValidUtf8(valid)
	assert.NoError(t, err)
}

func TestCheckIssueWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueWithSig(t, 1000)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkIssueWithSig(tx, info)
	assert.NoError(t, err, "checkIssueWithSig failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueWithSig(tx, info)
	assert.Error(t, err, "checkIssueWithSig did not fail with invalid timestamp")
}

func TestCheckIssueWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueWithProofs(t, 1000)
	info := defaultCheckerInfo(t)
	to.stor.addBlock(t, blockID0)

	_, err := to.tc.checkIssueWithProofs(tx, info)
	assert.NoError(t, err, "checkIssueWithProofs failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueWithProofs(tx, info)
	assert.Error(t, err, "checkIssueWithProofs did not fail with invalid timestamp")
}

func TestCheckReissueWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueWithSig(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.ReissueBugWindowTimeEnd + 1
	_, err := to.tc.checkReissueWithSig(tx, info)
	assert.NoError(t, err, "checkReissueWithSig failed with valid reissue tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkReissueWithSig(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	_, err = to.tc.checkReissueWithSig(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkReissueWithSig(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueWithSig failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueWithSig(tx, info)
	assert.Error(t, err, "checkReissueWithSig did not fail when trying to reissue unreissueable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckReissueWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueWithProofs(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.ReissueBugWindowTimeEnd + 1

	_, err := to.tc.checkReissueWithProofs(tx, info)
	assert.Error(t, err, "checkReissueWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkReissueWithProofs(tx, info)
	assert.NoError(t, err, "checkReissueWithProofs failed with valid reissue tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkReissueWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	_, err = to.tc.checkReissueWithProofs(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkReissueWithProofs(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueWithProofs failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueWithProofs(tx, info)
	assert.Error(t, err, "checkReissueWithProofs did not fail when trying to reissue unreissueable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckBurnWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithSig(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkBurnWithSig(tx, info)
	assert.NoError(t, err, "checkBurnWithSig failed with valid burn tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkBurnWithSig(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkBurnWithSig(tx, info)
	assert.Error(t, err, "checkBurnWithSig did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	_, err = to.tc.checkBurnWithSig(tx, info)
	assert.NoError(t, err, "checkBurnWithSig failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	_, err = to.tc.checkBurnWithSig(tx, info)
	assert.Error(t, err, "checkBurnWithSig did not fail with invalid timestamp")
}

func TestCheckBurnWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithProofs(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkBurnWithProofs(tx, info)
	assert.Error(t, err, "checkBurnWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkBurnWithProofs(tx, info)
	assert.NoError(t, err, "checkBurnWithProofs failed with valid burn tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkBurnWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkBurnWithSig(tx, info)
	assert.Error(t, err, "checkBurnWithSig did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	_, err = to.tc.checkBurnWithProofs(tx, info)
	assert.NoError(t, err, "checkBurnWithSig failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	_, err = to.tc.checkBurnWithProofs(tx, info)
	assert.Error(t, err, "checkBurnWithProofs did not fail with invalid timestamp")
}

func TestCheckExchangeWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeWithSig(t)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err, "checkExchangeWithSig failed with valid exchange")

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.recipientInfo.addr
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, proto.Script(testGlobal.scriptBytes), blockID0)
	assert.NoError(t, err)

	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange with smart account before SmartAccountTrading activation")

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err, "checkExchangeWithSig failed with valid exchange")

	// Make one of involved assets smart.
	smartAsset := tx.BuyOrder.AssetPair.AmountAsset.ID
	to.stor.createSmartAsset(t, smartAsset)

	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange with smart assets before SmartAssets activation")

	to.stor.activateFeature(t, int16(settings.SmartAssets))
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err, "checkExchangeWithSig failed with valid exchange")

	// Check that smart assets are detected properly.
	smartAssets, err := to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	// Now overfill volume and make sure check fails.
	tx.Amount = tx.SellOrder.Amount + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills sell order amount volume")
	tx.Amount = tx.SellOrder.Amount

	tx.BuyMatcherFee = tx.SellOrder.MatcherFee + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills sell order matcher fee volume")
	tx.BuyMatcherFee = tx.SellOrder.MatcherFee

	tx.BuyMatcherFee = tx.BuyOrder.MatcherFee + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills buy order matcher fee volume")
	tx.BuyMatcherFee = tx.BuyOrder.MatcherFee

	tx.Amount = tx.BuyOrder.Amount + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills buy order amount volume")
	tx.Amount = tx.BuyOrder.Amount
}

func TestCheckExchangeWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	txOV2 := createExchangeWithProofs(t)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	to.stor.createAsset(t, testGlobal.asset2.asset.ID)

	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail prior to SmartAccountTrading activation")

	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail prior to SmartAccountTrading activation")

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))

	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.NoError(t, err, "checkExchangeWithProofs failed with valid exchange")

	// Make one of involved assets smart.
	smartAsset := txOV2.GetBuyOrderFull().GetAssetPair().AmountAsset.ID
	to.stor.createSmartAsset(t, smartAsset)

	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange with smart assets before SmartAssets activation")

	to.stor.activateFeature(t, int16(settings.SmartAssets))
	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.NoError(t, err, "checkExchangeWithProofs failed with valid exchange")

	// Check that smart assets are detected properly.
	smartAssets, err := to.tc.checkExchangeWithProofs(txOV2, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	// Check validation of ExchangeWithProofs with Orders version 3
	to.stor.activateFeature(t, int16(settings.OrderV3))

	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.NoError(t, err, "checkExchangeWithProofs failed with valid exchange")

	smartAssets, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	txOV3 := createExchangeWithProofsWithOrdersV3(t)

	// Matcher fee asset should not be added to the list of smart assets even if it is smart.
	smartAsset2 := txOV3.GetBuyOrderFull().GetMatcherFeeAsset().ID
	to.stor.createSmartAsset(t, smartAsset2)

	_, err = to.tc.checkExchangeWithProofs(txOV3, info)
	assert.NoError(t, err, "checkExchangeWithProofs failed with valid exchange")

	smartAssets, err = to.tc.checkExchangeWithProofs(txOV3, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.ElementsMatch(t, []crypto.Digest{smartAsset}, smartAssets)

	// Now overfill volume and make sure check fails.
	bo := txOV2.GetBuyOrderFull()
	so := txOV2.GetSellOrderFull()
	txOV2.Amount = so.GetAmount() + 1
	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange that overfills sell order amount volume")
	txOV2.Amount = so.GetAmount()

	txOV2.BuyMatcherFee = so.GetMatcherFee() + 1
	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange that overfills sell order matcher fee volume")
	txOV2.BuyMatcherFee = so.GetMatcherFee()

	txOV2.BuyMatcherFee = bo.GetMatcherFee() + 1
	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange that overfills buy order matcher fee volume")
	txOV2.BuyMatcherFee = bo.GetMatcherFee()

	txOV2.Amount = bo.GetAmount() + 1
	_, err = to.tc.checkExchangeWithProofs(txOV2, info)
	assert.Error(t, err, "checkExchangeWithProofs did not fail with exchange that overfills buy order amount volume")
	txOV2.Amount = bo.GetAmount()
}

func TestCheckLeaseWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseWithSig(t)
	info := defaultCheckerInfo(t)
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	_, err := to.tc.checkLeaseWithSig(tx, info)
	assert.Error(t, err, "checkLeaseWithSig did not fail when leasing to self")

	tx = createLeaseWithSig(t)
	_, err = to.tc.checkLeaseWithSig(tx, info)
	assert.NoError(t, err, "checkLeaseWithSig failed with valid lease tx")
}

func TestCheckLeaseWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseWithProofs(t)
	info := defaultCheckerInfo(t)
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	_, err := to.tc.checkLeaseWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseWithProofs did not fail when leasing to self")

	tx = createLeaseWithProofs(t)

	_, err = to.tc.checkLeaseWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkLeaseWithProofs(tx, info)
	assert.NoError(t, err, "checkLeaseWithProofs failed with valid lease tx")
}

func TestCheckLeaseCancelWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseWithSig(t)
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelWithSig(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelWithSig(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithSig did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseWithSig(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithSig failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkLeaseCancelWithSig(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithSig did not fail when cancelling lease with different sender")
	tx = createLeaseCancelWithSig(t, *leaseTx.ID)

	_, err = to.tc.checkLeaseCancelWithSig(tx, info)
	assert.NoError(t, err, "checkLeaseCancelWithSig failed with valid leaseCancel tx")

	_, err = to.tc.checkLeaseWithSig(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithSig did not fail when cancelling same lease multiple times")
}

func TestCheckLeaseCancelWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseWithProofs(t)
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseWithProofs(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithProofs failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail when cancelling lease with different sender")
	tx = createLeaseCancelWithProofs(t, *leaseTx.ID)

	_, err = to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.NoError(t, err, "checkLeaseCancelWithProofs failed with valid leaseCancel tx")
	err = to.tp.performLeaseCancelWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelWithProofs() failed")

	_, err = to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail when cancelling same lease multiple times")
}

func TestCheckCreateAliasWithSig(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasWithSig(t)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkCreateAliasWithSig(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithSig failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasWithSig(tx, info)
	assert.Error(t, err, "checkCreateAliasWithSig did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasWithSig(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed when stealing aliases is allowed")
}

func TestCheckCreateAliasWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasWithProofs(t)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkCreateAliasWithProofs(tx, info)
	assert.Error(t, err, "checkCreateAliasWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithProofs failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithProofs failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.Error(t, err, "checkCreateAliasWithProofs did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed when stealing aliases is allowed")
}

func TestCheckMassTransferWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	entriesNum := 50
	entries := generateMassTransferEntries(t, entriesNum)
	tx := createMassTransferWithProofs(t, entries)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkMassTransferWithProofs(tx, info)
	assert.Error(t, err, "checkMassTransferWithProofs did not fail prior to feature activation")
	assert.EqualError(t, err, "MassTransfer transaction has not been activated yet")

	// Activate MassTransfer.
	to.stor.activateFeature(t, int16(settings.MassTransfer))
	_, err = to.tc.checkMassTransferWithProofs(tx, info)
	assert.Error(t, err, "checkMassTransferWithProofs did not fail with unissued asset")
	assert.EqualError(t, err, fmt.Sprintf("unknown asset %s", tx.Asset.ID.String()))

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	_, err = to.tc.checkMassTransferWithProofs(tx, info)
	assert.NoError(t, err, "checkMassTransferWithProofs failed with valid massTransfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.Asset.ID)
	smartAssets, err := to.tc.checkMassTransferWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.Asset.ID, smartAssets[0])
}

func TestCheckDataWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataWithProofs(t, 1)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail prior to feature activation")
	assert.EqualError(t, err, "Data transaction has not been activated yet")

	// Activate Data transactions.
	to.stor.activateFeature(t, int16(settings.DataTransaction))
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.NoError(t, err, "checkDataWithProofs failed with valid Data tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail with invalid timestamp")
}

func TestCheckSponsorshipWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSponsorshipWithProofs(t)
	assetInfo := to.stor.createAsset(t, tx.AssetID)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkSponsorshipWithProofs(tx, info)
	assert.Error(t, err, "checkSponsorshipWithProofs did not fail prior to feature activation")
	assert.EqualError(t, err, "sponsorship has not been activated yet")

	// Activate sponsorship.
	to.stor.activateFeature(t, int16(settings.FeeSponsorship))
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.NoError(t, err, "checkSponsorshipWithProofs failed with valid Sponsorship tx")
	to.stor.activateSponsorship(t)

	// Check min fee.
	feeConst, ok := feeConstants[proto.SponsorshipTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.Error(t, err, "checkSponsorshipWithProofs did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.NoError(t, err, "checkSponsorshipWithProofs failed with valid Sponsorship tx")

	// Check invalid sender.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.NoError(t, err, "checkSponsorshipWithProofs failed with valid Sponsorship tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSponsorshipWithProofs(tx, info)
	assert.Error(t, err, "checkSponsorshipWithProofs did not fail with invalid timestamp")
}

func TestCheckSetScriptWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetScriptWithProofs(t)
	info := defaultCheckerInfo(t)

	// Activate sponsorship.
	to.stor.activateSponsorship(t)

	// Activate SmartAccounts.
	to.stor.activateFeature(t, int16(settings.SmartAccounts))
	_, err := to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	// Check min fee.
	feeConst, ok := feeConstants[proto.SetScriptTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	// Test script activation rules.
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	scriptV3Path := filepath.Join(dir, "testdata", "scripts", "version3.base64")
	scriptBase64, err := ioutil.ReadFile(scriptV3Path)
	assert.NoError(t, err)
	scriptBytes, err := reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err)
	prevScript := tx.Script
	tx.Script = proto.Script(scriptBytes)
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with Script V3 before Ride4DApps activation")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	complexScriptPath := filepath.Join(dir, "testdata", "scripts", "exceeds_complexity.base64")
	scriptBase64, err = ioutil.ReadFile(complexScriptPath)
	assert.NoError(t, err)
	scriptBytes, err = reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err)
	tx.Script = proto.Script(scriptBytes)
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with Script that exceeds complexity limit")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with invalid timestamp")
}

func TestCheckSetAssetScriptWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetAssetScriptWithProofs(t)
	info := defaultCheckerInfo(t)

	to.stor.addBlock(t, blockID0)

	// Must fail on non-smart assets.
	_, err := to.tc.checkSetAssetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetAssetScriptWithProofs did not fail with non-smart asset")

	// Make it smart.
	err = to.stor.entities.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, blockID0)
	assert.NoError(t, err, "setAssetScript failed")

	// Now should pass.
	_, err = to.tc.checkSetAssetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetAssetScriptWithProofs failed with valid SetAssetScriptWithProofs tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkSetAssetScriptWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSetAssetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetAssetScriptWithProofs did not fail with invalid timestamp")
}

func TestCheckInvokeScriptWithProofs(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	pmts := []proto.ScriptPayment{
		{Amount: 1, Asset: *testGlobal.asset0.asset},
	}
	tx := createInvokeScriptWithProofs(t, pmts, proto.FunctionCall{}, 1)
	info := defaultCheckerInfo(t)
	to.stor.addBlock(t, blockID0)
	assetId := tx.Payments[0].Asset.ID
	to.stor.createAsset(t, assetId)

	// Check activation.
	_, err := to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.Error(t, err, "checkInvokeScriptWithProofs did not fail prior to Ride4DApps activation")
	to.stor.activateFeature(t, int16(settings.Ride4DApps))
	_, err = to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkInvokeScriptWithProofs failed with valid tx")

	// Check non-issued asset.
	tx.Payments[0].Asset = *testGlobal.asset2.asset
	_, err = to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.Error(t, err, "checkInvokeScriptWithProofs did not fail with invalid asset")
	tx.Payments[0].Asset = *testGlobal.asset0.asset

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, assetId)
	smartAssets, err := to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, assetId, smartAssets[0])

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.Error(t, err, "checkInvokeScriptWithProofs did not fail with invalid timestamp")
}

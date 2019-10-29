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
	return &checkerInfo{false, defaultTimestamp, defaultTimestamp - settings.MainNetSettings.MaxTxTimeBackOffset/2, blockID0, 100500}
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

func TestCheckTransferV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)
	info := defaultCheckerInfo(t)

	assetId := tx.FeeAsset.ID

	_, err := to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)
	_, err = to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err, "checkTransferV1 failed with valid transfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AmountAsset.ID)
	smartAssets, err := to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AmountAsset.ID, smartAssets[0])

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	_, err = to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err, "checkTransferV1 failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with invalid timestamp")
}

func TestCheckTransferV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV2(t)
	info := defaultCheckerInfo(t)

	assetId := tx.FeeAsset.ID

	_, err := to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)

	_, err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	to.stor.createAsset(t, assetId)
	_, err = to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err, "checkTransferV2 failed with valid transfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AmountAsset.ID)
	smartAssets, err := to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AmountAsset.ID, smartAssets[0])

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	_, err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err, "checkTransferV2 failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with invalid timestamp")
}

func TestCheckIssueV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV1(t, 1000)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkIssueV1(tx, info)
	assert.NoError(t, err, "checkIssueV1 failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueV1(tx, info)
	assert.Error(t, err, "checkIssueV1 did not fail with invalid timestamp")
}

func TestCheckIssueV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV2(t, 1000)
	info := defaultCheckerInfo(t)
	to.stor.addBlock(t, blockID0)

	_, err := to.tc.checkIssueV2(tx, info)
	assert.NoError(t, err, "checkIssueV2 failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueV2(tx, info)
	assert.Error(t, err, "checkIssueV2 did not fail with invalid timestamp")
}

func TestCheckReissueV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueV1(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.ReissueBugWindowTimeEnd + 1
	_, err := to.tc.checkReissueV1(tx, info)
	assert.NoError(t, err, "checkReissueV1 failed with valid reissue tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkReissueV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	_, err = to.tc.checkReissueV1(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkReissueV1(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV1 failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueV1(tx, info)
	assert.Error(t, err, "checkReissueV1 did not fail when trying to reissue unreissueable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckReissueV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueV2(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.ReissueBugWindowTimeEnd + 1

	_, err := to.tc.checkReissueV2(tx, info)
	assert.Error(t, err, "checkReissueV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkReissueV2(tx, info)
	assert.NoError(t, err, "checkReissueV2 failed with valid reissue tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkReissueV2(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	_, err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV2 failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueV2(tx, info)
	assert.Error(t, err, "checkReissueV2 did not fail when trying to reissue unreissueable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckBurnV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnV1(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with valid burn tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	_, err = to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	_, err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with invalid timestamp")
}

func TestCheckBurnV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnV2(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkBurnV2(tx, info)
	assert.Error(t, err, "checkBurnV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV2 failed with valid burn tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	_, err = to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	_, err = to.tc.checkBurnV2(tx, info)
	assert.Error(t, err, "checkBurnV2 did not fail with invalid timestamp")
}

func TestCheckExchangeV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeV1(t)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.NoError(t, err, "checkExchangeV1 failed with valid exchange")

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.recipientInfo.addr
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, proto.Script(testGlobal.scriptBytes), blockID0)
	assert.NoError(t, err)

	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange with smart account before SmartAccountTrading activation")

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.NoError(t, err, "checkExchangeV1 failed with valid exchange")

	// Make one of involved assets smart.
	smartAsset := tx.BuyOrder.AssetPair.AmountAsset.ID
	to.stor.createSmartAsset(t, smartAsset)

	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange with smart assets before SmartAssets activation")

	to.stor.activateFeature(t, int16(settings.SmartAssets))
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.NoError(t, err, "checkExchangeV1 failed with valid exchange")

	// Check that smart assets are detected properly.
	smartAssets, err := to.tc.checkExchangeV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	// Now overfill volume and make sure check fails.
	tx.Amount = tx.SellOrder.Amount + 1
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange that overfills sell order amount volume")
	tx.Amount = tx.SellOrder.Amount

	tx.BuyMatcherFee = tx.SellOrder.MatcherFee + 1
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange that overfills sell order matcher fee volume")
	tx.BuyMatcherFee = tx.SellOrder.MatcherFee

	tx.BuyMatcherFee = tx.BuyOrder.MatcherFee + 1
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange that overfills buy order matcher fee volume")
	tx.BuyMatcherFee = tx.BuyOrder.MatcherFee

	tx.Amount = tx.BuyOrder.Amount + 1
	_, err = to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange that overfills buy order amount volume")
	tx.Amount = tx.BuyOrder.Amount
}

func TestCheckExchangeV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	txOV2 := createExchangeV2(t)
	info := defaultCheckerInfo(t)
	_, err := to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	to.stor.createAsset(t, testGlobal.asset2.asset.ID)

	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail prior to SmartAccountTrading activation")

	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail prior to SmartAccountTrading activation")

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))

	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.NoError(t, err, "checkExchangeV2 failed with valid exchange")

	// Make one of involved assets smart.
	smartAsset := txOV2.GetBuyOrderFull().GetAssetPair().AmountAsset.ID
	to.stor.createSmartAsset(t, smartAsset)

	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange with smart assets before SmartAssets activation")

	to.stor.activateFeature(t, int16(settings.SmartAssets))
	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.NoError(t, err, "checkExchangeV2 failed with valid exchange")

	// Check that smart assets are detected properly.
	smartAssets, err := to.tc.checkExchangeV2(txOV2, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	// Check validation of ExchangeV2 with Orders version 3
	to.stor.activateFeature(t, int16(settings.OrderV3))

	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.NoError(t, err, "checkExchangeV2 failed with valid exchange")

	smartAssets, err = to.tc.checkExchangeV2(txOV2, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, smartAsset, smartAssets[0])

	txOV3 := createExchangeV2WithOrdersV3(t)

	// Matcher fee asset should not be added to the list of smart assets even if it is smart.
	smartAsset2 := txOV3.GetBuyOrderFull().GetMatcherFeeAsset().ID
	to.stor.createSmartAsset(t, smartAsset2)

	_, err = to.tc.checkExchangeV2(txOV3, info)
	assert.NoError(t, err, "checkExchangeV2 failed with valid exchange")

	smartAssets, err = to.tc.checkExchangeV2(txOV3, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.ElementsMatch(t, []crypto.Digest{smartAsset}, smartAssets)

	// Now overfill volume and make sure check fails.
	bo := txOV2.GetBuyOrderFull()
	so := txOV2.GetSellOrderFull()
	txOV2.Amount = so.GetAmount() + 1
	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange that overfills sell order amount volume")
	txOV2.Amount = so.GetAmount()

	txOV2.BuyMatcherFee = so.GetMatcherFee() + 1
	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange that overfills sell order matcher fee volume")
	txOV2.BuyMatcherFee = so.GetMatcherFee()

	txOV2.BuyMatcherFee = bo.GetMatcherFee() + 1
	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange that overfills buy order matcher fee volume")
	txOV2.BuyMatcherFee = bo.GetMatcherFee()

	txOV2.Amount = bo.GetAmount() + 1
	_, err = to.tc.checkExchangeV2(txOV2, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange that overfills buy order amount volume")
	txOV2.Amount = bo.GetAmount()
}

func TestCheckLeaseV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV1(t)
	info := defaultCheckerInfo(t)
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	_, err := to.tc.checkLeaseV1(tx, info)
	assert.Error(t, err, "checkLeaseV1 did not fail when leasing to self")

	tx = createLeaseV1(t)
	_, err = to.tc.checkLeaseV1(tx, info)
	assert.NoError(t, err, "checkLeaseV1 failed with valid lease tx")
}

func TestCheckLeaseV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV2(t)
	info := defaultCheckerInfo(t)
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	_, err := to.tc.checkLeaseV2(tx, info)
	assert.Error(t, err, "checkLeaseV2 did not fail when leasing to self")

	tx = createLeaseV2(t)

	_, err = to.tc.checkLeaseV2(tx, info)
	assert.Error(t, err, "checkLeaseV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkLeaseV2(tx, info)
	assert.NoError(t, err, "checkLeaseV2 failed with valid lease tx")
}

func TestCheckLeaseCancelV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV1(t)
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelV1(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelV1(tx, info)
	assert.Error(t, err, "checkLeaseCancelV1 did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseV1(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1 failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkLeaseCancelV1(tx, info)
	assert.Error(t, err, "checkLeaseCancelV1 did not fail when cancelling lease with different sender")
	tx = createLeaseCancelV1(t, *leaseTx.ID)

	_, err = to.tc.checkLeaseCancelV1(tx, info)
	assert.NoError(t, err, "checkLeaseCancelV1 failed with valid leaseCancel tx")

	_, err = to.tc.checkLeaseV1(tx, info)
	assert.Error(t, err, "checkLeaseCancelV1 did not fail when cancelling same lease multiple times")
}

func TestCheckLeaseCancelV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV2(t)
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelV2(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseV2(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2 failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling lease with different sender")
	tx = createLeaseCancelV2(t, *leaseTx.ID)

	_, err = to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkLeaseCancelV2(tx, info)
	assert.NoError(t, err, "checkLeaseCancelV2 failed with valid leaseCancel tx")
	err = to.tp.performLeaseCancelV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV2() failed")

	_, err = to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling same lease multiple times")
}

func TestCheckCreateAliasV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV1(t)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkCreateAliasV1(tx, info)
	assert.NoError(t, err, "checkCreateAliasV1 failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV1 failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasV1(tx, info)
	assert.Error(t, err, "checkCreateAliasV1 did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasV1(tx, info)
	assert.NoError(t, err, "checkCreateAliasV1 failed when stealing aliases is allowed")
}

func TestCheckCreateAliasV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV2(t)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkCreateAliasV2(tx, info)
	assert.Error(t, err, "checkCreateAliasV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkCreateAliasV2(tx, info)
	assert.NoError(t, err, "checkCreateAliasV2 failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV2 failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasV2(tx, info)
	assert.Error(t, err, "checkCreateAliasV2 did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasV2(tx, info)
	assert.NoError(t, err, "checkCreateAliasV1 failed when stealing aliases is allowed")
}

func TestCheckMassTransferV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	entriesNum := 50
	entries := generateMassTransferEntries(t, entriesNum)
	tx := createMassTransferV1(t, entries)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkMassTransferV1(tx, info)
	assert.Error(t, err, "checkMassTransferV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "MassTransfer transaction has not been activated yet")

	// Activate MassTransfer.
	to.stor.activateFeature(t, int16(settings.MassTransfer))
	_, err = to.tc.checkMassTransferV1(tx, info)
	assert.Error(t, err, "checkMassTransferV1 did not fail with unissued asset")
	assert.EqualError(t, err, fmt.Sprintf("unknown asset %s", tx.Asset.ID.String()))

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	_, err = to.tc.checkMassTransferV1(tx, info)
	assert.NoError(t, err, "checkMassTransferV1 failed with valid massTransfer tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.Asset.ID)
	smartAssets, err := to.tc.checkMassTransferV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.Asset.ID, smartAssets[0])
}

func TestCheckDataV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataV1(t, 1)
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkDataV1(tx, info)
	assert.Error(t, err, "checkDataV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "Data transaction has not been activated yet")

	// Activate Data transactions.
	to.stor.activateFeature(t, int16(settings.DataTransaction))
	_, err = to.tc.checkDataV1(tx, info)
	assert.NoError(t, err, "checkDataV1 failed with valid Data tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkDataV1(tx, info)
	assert.Error(t, err, "checkDataV1 did not fail with invalid timestamp")
}

func TestCheckSponsorshipV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSponsorshipV1(t)
	assetInfo := to.stor.createAsset(t, tx.AssetID)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo(t)

	_, err := to.tc.checkSponsorshipV1(tx, info)
	assert.Error(t, err, "checkSponsorshipV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "sponsorship has not been activated yet")

	// Activate sponsorship.
	to.stor.activateFeature(t, int16(settings.FeeSponsorship))
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")
	to.stor.activateSponsorship(t)

	// Check min fee.
	feeConst, ok := feeConstants[proto.SponsorshipTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.Error(t, err, "checkSponsorshipV1 did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")

	// Check invalid sender.
	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSponsorshipV1(tx, info)
	assert.Error(t, err, "checkSponsorshipV1 did not fail with invalid timestamp")
}

func TestCheckSetScriptV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetScriptV1(t)
	info := defaultCheckerInfo(t)

	// Activate sponsorship.
	to.stor.activateSponsorship(t)

	// Activate SmartAccounts.
	to.stor.activateFeature(t, int16(settings.SmartAccounts))
	_, err := to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	// Check min fee.
	feeConst, ok := feeConstants[proto.SetScriptTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

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
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with Script V3 before Ride4DApps activation")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	complexScriptPath := filepath.Join(dir, "testdata", "scripts", "exceeds_complexity.base64")
	scriptBase64, err = ioutil.ReadFile(complexScriptPath)
	assert.NoError(t, err)
	scriptBytes, err = reader.ScriptBytesFromBase64(scriptBase64)
	assert.NoError(t, err)
	tx.Script = proto.Script(scriptBytes)
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with Script that exceeds complexity limit")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with invalid timestamp")
}

func TestCheckSetAssetScriptV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetAssetScriptV1(t)
	info := defaultCheckerInfo(t)

	to.stor.addBlock(t, blockID0)

	// Must fail on non-smart assets.
	_, err := to.tc.checkSetAssetScriptV1(tx, info)
	assert.Error(t, err, "checkSetAssetScriptV1 did not fail with non-smart asset")

	// Make it smart.
	err = to.stor.entities.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, blockID0)
	assert.NoError(t, err, "setAssetScript failed")

	// Now should pass.
	_, err = to.tc.checkSetAssetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetAssetScriptV1 failed with valid SetAssetScriptV1 tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkSetAssetScriptV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSetAssetScriptV1(tx, info)
	assert.Error(t, err, "checkSetAssetScriptV1 did not fail with invalid timestamp")
}

func TestCheckInvokeScriptV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	pmts := []proto.ScriptPayment{
		{Amount: 1, Asset: *testGlobal.asset0.asset},
	}
	tx := createInvokeScriptV1(t, pmts, proto.FunctionCall{}, 1)
	info := defaultCheckerInfo(t)
	to.stor.addBlock(t, blockID0)
	assetId := tx.Payments[0].Asset.ID
	to.stor.createAsset(t, assetId)

	// Check activation.
	_, err := to.tc.checkInvokeScriptV1(tx, info)
	assert.Error(t, err, "checkInvokeScriptV1 did not fail prior to Ride4DApps activation")
	to.stor.activateFeature(t, int16(settings.Ride4DApps))
	_, err = to.tc.checkInvokeScriptV1(tx, info)
	assert.NoError(t, err, "checkInvokeScriptV1 failed with valid tx")

	// Check non-issued asset.
	tx.Payments[0].Asset = *testGlobal.asset2.asset
	_, err = to.tc.checkInvokeScriptV1(tx, info)
	assert.Error(t, err, "checkInvokeScriptV1 did not fail with invalid asset")
	tx.Payments[0].Asset = *testGlobal.asset0.asset

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, assetId)
	smartAssets, err := to.tc.checkInvokeScriptV1(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, assetId, smartAssets[0])

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkInvokeScriptV1(tx, info)
	assert.Error(t, err, "checkInvokeScriptV1 did not fail with invalid timestamp")
}

package state

import (
	"encoding/base64"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

var (
	genSig = crypto.MustSignatureFromBase58(genesisSignature)
)

type checkerTestObjects struct {
	stor *testStorageObjects
	tc   *transactionChecker
	tp   *transactionPerformer
}

func createCheckerTestObjects(t *testing.T) *checkerTestObjects {
	stor := createStorageObjects(t, true)
	tc, err := newTransactionChecker(proto.NewBlockIDFromSignature(genSig), stor.entities, settings.MainNetSettings)
	require.NoError(t, err, "newTransactionChecker() failed")
	tp, err := newTransactionPerformer(stor.entities, settings.MainNetSettings)
	require.NoError(t, err, "newTransactionPerformer() failed")
	return &checkerTestObjects{stor, tc, tp}
}

func defaultCheckerInfo() *checkerInfo {
	return &checkerInfo{
		currentTimestamp: defaultTimestamp,
		parentTimestamp:  defaultTimestamp - settings.MainNetSettings.MaxTxTimeBackOffset/2,
		blockID:          blockID0,
		blockVersion:     1,
		height:           100500,
	}
}

func TestCheckGenesis(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createGenesis()
	info := defaultCheckerInfo()

	_, err := to.tc.checkGenesis(tx, info)
	assert.EqualError(t, err, "genesis transaction inside of non-genesis block")

	info.blockID = proto.NewBlockIDFromSignature(genSig)
	_, err = to.tc.checkGenesis(tx, info)
	assert.EqualError(t, err, "genesis transaction on non zero height")

	info.height = 0
	_, err = to.tc.checkGenesis(tx, info)
	assert.NoError(t, err, "checkGenesis failed in non-initialisation mode")

	to.stor.hs.amend = false
	_, err = to.tc.checkGenesis(tx, info)
	assert.NoError(t, err, "checkGenesis failed with valid genesis tx")
}

func TestCheckPayment(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createPayment(t)
	info := defaultCheckerInfo()
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
	to := createCheckerTestObjects(t)

	tx := createTransferWithSig(t)
	info := defaultCheckerInfo()

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
	assert.Error(t, err, "checkTransferWithSig did not fail with non-sponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("Asset %s is not sponsored, cannot be used to pay fees", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.NoError(t, err, "checkTransferWithSig failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferWithSig(tx, info)
	assert.Error(t, err, "checkTransferWithSig did not fail with invalid timestamp")
}

func TestCheckTransferWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createTransferWithProofs(t)
	info := defaultCheckerInfo()

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
	assert.Error(t, err, "checkTransferWithProofs did not fail with non-sponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("Asset %s is not sponsored, cannot be used to pay fees", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.NoError(t, err, "checkTransferWithProofs failed with valid sponsored asset")

	tx.Timestamp = 0
	_, err = to.tc.checkTransferWithProofs(tx, info)
	assert.Error(t, err, "checkTransferWithProofs did not fail with invalid timestamp")
}

func TestCheckIsValidUtf8(t *testing.T) {
	to := createCheckerTestObjects(t)

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
	to := createCheckerTestObjects(t)

	tx := createIssueWithSig(t, 1000)
	info := defaultCheckerInfo()
	_, err := to.tc.checkIssueWithSig(tx, info)
	assert.NoError(t, err, "checkIssueWithSig failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueWithSig(tx, info)
	assert.Error(t, err, "checkIssueWithSig did not fail with invalid timestamp")
}

func TestCheckIssueWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createIssueWithProofs(t, 1000)
	info := defaultCheckerInfo()
	to.stor.addBlock(t, blockID0)

	_, err := to.tc.checkIssueWithProofs(tx, info)
	assert.NoError(t, err, "checkIssueWithProofs failed with valid issue tx")

	tx.Timestamp = 0
	_, err = to.tc.checkIssueWithProofs(tx, info)
	assert.Error(t, err, "checkIssueWithProofs did not fail with invalid timestamp")
}

func TestCheckReissueWithSig(t *testing.T) {
	to := createCheckerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueWithSig(t, 1000)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo()
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
	err = to.tp.performReissueWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performReissueWithSig failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueWithSig(tx, info)
	assert.Error(t, err, "checkReissueWithSig did not fail when trying to reissue non-reissuable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckReissueWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	tx := createReissueWithProofs(t, 1000)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo()
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
	err = to.tp.performReissueWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performReissueWithProofs failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	_, err = to.tc.checkReissueWithProofs(tx, info)
	assert.Error(t, err, "checkReissueWithProofs did not fail when trying to reissue non-reissuable asset")
	assert.EqualError(t, err, "attempt to reissue asset which is not reissuable")
}

func TestCheckBurnWithSig(t *testing.T) {
	to := createCheckerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithSig(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo()

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
	to := createCheckerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithProofs(t)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo()

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
	to := createCheckerTestObjects(t)

	tx := createExchangeWithSig(t)
	info := defaultCheckerInfo()
	_, err := to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err, "checkExchangeWithSig failed with valid exchange")

	// Set script.
	to.stor.addBlock(t, blockID0)
	addr := testGlobal.recipientInfo.addr
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.recipientInfo.pk, blockID0)
	assert.NoError(t, err)

	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange with smart account before SmartAccountTrading activation")

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.NoError(t, err, "checkExchangeWithSig failed with valid exchange")

	// Make one of involved assets smart.
	smartAsset := tx.Order1.AssetPair.AmountAsset.ID
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
	tx.Amount = tx.Order2.Amount + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills sell order amount volume")
	tx.Amount = tx.Order2.Amount

	tx.BuyMatcherFee = tx.Order2.MatcherFee + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills sell order matcher fee volume")
	tx.BuyMatcherFee = tx.Order2.MatcherFee

	tx.BuyMatcherFee = tx.Order1.MatcherFee + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills buy order matcher fee volume")
	tx.BuyMatcherFee = tx.Order1.MatcherFee

	tx.Amount = tx.Order1.Amount + 1
	_, err = to.tc.checkExchangeWithSig(tx, info)
	assert.Error(t, err, "checkExchangeWithSig did not fail with exchange that overfills buy order amount volume")
	tx.Amount = tx.Order1.Amount
}

func TestCheckExchangeWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	txOV2 := createExchangeWithProofs(t)
	info := defaultCheckerInfo()
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
	smartAsset := txOV2.GetOrder1().GetAssetPair().AmountAsset.ID
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

	txOV3 := createExchangeV2WithProofsWithOrdersV3(t, orderBuildInfo{
		price:  10e8,
		amount: 100,
	})

	// Matcher fee asset should be added to the list of smart assets when it is smart.
	smartAsset2 := txOV3.GetOrder1().GetMatcherFeeAsset().ID
	to.stor.createSmartAsset(t, smartAsset2)

	_, err = to.tc.checkExchangeWithProofs(txOV3, info)
	assert.NoError(t, err, "checkExchangeWithProofs failed with valid exchange")

	smartAssets, err = to.tc.checkExchangeWithProofs(txOV3, info)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(smartAssets))
	assert.ElementsMatch(t, []crypto.Digest{smartAsset, smartAsset2}, smartAssets)

	// Now overfill volume and make sure check fails.
	bo := txOV2.GetOrder1()
	so := txOV2.GetOrder2()
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

func TestCheckUnorderedExchangeV2WithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createUnorderedExchangeWithProofs(t, 2)
	info := defaultCheckerInfo()

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	to.stor.createAsset(t, testGlobal.asset2.asset.ID)

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	to.stor.activateFeature(t, int16(settings.SmartAssets))
	to.stor.activateFeature(t, int16(settings.OrderV3))
	to.stor.activateFeature(t, int16(settings.BlockV5))

	_, err := to.tc.checkExchangeWithProofs(tx, info)
	assert.Errorf(t, err, "have to fail on incorrect order of orders after activation of BlockV5")
}

func TestCheckUnorderedExchangeV3WithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createUnorderedExchangeWithProofs(t, 3)
	info := defaultCheckerInfo()

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	to.stor.createAsset(t, testGlobal.asset2.asset.ID)

	to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	to.stor.activateFeature(t, int16(settings.SmartAssets))
	to.stor.activateFeature(t, int16(settings.OrderV3))
	to.stor.activateFeature(t, int16(settings.BlockV5))

	_, err := to.tc.checkExchangeWithProofs(tx, info)
	assert.NoErrorf(t, err, "failed on with incorrect order of orders after activation of BlockV5")
}

func TestCheckLeaseWithSig(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createLeaseWithSig(t)
	info := defaultCheckerInfo()
	tx.Recipient = proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)
	_, err := to.tc.checkLeaseWithSig(tx, info)
	assert.Error(t, err, "checkLeaseWithSig did not fail when leasing to self")

	tx = createLeaseWithSig(t)
	_, err = to.tc.checkLeaseWithSig(tx, info)
	assert.NoError(t, err, "checkLeaseWithSig failed with valid lease tx")
}

func TestCheckLeaseWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createLeaseWithProofs(t)
	info := defaultCheckerInfo()
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
	to := createCheckerTestObjects(t)

	leaseTx := createLeaseWithSig(t)
	info := defaultCheckerInfo()
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelWithSig(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelWithSig(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithSig did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseWithSig(leaseTx, defaultPerformerInfo())
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
	to := createCheckerTestObjects(t)

	leaseTx := createLeaseWithProofs(t)
	info := defaultCheckerInfo()
	info.currentTimestamp = settings.MainNetSettings.AllowMultipleLeaseCancelUntilTime + 1
	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)

	_, err := to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseWithProofs(leaseTx, defaultPerformerInfo())
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
	err = to.tp.performLeaseCancelWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseCancelWithProofs() failed")

	_, err = to.tc.checkLeaseCancelWithProofs(tx, info)
	assert.Error(t, err, "checkLeaseCancelWithProofs did not fail when cancelling same lease multiple times")
}

func TestCheckCreateAliasWithSig(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createCreateAliasWithSig(t)
	info := defaultCheckerInfo()

	_, err := to.tc.checkCreateAliasWithSig(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithSig failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasWithSig(tx, info)
	assert.Error(t, err, "checkCreateAliasWithSig did not fail when using alias which is already taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasWithSig(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed when stealing aliases is allowed")
}

func TestCheckCreateAliasWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createCreateAliasWithProofs(t)
	info := defaultCheckerInfo()

	_, err := to.tc.checkCreateAliasWithProofs(tx, info)
	assert.Error(t, err, "checkCreateAliasWithProofs did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithProofs failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithProofs failed")
	to.stor.flush(t)

	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.Error(t, err, "checkCreateAliasWithProofs did not fail when using alias which is already taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	_, err = to.tc.checkCreateAliasWithProofs(tx, info)
	assert.NoError(t, err, "checkCreateAliasWithSig failed when stealing aliases is allowed")
}

func TestCheckMassTransferWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	entriesNum := 50
	entries := generateMassTransferEntries(t, entriesNum)
	tx := createMassTransferWithProofs(t, entries)
	info := defaultCheckerInfo()

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
	to := createCheckerTestObjects(t)

	tx := createDataWithProofs(t, 1)
	info := defaultCheckerInfo()

	_, err := to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail prior to feature activation")
	assert.EqualError(t, err, "Data transaction has not been activated yet")

	// Activate Data transactions.
	to.stor.activateFeature(t, int16(settings.DataTransaction))
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.NoError(t, err, "checkDataWithProofs failed with valid Data tx")

	// Check invalid timestamp failure.
	prevTimestamp := tx.Timestamp
	tx.Timestamp = 0
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail with invalid timestamp")
	assert.EqualError(t, err, "invalid timestamp: Transaction timestamp 0 is more than 7200000ms in the past: early transaction creation time")
	tx.Timestamp = prevTimestamp

	// Check data entries
	tx.Entries = append(tx.Entries, &proto.BooleanDataEntry{})
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail with invalid data entry")
	assert.EqualError(t, err, "at least one of the DataWithProofs entry is not valid: invalid entry 1: empty entry key")
	tx.Entries = tx.Entries[:len(tx.Entries)-1]

	// Check data tx size binary before rideV6 activation
	bigEntry := &proto.BinaryDataEntry{Key: "NOTE: see key duplication validation in transactions_test.go", Value: make([]byte, 32*1024-1)}
	bigEntries := proto.DataEntries{bigEntry, bigEntry, bigEntry, bigEntry, bigEntry}
	tx.Entries = append(tx.Entries, bigEntries...)
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.Error(t, err, "checkDataWithProofs did not fail with tx size limit exceeding")
	assert.EqualError(t, err, "data tx binary size limit exceeded, limit=153600, actual size=164299")
	tx.Entries = tx.Entries[:len(tx.Entries)-len(bigEntries)]

	to.stor.activateFeature(t, int16(settings.RideV6))
	tx.Entries = append(tx.Entries, bigEntries...)
	_, err = to.tc.checkDataWithProofs(tx, info)
	assert.NoError(t, err, "checkDataWithProofs failed with valid Data tx")
	tx.Entries = tx.Entries[:len(tx.Entries)-len(bigEntries)]
}

func TestCheckSponsorshipWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createSponsorshipWithProofs(t, 1000)
	assetInfo := to.stor.createAsset(t, tx.AssetID)
	tx.SenderPK = assetInfo.issuer
	info := defaultCheckerInfo()

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
	assert.EqualError(t, err, fmt.Sprintf("Fee %d does not exceed minimal value of %d WAVES. ", tx.Fee, FeeUnit*feeConst))
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
	to := createCheckerTestObjects(t)

	tx := createSetScriptWithProofs(t)
	info := defaultCheckerInfo()

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
	assert.EqualError(t, err, fmt.Sprintf("Fee %d does not exceed minimal value of %d WAVES. ", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	// Test script activation rules.
	scriptBytes, err := readTestScript("version3.base64")
	require.NoError(t, err)
	prevScript := tx.Script
	tx.Script = scriptBytes
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with Script V3 before Ride4DApps activation")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	scriptBytes, err = readTestScript("exceeds_complexity.base64")
	require.NoError(t, err)
	tx.Script = scriptBytes
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with Script that exceeds complexity limit")
	tx.Script = prevScript
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err, "checkSetScriptWithProofs failed with valid SetScriptWithProofs tx")

	// Check script that exceeds 32kb in size
	to.stor.activateFeature(t, int16(settings.RideV5))
	scriptBytes, err = readTestScript("ride5_dapp_exceeds_32kb.base64")
	require.NoError(t, err)
	tx.Script = scriptBytes
	// Big script, RideV6 feature is not activated
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.EqualError(t, err, "checkScript() tx HRXWrnrRy1f7Ur3SNXTtVkNFHNgoqUkpQTB8foqVbptx: script size 32857 is greater than limit of 32768")
	// RideV6 feature is active, but fee is not enough
	to.stor.activateFeature(t, int16(settings.RideV6))
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.EqualError(t, err, "Fee 1000000 does not exceed minimal value of 3300000 WAVES. ") // Scala error text, note the space at the end
	// Everything fine now
	tx.Fee = 3300000
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.NoError(t, err)

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	_, err = to.tc.checkSetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetScriptWithProofs did not fail with invalid timestamp")
}

func TestCheckSetScriptWithProofsCheckScriptComplexity(t *testing.T) {
	tests := []struct {
		estimationStub            ride.TreeEstimation
		libVersions               []ast.LibraryVersion
		isDapp                    bool
		reducedVerifierComplexity bool
		valid                     bool
	}{
		// libVersion 1, 2
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV12 - 1,
				Verifier:   MaxVerifierScriptComplexityReduced - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV12,
				Verifier:   MaxVerifierScriptComplexityReduced,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV12 + 1,
				Verifier:   MaxVerifierScriptComplexityReduced,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV12,
				Verifier:   MaxVerifierScriptComplexityReduced + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		// libVersion 3, 4
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34 - 1,
				Verifier:   MaxVerifierScriptComplexity - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV12,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34 + 1,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34,
				Verifier:   MaxVerifierScriptComplexity + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		// libVersion 5
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV5 - 1,
				Verifier:   MaxVerifierScriptComplexity - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV5},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV5,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV5},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV5 + 1,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV5},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV5,
				Verifier:   MaxVerifierScriptComplexity + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV5},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		// libVersion 6
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6 - 1,
				Verifier:   MaxVerifierScriptComplexity - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6 + 1,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6,
				Verifier:   MaxVerifierScriptComplexity + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		// libVersion 3, 4, 5, 6 - reduced script complexity
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34,
				Verifier:   MaxVerifierScriptComplexityReduced - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4, ast.LibV5, ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: true,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34,
				Verifier:   MaxVerifierScriptComplexityReduced,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4, ast.LibV5, ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: true,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34,
				Verifier:   MaxVerifierScriptComplexityReduced + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV3, ast.LibV4, ast.LibV5, ast.LibV6},
			isDapp:                    true,
			reducedVerifierComplexity: true,
			valid:                     false,
		},
		// not DApp
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6,
				Verifier:   MaxVerifierScriptComplexityReduced - 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    false,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6,
				Verifier:   MaxVerifierScriptComplexityReduced,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    false,
			reducedVerifierComplexity: false,
			valid:                     true,
		},
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV6,
				Verifier:   MaxVerifierScriptComplexityReduced + 1,
			},
			libVersions:               []ast.LibraryVersion{ast.LibV1, ast.LibV2},
			isDapp:                    false,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
		// unknown lib version
		{
			estimationStub: ride.TreeEstimation{
				Estimation: MaxCallableScriptComplexityV34,
				Verifier:   MaxVerifierScriptComplexity,
			},
			libVersions:               []ast.LibraryVersion{128},
			isDapp:                    true,
			reducedVerifierComplexity: false,
			valid:                     false,
		},
	}
	for i, tc := range tests {
		for _, libVersion := range tc.libVersions {
			var checker transactionChecker

			err := checker.checkScriptComplexity(libVersion, tc.estimationStub, tc.isDapp, tc.reducedVerifierComplexity)
			if tc.valid {
				assert.NoError(t, err, "test case %d, libVersion %d", i, libVersion)
			} else {
				assert.Error(t, err, "test case %d, libVersion %d", i, libVersion)
			}
		}
	}
}

func TestCheckSetScriptWithProofsCheckDAppCallables(t *testing.T) {
	tests := []struct {
		comment         string
		source          string
		rideV6Activated bool
		valid           bool
	}{
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				func test(a: List[Int|String], b: Int|String) = []
				
				@Callable(i)
				func f(a: Int) = []`,
			source:          "AAIGAAAAAAAAAAcIAhIDCgEBAAAAAQEAAAAEdGVzdAAAAAIAAAABYQAAAAFiBQAAAANuaWwAAAABAAAAAWkBAAAAAWYAAAABAAAAAWEFAAAAA25pbAAAAABzg4fU",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				func test(a: List[Int|String], b: Int|String) = []
				
				@Callable(i)
				func f(a: List[Int]) = []`,
			source:          "AAIGAAAAAAAAAAcIAhIDCgERAAAAAQEAAAAEdGVzdAAAAAIAAAABYQAAAAFiBQAAAANuaWwAAAABAAAAAWkBAAAAAWYAAAABAAAAAWEFAAAAA25pbAAAAABqAgBk",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 5 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				@Callable(i)
				func f(a: List[Int|String], b: Int|String) = []`,
			source:          "AAIFAAAAAAAAAAgIAhIECgIZCQAAAAAAAAABAAAAAWkBAAAAAWYAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAAcJFxY=",
			rideV6Activated: false,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 5 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				@Callable(i)
				func f(a: List[Int|String], b: Int|String) = []`,
			source:          "AAIFAAAAAAAAAAgIAhIECgIZCQAAAAAAAAABAAAAAWkBAAAAAWYAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAAcJFxY=",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				func test(a: List[Int|String], b: Int|String) = []
				
				@Verifier(tx)
				func verify() = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)`,
			source:          "AAIGAAAAAAAAAAIIAgAAAAEBAAAABHRlc3QAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAACAUAAAACdHgAAAAPc2VuZGVyUHVibGljS2V5UI0jkA==",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				func test(a: List[Int|String], b: Int|String) = []
				
				@Callable(i)
				func f(a: Int, b: ByteVector, c: String, d: Boolean, e: List[Int]) = []`,
			source:          "AAIGAAAAAAAAAAsIAhIHCgUBAggEEQAAAAEBAAAABHRlc3QAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAQAAAAFpAQAAAAFmAAAABQAAAAFhAAAAAWIAAAABYwAAAAFkAAAAAWUFAAAAA25pbAAAAAAhm5rh",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				@Callable(i)
				func f(a: Int|String) = []`,
			source:          "AAIGAAAAAAAAAAcIAhIDCgEJAAAAAAAAAAEAAAABaQEAAAABZgAAAAEAAAABYQUAAAADbmlsAAAAAK91tTo=",
			rideV6Activated: true,
			valid:           false,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				@Callable(i)
				func f(a: List[Int|String]) = []`,
			source:          "AAIGAAAAAAAAAAcIAhIDCgEZAAAAAAAAAAEAAAABaQEAAAABZgAAAAEAAAABYQUAAAADbmlsAAAAAJ5dF/0=",
			rideV6Activated: true,
			valid:           false,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				#serialized without metadata
				@Callable(i)
				func f(a: Int|String, b: List[Int|String]) = []`,
			source:          "AAIGAAAAAAAAAAIIAgAAAAAAAAABAAAAAWkBAAAAAWYAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAASMZSY=",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				#serialized without metadata
				@Callable(i)
				func f(a: List[Int|String]) = []`,
			source:          "AAIGAAAAAAAAAAIIAgAAAAAAAAABAAAAAWkBAAAAAWYAAAABAAAAAWEFAAAAA25pbAAAAABbsAMQ",
			rideV6Activated: true,
			valid:           true,
		},
		{
			comment: `
				{-# STDLIB_VERSION 6 #-}
				{-# CONTENT_TYPE DAPP #-}
				{-# SCRIPT_TYPE ACCOUNT #-}
				
				func test(a: List[Int|String], b: Int|String) = []
				
				#serialized without metadata
				@Callable(i)
				func f(a: Int|String) = []`,
			source:          "AAIGAAAAAAAAAAIIAgAAAAEBAAAABHRlc3QAAAACAAAAAWEAAAABYgUAAAADbmlsAAAAAQAAAAFpAQAAAAFmAAAAAQAAAAFhBQAAAANuaWwAAAAAYJr92g==",
			rideV6Activated: true,
			valid:           true,
		},
	}
	for i, tc := range tests {
		var checker transactionChecker

		script, err := base64.StdEncoding.DecodeString(tc.source)
		require.NoError(t, err)
		tree, err := serialization.Parse(script)
		require.NoError(t, err)

		err = checker.checkDAppCallables(tree, tc.rideV6Activated)
		if tc.valid {
			assert.NoError(t, err, "test case %d: %s", i, tc.comment)
		} else {
			assert.Error(t, err, "test case %d: %s", i, tc.comment)
		}
	}
}

func TestCheckSetAssetScriptWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createSetAssetScriptWithProofs(t)
	info := defaultCheckerInfo()

	assetInfo := defaultAssetInfo(proto.DigestTail(tx.AssetID), true)
	assetInfo.issuer = tx.SenderPK
	to.stor.createAssetUsingInfo(t, tx.AssetID, assetInfo)

	// Must fail on non-smart assets.
	_, err := to.tc.checkSetAssetScriptWithProofs(tx, info)
	assert.Error(t, err, "checkSetAssetScriptWithProofs did not fail with non-smart asset")

	// Make it smart.
	err = to.stor.entities.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, blockID0)
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
	to := createCheckerTestObjects(t)

	payments := []proto.ScriptPayment{
		{Amount: 1, Asset: *testGlobal.asset0.asset},
	}
	tx := createInvokeScriptWithProofs(t, payments, proto.FunctionCall{}, proto.OptionalAsset{}, 1)
	info := defaultCheckerInfo()
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

func TestCheckUpdateAssetInfoWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createUpdateAssetInfoWithProofs(t)
	// We create asset using random block here on purpose, this way
	// heights are not messed up in this test.
	assetInfo := to.stor.createAssetUsingRandomBlock(t, tx.AssetID)
	to.stor.createAsset(t, tx.FeeAsset.ID)
	tx.SenderPK = assetInfo.issuer

	info := defaultCheckerInfo()
	info.height = 100001

	// Check fail prior to activation.
	_, err := to.tc.checkUpdateAssetInfoWithProofs(tx, info)
	assert.EqualError(t, err, "BlockV5 must be activated for UpdateAssetInfo transaction")

	to.stor.activateFeature(t, int16(settings.BlockV5))

	// Check valid.
	_, err = to.tc.checkUpdateAssetInfoWithProofs(tx, info)
	assert.NoError(t, err, "checkUpdateAssetInfoWithProofs failed with valid tx")

	// Check that smart assets are detected properly.
	to.stor.createSmartAsset(t, tx.AssetID)
	smartAssets, err := to.tc.checkUpdateAssetInfoWithProofs(tx, info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(smartAssets))
	assert.Equal(t, tx.AssetID, smartAssets[0])

	tx.SenderPK = testGlobal.recipientInfo.pk
	_, err = to.tc.checkUpdateAssetInfoWithProofs(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	info.height = 99999
	_, err = to.tc.checkUpdateAssetInfoWithProofs(tx, info)
	correctError := fmt.Sprintf("Can't update info of asset with id=%s before height %d, current height is %d", tx.AssetID.String(), 1+to.tc.settings.MinUpdateAssetInfoInterval, info.height+1)
	assert.EqualError(t, err, correctError)
}

func TestCheckInvokeExpressionWithProofs(t *testing.T) {
	to := createCheckerTestObjects(t)

	tx := createInvokeExpressionWithProofs(t, make([]byte, 1), proto.OptionalAsset{}, 1)
	info := defaultCheckerInfo()

	// Check activation.
	_, err := to.tc.checkInvokeScriptWithProofs(tx, info)
	assert.Error(t, err, "checkInvokeExpressionWithProofs did not fail prior to feature InvokeExpression activation")
}

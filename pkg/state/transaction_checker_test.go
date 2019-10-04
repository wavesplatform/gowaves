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
	err := to.tc.checkGenesis(tx, info)
	info.blockID = crypto.MustSignatureFromBase58(genesisSignature)
	assert.Error(t, err, "checkGenesis accepted genesis tx in non-initialisation mode")
	info.initialisation = true
	err = to.tc.checkGenesis(tx, info)
	assert.NoError(t, err, "checkGenesis failed with valid genesis tx")
	info.blockID = blockID0
	err = to.tc.checkGenesis(tx, info)
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
	err := to.tc.checkPayment(tx, info)
	assert.Error(t, err, "checkPayment accepted payment tx after Block v3 height")
	info.height = 10
	err = to.tc.checkPayment(tx, info)
	assert.NoError(t, err, "checkPayment failed with valid payment tx")

	tx.Timestamp = 0
	err = to.tc.checkPayment(tx, info)
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

	err := to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)
	err = to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err, "checkTransferV1 failed with valid transfer tx")

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	err = to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	err = to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err, "checkTransferV1 failed with valid sponsored asset")

	tx.Timestamp = 0
	err = to.tc.checkTransferV1(tx, info)
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

	err := to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with invalid transfer asset")

	to.stor.createAsset(t, assetId)

	err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	to.stor.createAsset(t, assetId)
	err = to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err, "checkTransferV2 failed with valid transfer tx")

	// Sponsorship checks.
	to.stor.activateSponsorship(t)
	err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with unsponsored asset")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): asset %s is not sponsored", assetId.String()))
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, info.blockID)
	assert.NoError(t, err, "sponsorAsset() failed")
	err = to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err, "checkTransferV2 failed with valid sponsored asset")

	tx.Timestamp = 0
	err = to.tc.checkTransferV2(tx, info)
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
	err := to.tc.checkIssueV1(tx, info)
	assert.NoError(t, err, "checkIssueV1 failed with valid issue tx")

	tx.Timestamp = 0
	err = to.tc.checkIssueV1(tx, info)
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

	err := to.tc.checkIssueV2(tx, info)
	assert.NoError(t, err, "checkIssueV2 failed with valid issue tx")

	tx.Timestamp = 0
	err = to.tc.checkIssueV2(tx, info)
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
	err := to.tc.checkReissueV1(tx, info)
	assert.NoError(t, err, "checkReissueV1 failed with valid reissue tx")

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	err = to.tc.checkReissueV1(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkReissueV1(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV1 failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	err = to.tc.checkReissueV1(tx, info)
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

	err := to.tc.checkReissueV2(tx, info)
	assert.Error(t, err, "checkReissueV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	err = to.tc.checkReissueV2(tx, info)
	assert.NoError(t, err, "checkReissueV2 failed with valid reissue tx")

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer

	tx.Reissuable = false
	err = to.tp.performReissueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV2 failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)

	err = to.tc.checkReissueV2(tx, info)
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

	err := to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with valid burn tx")

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	err = to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	err = to.tc.checkBurnV1(tx, info)
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

	err := to.tc.checkBurnV2(tx, info)
	assert.Error(t, err, "checkBurnV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	err = to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV2 failed with valid burn tx")

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	to.stor.activateFeature(t, int16(settings.BurnAnyTokens))
	err = to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	err = to.tc.checkBurnV2(tx, info)
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
	err := to.tc.checkExchangeV1(tx, info)
	assert.Error(t, err, "checkExchangeV1 did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)
	err = to.tc.checkExchangeV1(tx, info)
	assert.NoError(t, err, "checkExchangeV1 failed with valid exchange")
}

func TestCheckExchangeV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeV2(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkExchangeV2(tx, info)
	assert.Error(t, err, "checkExchangeV2 did not fail with exchange with unknown assets")

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	to.stor.createAsset(t, testGlobal.asset1.asset.ID)

	err = to.tc.checkExchangeV2(tx, info)
	assert.Error(t, err, "checkExchangeV2 did not fail prior to SmartAccountTrading activation")

	// TODO: uncomment when the following features will be implemented.
	//to.stor.activateFeature(t, int16(settings.SmartAccountTrading))
	//to.stor.activateFeature(t, int16(settings.OrderV3))

	//err = to.tc.checkExchangeV2(tx, info)
	//assert.NoError(t, err, "checkExchangeV2 failed with valid exchange")
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
	err := to.tc.checkLeaseV1(tx, info)
	assert.Error(t, err, "checkLeaseV1 did not fail when leasing to self")

	tx = createLeaseV1(t)
	err = to.tc.checkLeaseV1(tx, info)
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
	err := to.tc.checkLeaseV2(tx, info)
	assert.Error(t, err, "checkLeaseV2 did not fail when leasing to self")

	tx = createLeaseV2(t)

	err = to.tc.checkLeaseV2(tx, info)
	assert.Error(t, err, "checkLeaseV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	err = to.tc.checkLeaseV2(tx, info)
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

	err := to.tc.checkLeaseCancelV1(tx, info)
	assert.Error(t, err, "checkLeaseCancelV1 did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseV1(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1 failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkLeaseCancelV1(tx, info)
	assert.Error(t, err, "checkLeaseCancelV1 did not fail when cancelling lease with different sender")
	tx = createLeaseCancelV1(t, *leaseTx.ID)

	err = to.tc.checkLeaseCancelV1(tx, info)
	assert.NoError(t, err, "checkLeaseCancelV1 failed with valid leaseCancel tx")

	err = to.tc.checkLeaseV1(tx, info)
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

	err := to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling nonexistent lease")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performLeaseV2(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2 failed")
	to.stor.flush(t)

	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling lease with different sender")
	tx = createLeaseCancelV2(t, *leaseTx.ID)

	err = to.tc.checkLeaseCancelV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	err = to.tc.checkLeaseCancelV2(tx, info)
	assert.NoError(t, err, "checkLeaseCancelV2 failed with valid leaseCancel tx")
	err = to.tp.performLeaseCancelV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV2() failed")

	err = to.tc.checkLeaseCancelV2(tx, info)
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

	err := to.tc.checkCreateAliasV1(tx, info)
	assert.NoError(t, err, "checkCreateAliasV1 failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV1 failed")
	to.stor.flush(t)

	err = to.tc.checkCreateAliasV1(tx, info)
	assert.Error(t, err, "checkCreateAliasV1 did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	err = to.tc.checkCreateAliasV1(tx, info)
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

	err := to.tc.checkCreateAliasV2(tx, info)
	assert.Error(t, err, "checkCreateAliasV2 did not fail prior to SmartAccounts activation")

	to.stor.activateFeature(t, int16(settings.SmartAccounts))

	err = to.tc.checkCreateAliasV2(tx, info)
	assert.NoError(t, err, "checkCreateAliasV2 failed with valid createAlias tx")

	to.stor.addBlock(t, blockID0)
	err = to.tp.performCreateAliasV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV2 failed")
	to.stor.flush(t)

	err = to.tc.checkCreateAliasV2(tx, info)
	assert.Error(t, err, "checkCreateAliasV2 did not fail when using alias which is alredy taken")

	// Check that checker allows to steal aliases at specified timestamp window on MainNet.
	info.currentTimestamp = settings.MainNetSettings.StolenAliasesWindowTimeStart
	err = to.tc.checkCreateAliasV2(tx, info)
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

	err := to.tc.checkMassTransferV1(tx, info)
	assert.Error(t, err, "checkMassTransferV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "MassTransfer transaction has not been activated yet")

	// Activate MassTransfer.
	to.stor.activateFeature(t, int16(settings.MassTransfer))
	err = to.tc.checkMassTransferV1(tx, info)
	assert.Error(t, err, "checkMassTransferV1 did not fail with unissued asset")
	assert.EqualError(t, err, fmt.Sprintf("unknown asset %s", tx.Asset.ID.String()))

	to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	err = to.tc.checkMassTransferV1(tx, info)
	assert.NoError(t, err, "checkMassTransferV1 failed with valid massTransfer tx")
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

	err := to.tc.checkDataV1(tx, info)
	assert.Error(t, err, "checkDataV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "Data transaction has not been activated yet")

	// Activate Data transactions.
	to.stor.activateFeature(t, int16(settings.DataTransaction))
	err = to.tc.checkDataV1(tx, info)
	assert.NoError(t, err, "checkDataV1 failed with valid Data tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	err = to.tc.checkDataV1(tx, info)
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

	err := to.tc.checkSponsorshipV1(tx, info)
	assert.Error(t, err, "checkSponsorshipV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "sponsorship has not been activated yet")

	// Activate sponsorship.
	to.stor.activateFeature(t, int16(settings.FeeSponsorship))
	err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")
	to.stor.activateSponsorship(t)

	// Check min fee.
	feeConst, ok := feeConstants[proto.SponsorshipTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	err = to.tc.checkSponsorshipV1(tx, info)
	assert.Error(t, err, "checkSponsorshipV1 did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")

	// Check invalid sender.
	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkSponsorshipV1(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = assetInfo.issuer
	err = to.tc.checkSponsorshipV1(tx, info)
	assert.NoError(t, err, "checkSponsorshipV1 failed with valid Sponsorship tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	err = to.tc.checkSponsorshipV1(tx, info)
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
	err := to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	// Check min fee.
	feeConst, ok := feeConstants[proto.SetScriptTransaction]
	assert.Equal(t, ok, true)
	tx.Fee = FeeUnit*feeConst - 1
	err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with fee less than minimum")
	assert.EqualError(t, err, fmt.Sprintf("checkFee(): fee %d is less than minimum value of %d\n", tx.Fee, FeeUnit*feeConst))
	tx.Fee = FeeUnit * feeConst
	err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	// Test script activation rules.
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	scriptV3Path := filepath.Join(dir, "testdata", "scripts", "version3.base64")
	scriptBytes, err := ioutil.ReadFile(scriptV3Path)
	assert.NoError(t, err)
	prevScript := tx.Script
	tx.Script = proto.Script(scriptBytes)
	err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with Script V3 before Ride4DApps activation")
	tx.Script = prevScript
	err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	complexScriptPath := filepath.Join(dir, "testdata", "scripts", "exceeds_complexity.base64")
	scriptBytes, err = ioutil.ReadFile(complexScriptPath)
	assert.NoError(t, err)
	tx.Script = proto.Script(scriptBytes)
	err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with Script that exceeds complexity limit")
	tx.Script = prevScript
	err = to.tc.checkSetScriptV1(tx, info)
	assert.NoError(t, err, "checkSetScriptV1 failed with valid SetScriptV1 tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	err = to.tc.checkSetScriptV1(tx, info)
	assert.Error(t, err, "checkSetScriptV1 did not fail with invalid timestamp")
}

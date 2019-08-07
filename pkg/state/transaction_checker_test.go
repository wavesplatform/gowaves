package state

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type checkerTestObjects struct {
	stor     *storageObjects
	entities *blockchainEntitiesStorage
	tc       *transactionChecker
	tp       *transactionPerformer
}

func createCheckerTestObjects(t *testing.T) (*checkerTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, stor.stateDB, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	tc, err := newTransactionChecker(crypto.MustSignatureFromBase58(genesisSignature), entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionChecker() failed")
	tp, err := newTransactionPerformer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerormer() failed")
	return &checkerTestObjects{stor, entities, tc, tp}, path
}

func defaultCheckerInfo(t *testing.T) *checkerInfo {
	return &checkerInfo{false, defaultTimestamp, defaultTimestamp - settings.MainNetSettings.MaxTxTimeBackOffset/2, blockID0, 100500}
}

func TestCheckGenesis(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
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
	assert.Error(t, err, "checkPayment did not fail with invalid payment timestamp")
}

func TestCheckTransferV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)
	info := defaultCheckerInfo(t)

	err := to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with invalid transfer asset")

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	err = to.tc.checkTransferV1(tx, info)
	assert.NoError(t, err, "checkTransferV1 failed with valid transfer tx")

	tx.Timestamp = 0
	err = to.tc.checkTransferV1(tx, info)
	assert.Error(t, err, "checkTransferV1 did not fail with invalid payment timestamp")
}

func TestCheckTransferV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV2(t)
	info := defaultCheckerInfo(t)

	err := to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with invalid transfer asset")

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	err = to.tc.checkTransferV2(tx, info)
	assert.NoError(t, err, "checkTransferV2 failed with valid transfer tx")

	tx.Timestamp = 0
	err = to.tc.checkTransferV2(tx, info)
	assert.Error(t, err, "checkTransferV2 did not fail with invalid payment timestamp")
}

func TestCheckIssueV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV1(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkIssueV1(tx, info)
	assert.NoError(t, err, "checkIssueV1 failed with valid issue tx")

	tx.Timestamp = 0
	err = to.tc.checkIssueV1(tx, info)
	assert.Error(t, err, "checkIssueV1 did not fail with invalid issue timestamp")
}

func TestCheckIssueV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV2(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkIssueV2(tx, info)
	assert.NoError(t, err, "checkIssueV2 failed with valid issue tx")

	tx.Timestamp = 0
	err = to.tc.checkIssueV1(tx, info)
	assert.Error(t, err, "checkIssueV2 did not fail with invalid issue timestamp")
}

func TestCheckReissueV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)

	tx := createReissueV1(t)
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
	tx.SenderPK = testGlobal.senderInfo.pk

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
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)

	tx := createReissueV2(t)
	info := defaultCheckerInfo(t)
	info.currentTimestamp = settings.MainNetSettings.ReissueBugWindowTimeEnd + 1
	err := to.tc.checkReissueV2(tx, info)
	assert.NoError(t, err, "checkReissueV2 failed with valid reissue tx")

	temp := tx.Quantity
	tx.Quantity = math.MaxInt64 + 1
	err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset total value overflow")
	tx.Quantity = temp

	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkReissueV2(tx, info)
	assert.EqualError(t, err, "asset was issued by other address")
	tx.SenderPK = testGlobal.senderInfo.pk

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
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)

	tx := createBurnV1(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with valid burn tx")

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	activateFeature(t, to.entities, to.stor, int16(settings.BurnAnyTokens))
	err = to.tc.checkBurnV1(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with invalid burn timestamp")
}

func TestCheckBurnV2(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)

	tx := createBurnV2(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV2 failed with valid burn tx")

	// Change sender and make sure tx is invalid before activation of BurnAnyTokens feature.
	tx.SenderPK = testGlobal.recipientInfo.pk
	err = to.tc.checkBurnV1(tx, info)
	assert.Error(t, err, "checkBurnV1 did not fail with burn sender not equal to asset issuer before activation of BurnAnyTokens feature")

	// Activate BurnAnyTokens and make sure previous tx is now valid.
	activateFeature(t, to.entities, to.stor, int16(settings.BurnAnyTokens))
	err = to.tc.checkBurnV2(tx, info)
	assert.NoError(t, err, "checkBurnV1 failed with burn sender not equal to asset issuer after activation of BurnAnyTokens feature")

	tx.Timestamp = 0
	err = to.tc.checkBurnV2(tx, info)
	assert.Error(t, err, "checkBurnV2 did not fail with invalid burn timestamp")
}

func TestCheckExchange(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeV1(t)
	info := defaultCheckerInfo(t)
	err := to.tc.checkExchange(tx, info)
	assert.Error(t, err, "checkExchange did not fail with exchange with unknown assets")

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	createAsset(t, to.entities, to.stor, testGlobal.asset1.asset.ID)
	err = to.tc.checkExchange(tx, info)
	assert.NoError(t, err, "checkExchange failed with valid exchange")
}

func TestCheckLeaseV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
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
	assert.NoError(t, err, "checkLeaseV2 failed with valid lease tx")
}

func TestCheckLeaseCancelV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
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
	assert.NoError(t, err, "checkLeaseCancelV2 failed with valid leaseCancel tx")

	err = to.tc.checkLeaseV2(tx, info)
	assert.Error(t, err, "checkLeaseCancelV2 did not fail when cancelling same lease multiple times")
}

func TestCheckCreateAliasV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
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
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV2(t)
	info := defaultCheckerInfo(t)

	err := to.tc.checkCreateAliasV2(tx, info)
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
	activateFeature(t, to.entities, to.stor, int16(settings.MassTransfer))
	err = to.tc.checkMassTransferV1(tx, info)
	assert.Error(t, err, "checkMassTransferV1 did not fail with unissued asset")
	assert.EqualError(t, err, "unknown asset")

	createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	err = to.tc.checkMassTransferV1(tx, info)
	assert.NoError(t, err, "checkMassTransferV1 failed with valid massTransfer tx")
}

func TestCheckDataV1(t *testing.T) {
	to, path := createCheckerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataV1(t)
	info := defaultCheckerInfo(t)

	err := to.tc.checkDataV1(tx, info)
	assert.Error(t, err, "checkDataV1 did not fail prior to feature activation")
	assert.EqualError(t, err, "Data transaction has not been activated yet")

	// Activate Data transactions.
	activateFeature(t, to.entities, to.stor, int16(settings.DataTransaction))
	err = to.tc.checkDataV1(tx, info)
	assert.NoError(t, err, "checkDataV1 failed with valid Data tx")

	// Check invalid timestamp failure.
	tx.Timestamp = 0
	err = to.tc.checkDataV1(tx, info)
	assert.Error(t, err, "checkDataV1 did not fail with invalid Data tx timestamp")
}

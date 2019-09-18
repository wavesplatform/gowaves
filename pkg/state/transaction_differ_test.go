package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	defaultTimestamp = settings.MainNetSettings.CheckTempNegativeAfterTime
	defaultAmount    = uint64(100)
	defaultFee       = uint64(FeeUnit)
	defaultQuantity  = uint64(1000)
	defaultDecimals  = byte(7)
)

type differTestObjects struct {
	stor *testStorageObjects
	td   *transactionDiffer
	tp   *transactionPerformer
}

func createDifferTestObjects(t *testing.T) (*differTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	td, err := newTransactionDiffer(stor.entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	tp, err := newTransactionPerformer(stor.entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerformer() failed")
	return &differTestObjects{stor, td, tp}, path
}

func defaultDifferInfo(t *testing.T) *differInfo {
	return &differInfo{false, testGlobal.minerInfo.pk, defaultTimestamp}
}

func createGenesis(t *testing.T) *proto.Genesis {
	return proto.NewUnsignedGenesis(testGlobal.recipientInfo.addr, defaultAmount, defaultTimestamp)
}

func TestCreateDiffGenesis(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t)
	diff, err := to.td.createDiffGenesis(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffGenesis() failed")
	correctDiff := txDiff{testGlobal.recipientInfo.wavesKey: newBalanceDiff(int64(tx.Amount), 0, 0, false)}
	assert.Equal(t, correctDiff, diff)
}

func createPayment(t *testing.T) *proto.Payment {
	tx := proto.NewUnsignedPayment(testGlobal.senderInfo.pk, testGlobal.recipientInfo.addr, defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffPayment(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	diff, err := to.td.createDiffPayment(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffPayment() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createTransferV1(t *testing.T) *proto.TransferV1 {
	tx := proto.NewUnsignedTransferV1(testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffTransferV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)
	assetId := tx.FeeAsset.ID
	to.stor.createAsset(t, assetId)

	diff, err := to.td.createDiffTransferV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)

	to.stor.activateSponsorship(t)
	_, err = to.td.createDiffTransferV1(tx, defaultDifferInfo(t))
	assert.Error(t, err, "createDiffTransferV1() did not fail with unsponsored asset")
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	diff, err = to.td.createDiffTransferV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV1() failed with valid sponsored asset")

	feeInWaves, err := to.stor.entities.sponsoredAssets.sponsoredAssetToWaves(assetId, tx.Fee)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	correctDiff = txDiff{
		testGlobal.senderInfo.assetKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.issuerInfo.assetKey:    newBalanceDiff(int64(tx.Fee), 0, 0, true),
		testGlobal.issuerInfo.wavesKey:    newBalanceDiff(-int64(feeInWaves), 0, 0, true),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(feeInWaves), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createTransferV2(t *testing.T) *proto.TransferV2 {
	tx := proto.NewUnsignedTransferV2(testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffTransferV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV2(t)
	assetId := tx.FeeAsset.ID
	to.stor.createAsset(t, assetId)

	diff, err := to.td.createDiffTransferV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)

	to.stor.activateSponsorship(t)
	_, err = to.td.createDiffTransferV2(tx, defaultDifferInfo(t))
	assert.Error(t, err, "createDiffTransferV2() did not fail with unsponsored asset")
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	diff, err = to.td.createDiffTransferV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV2() failed with valid sponsored asset")

	feeInWaves, err := to.stor.entities.sponsoredAssets.sponsoredAssetToWaves(assetId, tx.Fee)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	correctDiff = txDiff{
		testGlobal.senderInfo.assetKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.issuerInfo.assetKey:    newBalanceDiff(int64(tx.Fee), 0, 0, true),
		testGlobal.issuerInfo.wavesKey:    newBalanceDiff(-int64(feeInWaves), 0, 0, true),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(feeInWaves), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createIssueV1(t *testing.T) *proto.IssueV1 {
	tx := proto.NewUnsignedIssueV1(testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV1(t)
	diff, err := to.td.createDiffIssueV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffIssueV1() failed")

	correctDiff := txDiff{
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:                       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:                        newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createIssueV2(t *testing.T) *proto.IssueV2 {
	tx := proto.NewUnsignedIssueV2('W', testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, testGlobal.scriptBytes, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV2(t)
	diff, err := to.td.createDiffIssueV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffIssueV2() failed")

	correctDiff := txDiff{
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:                       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:                        newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createReissueV1(t *testing.T) *proto.ReissueV1 {
	tx := proto.NewUnsignedReissueV1(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffReissueV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createReissueV1(t)
	diff, err := to.td.createDiffReissueV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffReissueV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey: newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createReissueV2(t *testing.T) *proto.ReissueV2 {
	tx := proto.NewUnsignedReissueV2('W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffReissueV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createReissueV2(t)
	diff, err := to.td.createDiffReissueV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffReissueV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey: newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createBurnV1(t *testing.T) *proto.BurnV1 {
	tx := proto.NewUnsignedBurnV1(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffBurnV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createBurnV1(t)
	diff, err := to.td.createDiffBurnV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffBurnV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createBurnV2(t *testing.T) *proto.BurnV2 {
	tx := proto.NewUnsignedBurnV2('W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffBurnV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createBurnV2(t)
	diff, err := to.td.createDiffBurnV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffBurnV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createExchangeV1(t *testing.T) *proto.ExchangeV1 {
	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3)
	err := bo.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	err = so.Sign(testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeV1(bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffExchangeV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeV1(t)
	diff, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	price := tx.Price * tx.Amount / priceConstant
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKey:  newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.recipientInfo.assetKey1: newBalanceDiff(int64(price), 0, 0, false),
		testGlobal.recipientInfo.wavesKey:  newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKey:     newBalanceDiff(int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.assetKey1:    newBalanceDiff(-int64(price), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:    newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee-tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createExchangeV2(t *testing.T) *proto.ExchangeV2 {
	bo := proto.NewUnsignedOrderV2(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3)
	err := bo.Sign(testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV2(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	err = so.Sign(testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeV2(bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffExchangeV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeV2(t)
	diff, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	price := tx.Price * tx.Amount / priceConstant
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKey:  newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.recipientInfo.assetKey1: newBalanceDiff(int64(price), 0, 0, false),
		testGlobal.recipientInfo.wavesKey:  newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKey:     newBalanceDiff(int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.assetKey1:    newBalanceDiff(-int64(price), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:    newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee-tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createLeaseV1(t *testing.T) *proto.LeaseV1 {
	tx := proto.NewUnsignedLeaseV1(testGlobal.senderInfo.pk, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV1(t)
	diff, err := to.td.createDiffLeaseV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, int64(tx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, int64(tx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createLeaseV2(t *testing.T) *proto.LeaseV2 {
	tx := proto.NewUnsignedLeaseV2(testGlobal.senderInfo.pk, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV2(t)
	diff, err := to.td.createDiffLeaseV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, int64(tx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, int64(tx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createLeaseCancelV1(t *testing.T, leaseID crypto.Digest) *proto.LeaseCancelV1 {
	tx := proto.NewUnsignedLeaseCancelV1(testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffLeaseCancelV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV1(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, blockID0)
	err := to.tp.performLeaseV1(leaseTx, info)
	assert.NoError(t, err, "performLeaseV1 failed")

	tx := createLeaseCancelV1(t, *leaseTx.ID)
	diff, err := to.td.createDiffLeaseCancelV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, -int64(leaseTx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, -int64(leaseTx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createLeaseCancelV2(t *testing.T, leaseID crypto.Digest) *proto.LeaseCancelV2 {
	tx := proto.NewUnsignedLeaseCancelV2('W', testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffLeaseCancelV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV2(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, blockID0)
	err := to.tp.performLeaseV2(leaseTx, info)
	assert.NoError(t, err, "performLeaseV2 failed")

	tx := createLeaseCancelV2(t, *leaseTx.ID)
	diff, err := to.td.createDiffLeaseCancelV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, -int64(leaseTx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, -int64(leaseTx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createCreateAliasV1(t *testing.T) *proto.CreateAliasV1 {
	aliasStr := "alias"
	aliasFull := fmt.Sprintf("alias:W:%s", aliasStr)
	alias, err := proto.NewAliasFromString(aliasFull)
	assert.NoError(t, err, "NewAliasFromString() failed")
	tx := proto.NewUnsignedCreateAliasV1(testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
	err = tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffCreateAliasV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV1(t)
	diff, err := to.td.createDiffCreateAliasV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffCreateAliasV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createCreateAliasV2(t *testing.T) *proto.CreateAliasV2 {
	aliasStr := "alias"
	aliasFull := fmt.Sprintf("alias:W:%s", aliasStr)
	alias, err := proto.NewAliasFromString(aliasFull)
	assert.NoError(t, err, "NewAliasFromString() failed")
	tx := proto.NewUnsignedCreateAliasV2(testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
	err = tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffCreateAliasV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV2(t)
	diff, err := to.td.createDiffCreateAliasV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffCreateAliasV2 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func generateMassTransferEntries(t *testing.T, entriesNum int) []proto.MassTransferEntry {
	res := make([]proto.MassTransferEntry, entriesNum)
	for i := 0; i < entriesNum; i++ {
		amount := uint64(i)
		rcp := generateRandomRecipient(t)
		entry := proto.MassTransferEntry{Recipient: rcp, Amount: amount}
		res[i] = entry
	}
	return res
}

func createMassTransferV1(t *testing.T, transfers []proto.MassTransferEntry) *proto.MassTransferV1 {
	tx := proto.NewUnsignedMassTransferV1(testGlobal.senderInfo.pk, *testGlobal.asset0.asset, transfers, defaultFee, defaultTimestamp, "attachment")
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffMassTransferV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx := createMassTransferV1(t, entries)
	diff, err := to.td.createDiffMassTransferV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffMassTransferV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, true),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	for _, entry := range entries {
		recipientAddr, err := recipientToAddress(entry.Recipient, to.stor.entities.aliases, true)
		assert.NoError(t, err, "recipientToAddress() failed")
		err = correctDiff.appendBalanceDiff(byteKey(*recipientAddr, tx.Asset.ToID()), newBalanceDiff(int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
		err = correctDiff.appendBalanceDiff(byteKey(testGlobal.senderInfo.addr, tx.Asset.ToID()), newBalanceDiff(-int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
	}
	assert.Equal(t, correctDiff, diff)
}

func createDataV1(t *testing.T, entriesNum int) *proto.DataV1 {
	tx := proto.NewUnsignedData(testGlobal.senderInfo.pk, defaultFee, defaultTimestamp)
	for i := 0; i < entriesNum; i++ {
		entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
		tx.Entries = append(tx.Entries, entry)
	}
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffDataV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataV1(t, 1)
	diff, err := to.td.createDiffDataV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffDataV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createSponsorshipV1(t *testing.T) *proto.SponsorshipV1 {
	feeConst, ok := feeConstants[proto.SponsorshipTransaction]
	assert.Equal(t, ok, true)
	tx := proto.NewUnsignedSponsorshipV1(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, FeeUnit*feeConst, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffSponsorshipV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSponsorshipV1(t)
	diff, err := to.td.createDiffSponsorshipV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffSponsorshipV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createSetScriptV1(t *testing.T) *proto.SetScriptV1 {
	feeConst, ok := feeConstants[proto.SetScriptTransaction]
	assert.Equal(t, ok, true)
	tx := proto.NewUnsignedSetScriptV1('W', testGlobal.senderInfo.pk, testGlobal.scriptBytes, FeeUnit*feeConst, defaultTimestamp)
	err := tx.Sign(testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffSetScriptV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetScriptV1(t)
	diff, err := to.td.createDiffSetScriptV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffSetScriptV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

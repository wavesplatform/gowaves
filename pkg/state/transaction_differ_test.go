package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	// priceConstant is used for exchange calculations.
	priceConstant = 10e7
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

func createGenesis(t *testing.T) *proto.Genesis {
	return proto.NewUnsignedGenesis(testGlobal.recipientInfo.addr, defaultAmount, defaultTimestamp)
}

func TestCreateDiffGenesis(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t)
	ch, err := to.td.createDiffGenesis(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffGenesis() failed")
	correctDiff := txDiff{testGlobal.recipientInfo.wavesKey: newBalanceDiff(int64(tx.Amount), 0, 0, false)}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createPayment(t *testing.T) *proto.Payment {
	tx := proto.NewUnsignedPayment(testGlobal.senderInfo.pk, testGlobal.recipientInfo.addr, defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffPayment(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	ch, err := to.td.createDiffPayment(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffPayment() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createTransferWithSig(t *testing.T) *proto.TransferWithSig {
	tx := proto.NewUnsignedTransferWithSig(testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), []byte("attachment"))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffTransferWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferWithSig(t)
	assetId := tx.FeeAsset.ID
	to.stor.createAsset(t, assetId)

	ch, err := to.td.createDiffTransferWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferWithSig() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKeys[0]:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)

	to.stor.activateSponsorship(t)
	_, err = to.td.createDiffTransferWithSig(tx, defaultDifferInfo(t))
	assert.Error(t, err, "createDiffTransferWithSig() did not fail with unsponsored asset")
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	ch, err = to.td.createDiffTransferWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferWithSig() failed with valid sponsored asset")

	feeInWaves, err := to.stor.entities.sponsoredAssets.sponsoredAssetToWaves(assetId, tx.Fee)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	correctDiff = txDiff{
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.issuerInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Fee), 0, 0, true),
		testGlobal.issuerInfo.wavesKey:        newBalanceDiff(-int64(feeInWaves), 0, 0, true),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(feeInWaves), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs = map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
		testGlobal.issuerInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createTransferWithProofs(t *testing.T) *proto.TransferWithProofs {
	tx := proto.NewUnsignedTransferWithProofs(2, testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), []byte("attachment"))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffTransferWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferWithProofs(t)
	assetId := tx.FeeAsset.ID
	to.stor.createAsset(t, assetId)

	ch, err := to.td.createDiffTransferWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKeys[0]:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)

	to.stor.activateSponsorship(t)
	_, err = to.td.createDiffTransferWithProofs(tx, defaultDifferInfo(t))
	assert.Error(t, err, "createDiffTransferWithProofs() did not fail with unsponsored asset")
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	ch, err = to.td.createDiffTransferWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferWithProofs() failed with valid sponsored asset")

	feeInWaves, err := to.stor.entities.sponsoredAssets.sponsoredAssetToWaves(assetId, tx.Fee)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	correctDiff = txDiff{
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true),
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.issuerInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Fee), 0, 0, true),
		testGlobal.issuerInfo.wavesKey:        newBalanceDiff(-int64(feeInWaves), 0, 0, true),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(feeInWaves), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs = map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
		testGlobal.issuerInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createIssueWithSig(t *testing.T, feeUnits int) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, defaultTimestamp, uint64(feeUnits*FeeUnit))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func createNFTIssueWithSig(t *testing.T) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testGlobal.senderInfo.pk, "nft", "nft asset", 1, 0, false, defaultTimestamp, defaultFee)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueWithSig(t, 1000)
	ch, err := to.td.createDiffIssueWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffIssueWithSig() failed")

	correctDiff := txDiff{
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:                       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:                        newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createIssueWithProofs(t *testing.T, feeUnits int) *proto.IssueWithProofs {
	tx := proto.NewUnsignedIssueWithProofs(2, 'W', testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, testGlobal.scriptBytes, defaultTimestamp, uint64(feeUnits*FeeUnit))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func createNFTIssueWithProofs(t *testing.T) *proto.IssueWithProofs {
	tx := proto.NewUnsignedIssueWithProofs(2, 'W', testGlobal.senderInfo.pk, "nfg", "nft like asset", 1, 0, false, testGlobal.scriptBytes, defaultTimestamp, defaultFee)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueWithProofs(t, 1000)
	ch, err := to.td.createDiffIssueWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffIssueWithProofs() failed")

	correctDiff := txDiff{
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:                       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:                        newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createReissueWithSig(t *testing.T, feeUnits int) *proto.ReissueWithSig {
	tx := proto.NewUnsignedReissueWithSig(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, uint64(feeUnits*FeeUnit))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffReissueWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createReissueWithSig(t, 1000)
	ch, err := to.td.createDiffReissueWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffReissueWithSig() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]: newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createReissueWithProofs(t *testing.T, feeUnits int) *proto.ReissueWithProofs {
	tx := proto.NewUnsignedReissueWithProofs(2, 'W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, uint64(feeUnits*FeeUnit))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffReissueWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createReissueWithProofs(t, 1000)
	ch, err := to.td.createDiffReissueWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffReissueWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]: newBalanceDiff(int64(tx.Quantity), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createBurnWithSig(t *testing.T) *proto.BurnWithSig {
	tx := proto.NewUnsignedBurnWithSig(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffBurnWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createBurnWithSig(t)
	ch, err := to.td.createDiffBurnWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffBurnWithSig() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createBurnWithProofs(t *testing.T) *proto.BurnWithProofs {
	tx := proto.NewUnsignedBurnWithProofs(2, 'W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffBurnWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createBurnWithProofs(t)
	ch, err := to.td.createDiffBurnWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffBurnWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.wavesKey:     newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:      newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createExchangeWithSig(t *testing.T) *proto.ExchangeWithSig {
	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithSig(bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

//TODO: this function is used in test that is commented for now
//func createExchangeWithSigParams(t *testing.T, price, amount uint64) *proto.ExchangeWithSig {
//	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, price, amount, 0, 0, 3)
//	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
//	assert.NoError(t, err, "bo.Sign() failed")
//	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, price, amount, 0, 0, 3)
//	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
//	assert.NoError(t, err, "so.Sign() failed")
//	tx := proto.NewUnsignedExchangeWithSig(bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
//	err = tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
//	assert.NoError(t, err, "tx.Sign() failed")
//	return tx
//}

func TestCreateDiffExchangeWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeWithSig(t)
	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	price := tx.Price * tx.Amount / priceConstant
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[1]: newBalanceDiff(int64(price), 0, 0, false),
		testGlobal.recipientInfo.wavesKey:     newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.assetKeys[1]:    newBalanceDiff(-int64(price), 0, 0, false),
		testGlobal.senderInfo.wavesKey:        newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:       newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee-tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
		testGlobal.matcherInfo.addr:   empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createExchangeWithProofs(t *testing.T) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV2(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV2(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(2, bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func createUnorderedExchangeWithProofs(t *testing.T, v int) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV3(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3, *testGlobal.asset2.asset)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV3(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3, *testGlobal.asset2.asset)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(byte(v), so, bo, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffExchangeWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeWithProofs(t)
	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	price := tx.Price * tx.Amount / priceConstant
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[1]: newBalanceDiff(int64(price), 0, 0, false),
		testGlobal.recipientInfo.wavesKey:     newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.assetKeys[1]:    newBalanceDiff(-int64(price), 0, 0, false),
		testGlobal.senderInfo.wavesKey:        newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:       newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee-tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
		testGlobal.matcherInfo.addr:   empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createExchangeWithProofsWithOrdersV3(t *testing.T) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV3(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 100, 0, 0, 3, *testGlobal.asset2.asset)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	require.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV3(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3, *testGlobal.asset2.asset)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	require.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(2, bo, so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	require.NoError(t, err, "tx.Sign() failed")
	return tx
}

func createExchangeWithProofsWithOrdersV4(t *testing.T, price, amount uint64) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV4(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, uint64(price), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	require.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV4(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, uint64(price), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	require.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(3, bo, so, uint64(price), bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	require.NoError(t, err, "tx.Sign() failed")
	return tx
}

func createExchangeV3WithProofsWithMixedOrders(t *testing.T, price1, price2, amount uint64) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV3(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, uint64(price1), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	require.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV4(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, uint64(price2), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	require.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(3, bo, so, uint64(price2), bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	require.NoError(t, err, "tx.Sign() failed")
	return tx
}

func createExchangeV2WithProofsWithOrdersV3(t *testing.T, price, amount uint64) *proto.ExchangeWithProofs {
	bo := proto.NewUnsignedOrderV3(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, uint64(price), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err := bo.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	require.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV3(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, uint64(price), uint64(amount), 0, 0, 3, *testGlobal.asset2.asset)
	err = so.Sign(proto.MainNetScheme, testGlobal.recipientInfo.sk)
	require.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithProofs(2, bo, so, uint64(price), bo.Amount, 1, 2, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.matcherInfo.sk)
	require.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffExchangeWithProofsWithOrdersV3(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)
		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createExchangeWithProofsWithOrdersV3(t)
	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	price := tx.Price * tx.Amount / priceConstant
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(-int64(tx.Amount), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[1]: newBalanceDiff(int64(price), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[2]: newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Amount), 0, 0, false),
		testGlobal.senderInfo.assetKeys[1]:    newBalanceDiff(-int64(price), 0, 0, false),
		testGlobal.senderInfo.assetKeys[2]:    newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.matcherInfo.assetKeys[2]:   newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
		testGlobal.matcherInfo.addr:   empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

//TODO: This test is based on real transaction from Testnet https://wavesexplorer.com/testnet/tx/6cEuK2q1FzhcVhiHUhYZXiZigroqTiRQ2Zswg139fcFW
// and produces an incorrect or unexpected diff, should be fixes some how
//func TestCreateDiffExchangeWithSignature(t *testing.T) {
//	to, path := createDifferTestObjects(t)
//
//	defer func() {
//		to.stor.close(t)
//		err := common.CleanTemporaryDirs(path)
//		assert.NoError(t, err, "failed to clean test data dirs")
//	}()
//
//	to.stor.createAssetWithDecimals(t, testGlobal.asset0.asset.ID, 8)
//	to.stor.createAssetWithDecimals(t, testGlobal.asset1.asset.ID, 8)
//
//	amount := uint64(394)
//	price := uint64(251566)
//
//	tx := createExchangeWithSigParams(t, price, amount)
//	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo(t))
//	assert.NoError(t, err, "createDiffExchange() failed")
//
//	priceAmount := price * amount
//	correctDiff := txDiff{
//		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(-int64(amount), 0, 0, false),
//		testGlobal.recipientInfo.assetKeys[1]: newBalanceDiff(int64(priceAmount), 0, 0, false),
//		testGlobal.recipientInfo.assetKeys[2]: newBalanceDiff(-int64(tx.SellMatcherFee), 0, 0, false),
//		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(int64(amount), 0, 0, false),
//		testGlobal.senderInfo.assetKeys[1]:    newBalanceDiff(-int64(priceAmount), 0, 0, false),
//		testGlobal.senderInfo.assetKeys[2]:    newBalanceDiff(-int64(tx.BuyMatcherFee), 0, 0, false),
//		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(tx.Fee), 0, 0, false),
//		testGlobal.matcherInfo.wavesKey:       newBalanceDiff(-int64(tx.Fee), 0, 0, false),
//		testGlobal.matcherInfo.assetKeys[2]:   newBalanceDiff(int64(tx.SellMatcherFee+tx.BuyMatcherFee), 0, 0, false),
//	}
//	correctAddrs := map[proto.Address]struct{}{
//		testGlobal.recipientInfo.addr: empty,
//		testGlobal.senderInfo.addr:    empty,
//		testGlobal.matcherInfo.addr:   empty,
//	}
//
//	assert.Equal(t, correctDiff, ch.diff)
//	assert.Equal(t, correctAddrs, ch.addrs)
//}
//
func TestCreateDiffExchangeV3WithProofsWithOrdersV4(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)
		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.createAssetWithDecimals(t, testGlobal.asset0.asset.ID, 0)
	to.stor.createAssetWithDecimals(t, testGlobal.asset1.asset.ID, 8)

	amount := uint64(1)
	price := uint64(10 * priceConstant)

	tx3o4 := createExchangeWithProofsWithOrdersV4(t, 10*priceConstant, amount)
	ch1, err := to.td.createDiffExchange(tx3o4, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	tx2o3 := createExchangeV2WithProofsWithOrdersV3(t, 10*priceConstant*priceConstant, amount)
	ch2, err := to.td.createDiffExchange(tx2o3, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	tx3mo := createExchangeV3WithProofsWithMixedOrders(t, 10*priceConstant*priceConstant, 10*priceConstant, amount)
	ch3, err := to.td.createDiffExchange(tx3mo, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffExchange() failed")

	priceAmount := price * amount
	correctDiff := txDiff{
		testGlobal.recipientInfo.assetKeys[0]: newBalanceDiff(-int64(amount), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[1]: newBalanceDiff(int64(priceAmount), 0, 0, false),
		testGlobal.recipientInfo.assetKeys[2]: newBalanceDiff(-int64(tx3o4.SellMatcherFee), 0, 0, false),
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(int64(amount), 0, 0, false),
		testGlobal.senderInfo.assetKeys[1]:    newBalanceDiff(-int64(priceAmount), 0, 0, false),
		testGlobal.senderInfo.assetKeys[2]:    newBalanceDiff(-int64(tx3o4.BuyMatcherFee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(tx3o4.Fee), 0, 0, false),
		testGlobal.matcherInfo.wavesKey:       newBalanceDiff(-int64(tx3o4.Fee), 0, 0, false),
		testGlobal.matcherInfo.assetKeys[2]:   newBalanceDiff(int64(tx3o4.SellMatcherFee+tx3o4.BuyMatcherFee), 0, 0, false),
	}
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
		testGlobal.matcherInfo.addr:   empty,
	}

	assert.Equal(t, correctDiff, ch1.diff)
	assert.Equal(t, correctAddrs, ch1.addrs)
	assert.Equal(t, correctDiff, ch2.diff)
	assert.Equal(t, correctAddrs, ch2.addrs)
	assert.Equal(t, correctDiff, ch3.diff)
	assert.Equal(t, correctAddrs, ch3.addrs)
}

func createLeaseWithSig(t *testing.T) *proto.LeaseWithSig {
	tx := proto.NewUnsignedLeaseWithSig(testGlobal.senderInfo.pk, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseWithSig(t)
	ch, err := to.td.createDiffLeaseWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseWithSig() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, int64(tx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, int64(tx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createLeaseWithProofs(t *testing.T) *proto.LeaseWithProofs {
	tx := proto.NewUnsignedLeaseWithProofs(2, testGlobal.senderInfo.pk, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), defaultAmount, defaultFee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseWithProofs(t)
	ch, err := to.td.createDiffLeaseWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, int64(tx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, int64(tx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createLeaseCancelWithSig(t *testing.T, leaseID crypto.Digest) *proto.LeaseCancelWithSig {
	tx := proto.NewUnsignedLeaseCancelWithSig(testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffLeaseCancelWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseWithSig(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, blockID0)
	err := to.tp.performLeaseWithSig(leaseTx, info)
	assert.NoError(t, err, "performLeaseWithSig failed")

	tx := createLeaseCancelWithSig(t, *leaseTx.ID)
	ch, err := to.td.createDiffLeaseCancelWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelWithSig() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, -int64(leaseTx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, -int64(leaseTx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createLeaseCancelWithProofs(t *testing.T, leaseID crypto.Digest) *proto.LeaseCancelWithProofs {
	tx := proto.NewUnsignedLeaseCancelWithProofs(2, 'W', testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffLeaseCancelWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseWithProofs(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, blockID0)
	err := to.tp.performLeaseWithProofs(leaseTx, info)
	assert.NoError(t, err, "performLeaseWithProofs failed")

	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)
	ch, err := to.td.createDiffLeaseCancelWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    newBalanceDiff(-int64(tx.Fee), 0, -int64(leaseTx.Amount), false),
		testGlobal.recipientInfo.wavesKey: newBalanceDiff(0, -int64(leaseTx.Amount), 0, false),
		testGlobal.minerInfo.wavesKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.recipientInfo.addr: empty,
		testGlobal.senderInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createCreateAliasWithSig(t *testing.T) *proto.CreateAliasWithSig {
	aliasStr := "alias"
	aliasFull := fmt.Sprintf("alias:W:%s", aliasStr)
	alias, err := proto.NewAliasFromString(aliasFull)
	assert.NoError(t, err, "NewAliasFromString() failed")
	tx := proto.NewUnsignedCreateAliasWithSig(testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffCreateAliasWithSig(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasWithSig(t)
	ch, err := to.td.createDiffCreateAliasWithSig(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffCreateAliasWithSig failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createCreateAliasWithProofs(t *testing.T) *proto.CreateAliasWithProofs {
	aliasStr := "alias"
	aliasFull := fmt.Sprintf("alias:W:%s", aliasStr)
	alias, err := proto.NewAliasFromString(aliasFull)
	assert.NoError(t, err, "NewAliasFromString() failed")
	tx := proto.NewUnsignedCreateAliasWithProofs(2, testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
	err = tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffCreateAliasWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasWithProofs(t)
	ch, err := to.td.createDiffCreateAliasWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffCreateAliasWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
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

func createMassTransferWithProofs(t *testing.T, transfers []proto.MassTransferEntry) *proto.MassTransferWithProofs {
	tx := proto.NewUnsignedMassTransferWithProofs(1, testGlobal.senderInfo.pk, *testGlobal.asset0.asset, transfers, defaultFee, defaultTimestamp, []byte("attachment"))
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffMassTransferWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	entriesNum := 66
	entries := generateMassTransferEntries(t, entriesNum)
	tx := createMassTransferWithProofs(t, entries)
	ch, err := to.td.createDiffMassTransferWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffMassTransferWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, true),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	for _, entry := range entries {
		recipientAddr, err := recipientToAddress(entry.Recipient, to.stor.entities.aliases, true)
		assert.NoError(t, err, "recipientToAddress() failed")
		err = correctDiff.appendBalanceDiff(byteKey(*recipientAddr, tx.Asset.ToID()), newBalanceDiff(int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
		err = correctDiff.appendBalanceDiff(byteKey(testGlobal.senderInfo.addr, tx.Asset.ToID()), newBalanceDiff(-int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
		correctAddrs[*recipientAddr] = empty
	}
	assert.Equal(t, correctDiff, ch.diff)
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createDataWithProofs(t *testing.T, entriesNum int) *proto.DataWithProofs {
	tx := proto.NewUnsignedData(1, testGlobal.senderInfo.pk, defaultFee, defaultTimestamp)
	for i := 0; i < entriesNum; i++ {
		entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
		tx.Entries = append(tx.Entries, entry)
	}
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffDataWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataWithProofs(t, 1)
	ch, err := to.td.createDiffDataWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffDataWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createSponsorshipWithProofs(t *testing.T, fee uint64) *proto.SponsorshipWithProofs {
	tx := proto.NewUnsignedSponsorshipWithProofs(1, testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, FeeUnit*fee, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffSponsorshipWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSponsorshipWithProofs(t, 1000)
	ch, err := to.td.createDiffSponsorshipWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffSponsorshipWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createSetScriptWithProofs(t *testing.T) *proto.SetScriptWithProofs {
	feeConst, ok := feeConstants[proto.SetScriptTransaction]
	assert.Equal(t, ok, true)
	tx := proto.NewUnsignedSetScriptWithProofs(1, 'W', testGlobal.senderInfo.pk, testGlobal.scriptBytes, FeeUnit*feeConst, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffSetScriptWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetScriptWithProofs(t)
	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffSetScriptWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createSetAssetScriptWithProofs(t *testing.T) *proto.SetAssetScriptWithProofs {
	feeConst, ok := feeConstants[proto.SetAssetScriptTransaction]
	assert.Equal(t, ok, true)
	tx := proto.NewUnsignedSetAssetScriptWithProofs(1, 'W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, testGlobal.scriptBytes, FeeUnit*feeConst, defaultTimestamp)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffSetAssetScriptWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createSetAssetScriptWithProofs(t)
	ch, err := to.td.createDiffSetAssetScriptWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffSetAssetScriptWithProofs failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createInvokeScriptWithProofs(t *testing.T, pmts proto.ScriptPayments, fc proto.FunctionCall, feeAsset proto.OptionalAsset, fee uint64) *proto.InvokeScriptWithProofs {
	tx := proto.NewUnsignedInvokeScriptWithProofs(1,
		'W',
		testGlobal.senderInfo.pk,
		proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr),
		fc,
		pmts,
		feeAsset,
		fee,
		defaultTimestamp,
	)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffInvokeScriptWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	feeConst, ok := feeConstants[proto.InvokeScriptTransaction]
	assert.Equal(t, ok, true)
	paymentAmount0 := uint64(100500)
	paymentAmount1 := uint64(90)
	paymentAmount2 := uint64(42)
	pmts := []proto.ScriptPayment{
		{Amount: paymentAmount0, Asset: *testGlobal.asset0.asset},
		{Amount: paymentAmount1, Asset: proto.OptionalAsset{Present: false}},
		{Amount: paymentAmount2, Asset: *testGlobal.asset0.asset},
	}
	totalAssetAmount := paymentAmount0 + paymentAmount2
	totalWavesAmount := paymentAmount1
	tx := createInvokeScriptWithProofs(t, pmts, proto.FunctionCall{}, *testGlobal.asset0.asset, feeConst*FeeUnit)

	assetId := tx.FeeAsset.ID
	to.stor.createAsset(t, assetId)

	to.stor.activateSponsorship(t)
	_, err := to.td.createDiffInvokeScriptWithProofs(tx, defaultDifferInfo(t))
	assert.Error(t, err, "createDiffInvokeScriptWithProofs() did not fail with unsponsored asset")
	err = to.stor.entities.sponsoredAssets.sponsorAsset(assetId, 10, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	ch, err := to.td.createDiffInvokeScriptWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffInvokeScriptWithProofs() failed with valid sponsored asset")

	feeInWaves, err := to.stor.entities.sponsoredAssets.sponsoredAssetToWaves(assetId, tx.Fee)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	recipientAssetDiff := balanceDiff{
		balance:                      int64(totalAssetAmount),
		updateMinIntermediateBalance: true,
		minBalance:                   int64(paymentAmount0),
	}
	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[0]:    newBalanceDiff(-int64(tx.Fee+totalAssetAmount), 0, 0, true),
		testGlobal.senderInfo.wavesKey:        newBalanceDiff(-int64(totalWavesAmount), 0, 0, true),
		testGlobal.recipientInfo.assetKeys[0]: recipientAssetDiff,
		testGlobal.recipientInfo.wavesKey:     newBalanceDiff(int64(totalWavesAmount), 0, 0, true),
		testGlobal.issuerInfo.assetKeys[0]:    newBalanceDiff(int64(tx.Fee), 0, 0, true),
		testGlobal.issuerInfo.wavesKey:        newBalanceDiff(-int64(feeInWaves), 0, 0, true),
		testGlobal.minerInfo.wavesKey:         newBalanceDiff(int64(feeInWaves), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr:    empty,
		testGlobal.recipientInfo.addr: empty,
		testGlobal.issuerInfo.addr:    empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

func createUpdateAssetInfoWithProofs(t *testing.T) *proto.UpdateAssetInfoWithProofs {
	tx := proto.NewUnsignedUpdateAssetInfoWithProofs(1, 'W', testGlobal.asset0.asset.ID, testGlobal.senderInfo.pk, "noname", "someDescription", defaultTimestamp, *(testGlobal.asset1.asset), defaultFee)
	err := tx.Sign(proto.MainNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "tx.Sign() failed")
	return tx
}

func TestCreateDiffUpdateAssetInfoWithProofs(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createUpdateAssetInfoWithProofs(t)
	ch, err := to.td.createDiffUpdateAssetInfoWithProofs(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffUpdateAssetInfoWithProofs() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKeys[1]: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.assetKeys[1]:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, ch.diff)
	correctAddrs := map[proto.Address]struct{}{
		testGlobal.senderInfo.addr: empty,
	}
	assert.Equal(t, correctAddrs, ch.addrs)
}

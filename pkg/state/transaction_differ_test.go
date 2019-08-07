package state

import (
	"fmt"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	defaultTimestamp = settings.MainNetSettings.CheckTempNegativeAfterTime
	defaultAmount    = uint64(100)
	defaultFee       = uint64(1)
	defaultQuantity  = uint64(1000)
	defaultDecimals  = byte(7)
)

type differTestObjects struct {
	stor     *storageObjects
	entities *blockchainEntitiesStorage
	td       *transactionDiffer
	tp       *transactionPerformer
}

func createDifferTestObjects(t *testing.T) (*differTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, stor.stateDB, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	td, err := newTransactionDiffer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	tp, err := newTransactionPerformer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerformer() failed")
	return &differTestObjects{stor, entities, td, tp}, path
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
	return proto.NewUnsignedPayment(testGlobal.senderInfo.pk, testGlobal.recipientInfo.addr, defaultAmount, defaultFee, defaultTimestamp)
}

func TestCreateDiffPayment(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedTransferV1(testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
}

func TestCreateDiffTransferV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t)
	diff, err := to.td.createDiffTransferV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV1() failed")

	senderCorrectDiff := newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true)
	senderCorrectDiff.minBalance = -int64(tx.Amount + tx.Fee)
	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    senderCorrectDiff,
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createTransferV2(t *testing.T) *proto.TransferV2 {
	return proto.NewUnsignedTransferV2(testGlobal.senderInfo.pk, *(testGlobal.asset0.asset), *(testGlobal.asset0.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
}

func TestCreateDiffTransferV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV2(t)
	diff, err := to.td.createDiffTransferV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffTransferV2() failed")

	senderCorrectDiff := newBalanceDiff(-int64(tx.Amount+tx.Fee), 0, 0, true)
	senderCorrectDiff.minBalance = -int64(tx.Amount + tx.Fee)
	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    senderCorrectDiff,
		testGlobal.recipientInfo.assetKey: newBalanceDiff(int64(tx.Amount), 0, 0, true),
		testGlobal.minerInfo.assetKey:     newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

func createIssueV1(t *testing.T) *proto.IssueV1 {
	tx := proto.NewUnsignedIssueV1(testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, defaultTimestamp, defaultFee)
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err := tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	tx := proto.NewUnsignedIssueV2('W', testGlobal.senderInfo.pk, "name", "description", defaultQuantity, defaultDecimals, true, []byte{}, defaultTimestamp, defaultFee)
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err := tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffIssueV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedReissueV1(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
}

func TestCreateDiffReissueV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedReissueV2('W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
}

func TestCreateDiffReissueV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedBurnV1(testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
}

func TestCreateDiffBurnV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedBurnV2('W', testGlobal.senderInfo.pk, testGlobal.asset0.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
}

func TestCreateDiffBurnV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	return proto.NewUnsignedExchangeV1(*bo, *so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
}

func TestCreateDiffExchangeV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	so := proto.NewUnsignedOrderV2(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 100, 0, 0, 3)
	return proto.NewUnsignedExchangeV2(*bo, *so, bo.Price, bo.Amount, 1, 2, defaultFee, defaultTimestamp)
}

func TestCreateDiffExchangeV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err := tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err := tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestCreateDiffLeaseV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedLeaseCancelV1(testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
}

func TestCreateDiffLeaseCancelV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV1(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, info.blockID)
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
	return proto.NewUnsignedLeaseCancelV2('W', testGlobal.senderInfo.pk, leaseID, defaultFee, defaultTimestamp)
}

func TestCreateDiffLeaseCancelV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV2(t)
	info := defaultPerformerInfo(t)
	to.stor.addBlock(t, info.blockID)
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
	return proto.NewUnsignedCreateAliasV1(testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
}

func TestCreateDiffCreateAliasV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedCreateAliasV2(testGlobal.senderInfo.pk, *alias, defaultFee, defaultTimestamp)
}

func TestCreateDiffCreateAliasV2(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
	return proto.NewUnsignedMassTransferV1(testGlobal.senderInfo.pk, *testGlobal.asset0.asset, transfers, defaultFee, defaultTimestamp, "attachment")
}

func TestCreateDiffMassTransferV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
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
		recipientAddr, err := recipientToAddress(entry.Recipient, to.entities.aliases, true)
		assert.NoError(t, err, "recipientToAddress() failed")
		err = correctDiff.appendBalanceDiff(byteKey(*recipientAddr, tx.Asset.ToID()), newBalanceDiff(int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
		err = correctDiff.appendBalanceDiff(byteKey(testGlobal.senderInfo.addr, tx.Asset.ToID()), newBalanceDiff(-int64(entry.Amount), 0, 0, true))
		assert.NoError(t, err, "appendBalanceDiff() failed")
	}
	assert.Equal(t, correctDiff, diff)
}

func createDataV1(t *testing.T) *proto.DataV1 {
	tx := proto.NewUnsignedData(testGlobal.senderInfo.pk, defaultFee, defaultTimestamp)
	tx.Entries = proto.DataEntries([]proto.DataEntry{&proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}})
	return tx
}

func TestCreateDiffDataV1(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createDataV1(t)
	diff, err := to.td.createDiffDataV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffDataV1 failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey: newBalanceDiff(-int64(tx.Fee), 0, 0, false),
		testGlobal.minerInfo.wavesKey:  newBalanceDiff(int64(tx.Fee), 0, 0, false),
	}
	assert.Equal(t, correctDiff, diff)
}

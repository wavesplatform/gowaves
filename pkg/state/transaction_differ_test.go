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
	defaultTimestamp = uint64(1465742577614)
	defaultAmount    = uint64(100)
	defaultFee       = uint64(1)
	defaultQuantity  = uint64(1000)
	defaultDecimals  = byte(7)
)

type differTestObjects struct {
	stor *storageObjects
	td   *transactionDiffer
	tp   *transactionPerformer
}

func createDifferTestObjects(t *testing.T) (*differTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	td, err := newTransactionDiffer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	tp, err := newTransactionPerformer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerformer() failed")
	return &differTestObjects{stor, td, tp}, path
}

func defaultDifferInfo(t *testing.T) *differInfo {
	return &differInfo{false, testGlobal.minerInfo.pk}
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
	correctDiff := txDiff{testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(tx.Amount)}}
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
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(tx.Amount + tx.Fee)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(tx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createTransferV1(t *testing.T) *proto.TransferV1 {
	return proto.NewUnsignedTransferV1(testGlobal.senderInfo.pk, *(testGlobal.asset), *(testGlobal.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
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

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    balanceDiff{balance: -int64(tx.Amount + tx.Fee)},
		testGlobal.recipientInfo.assetKey: balanceDiff{balance: int64(tx.Amount)},
		testGlobal.minerInfo.assetKey:     balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createTransferV2(t *testing.T) *proto.TransferV2 {
	return proto.NewUnsignedTransferV2(testGlobal.senderInfo.pk, *(testGlobal.asset), *(testGlobal.asset), defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), "attachment")
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

	correctDiff := txDiff{
		testGlobal.senderInfo.assetKey:    balanceDiff{balance: -int64(tx.Amount + tx.Fee)},
		testGlobal.recipientInfo.assetKey: balanceDiff{balance: int64(tx.Amount)},
		testGlobal.minerInfo.assetKey:     balanceDiff{balance: int64(tx.Fee)},
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
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): balanceDiff{balance: int64(tx.Quantity)},
		testGlobal.senderInfo.wavesKey:                       balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:                        balanceDiff{balance: int64(tx.Fee)},
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
		stringKey(testGlobal.senderInfo.addr, tx.ID.Bytes()): balanceDiff{balance: int64(tx.Quantity)},
		testGlobal.senderInfo.wavesKey:                       balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:                        balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createReissueV1(t *testing.T) *proto.ReissueV1 {
	return proto.NewUnsignedReissueV1(testGlobal.senderInfo.pk, testGlobal.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
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
		testGlobal.senderInfo.assetKey: balanceDiff{balance: int64(tx.Quantity)},
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createReissueV2(t *testing.T) *proto.ReissueV2 {
	return proto.NewUnsignedReissueV2('W', testGlobal.senderInfo.pk, testGlobal.asset.ID, defaultQuantity, false, defaultTimestamp, defaultFee)
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
		testGlobal.senderInfo.assetKey: balanceDiff{balance: int64(tx.Quantity)},
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createBurnV1(t *testing.T) *proto.BurnV1 {
	return proto.NewUnsignedBurnV1(testGlobal.senderInfo.pk, testGlobal.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
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
		testGlobal.senderInfo.assetKey: balanceDiff{balance: -int64(tx.Amount)},
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createBurnV2(t *testing.T) *proto.BurnV2 {
	return proto.NewUnsignedBurnV2('W', testGlobal.senderInfo.pk, testGlobal.asset.ID, defaultAmount, defaultTimestamp, defaultFee)
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
		testGlobal.senderInfo.assetKey: balanceDiff{balance: -int64(tx.Amount)},
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createExchangeV1(t *testing.T) *proto.ExchangeV1 {
	pa, _ := proto.NewOptionalAssetFromString("")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
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
		testGlobal.recipientInfo.assetKey: balanceDiff{balance: -int64(tx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(price - tx.SellMatcherFee)},
		testGlobal.senderInfo.assetKey:    balanceDiff{balance: int64(tx.Amount)},
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(price + tx.BuyMatcherFee)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
		testGlobal.matcherInfo.wavesKey:   balanceDiff{balance: int64(tx.SellMatcherFee + tx.BuyMatcherFee - tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

func createExchangeV2(t *testing.T) *proto.ExchangeV2 {
	pa, _ := proto.NewOptionalAssetFromString("")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
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
		testGlobal.recipientInfo.assetKey: balanceDiff{balance: -int64(tx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(price - tx.SellMatcherFee)},
		testGlobal.senderInfo.assetKey:    balanceDiff{balance: int64(tx.Amount)},
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(price + tx.BuyMatcherFee)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
		testGlobal.matcherInfo.wavesKey:   balanceDiff{balance: int64(tx.SellMatcherFee + tx.BuyMatcherFee - tx.Fee)},
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
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(tx.Fee), leaseOut: int64(tx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{leaseIn: int64(tx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
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
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(tx.Fee), leaseOut: int64(tx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{leaseIn: int64(tx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
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
	err := to.tp.performLeaseV1(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1 failed")

	tx := createLeaseCancelV1(t, *leaseTx.ID)
	diff, err := to.td.createDiffLeaseCancelV1(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelV1() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(tx.Fee), leaseOut: -int64(leaseTx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{leaseIn: -int64(leaseTx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
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
	err := to.tp.performLeaseV2(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2 failed")

	tx := createLeaseCancelV2(t, *leaseTx.ID)
	diff, err := to.td.createDiffLeaseCancelV2(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffLeaseCancelV2() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: -int64(tx.Fee), leaseOut: -int64(leaseTx.Amount)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{leaseIn: -int64(leaseTx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
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
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
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
		testGlobal.senderInfo.wavesKey: balanceDiff{balance: -int64(tx.Fee)},
		testGlobal.minerInfo.wavesKey:  balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}

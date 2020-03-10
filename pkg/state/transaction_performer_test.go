package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type performerTestObjects struct {
	stor *testStorageObjects
	tp   *transactionPerformer
}

func createPerformerTestObjects(t *testing.T) (*performerTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	tp, err := newTransactionPerformer(stor.entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerformer() failed")
	return &performerTestObjects{stor, tp}, path
}

func defaultPerformerInfo(t *testing.T) *performerInfo {
	return &performerInfo{false, 0, blockID0}
}

func TestPerformIssueWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createIssueWithSig(t, 1000)
	err := to.tp.performIssueWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueWithSig() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:   tx.SenderPK,
			decimals: int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: 1,
			reissuable:               tx.Reissuable,
		},
	}

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformIssueWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createIssueWithProofs(t, 1000)
	err := to.tp.performIssueWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueWithProofs() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:   tx.SenderPK,
			decimals: int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: 1,
			reissuable:               tx.Reissuable,
		},
	}

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformReissueWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueWithSig(t)
	err := to.tp.performReissueWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueWithSig() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueWithSig transaction")
}

func TestPerformReissueWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueWithProofs(t)
	err := to.tp.performReissueWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueWithProofs() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueWithSig transaction")
}

func TestPerformBurnWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithSig(t)
	err := to.tp.performBurnWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnWithSig() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithSig transaction")
}

func TestPerformBurnWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithProofs(t)
	err := to.tp.performBurnWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnWithProofs() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithProofs transaction")
}

func TestPerformExchange(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createExchangeWithSig(t)
	err := to.tp.performExchange(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performExchange() failed")

	sellOrderId, err := tx.GetSellOrderFull().GetID()
	assert.NoError(t, err)

	filledFee, err := to.stor.entities.ordersVolumes.newestFilledFee(sellOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)

	filledAmount, err := to.stor.entities.ordersVolumes.newestFilledAmount(sellOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	buyOrderId, err := tx.GetBuyOrderFull().GetID()
	assert.NoError(t, err)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(buyOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(buyOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	to.stor.flush(t)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(sellOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(sellOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(buyOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(buyOrderId, true)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)
}

func TestPerformLeaseWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithSig(t)
	err := to.tp.performLeaseWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithProofs(t)
	err := to.tp.performLeaseWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseCancelWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithSig(t)
	err := to.tp.performLeaseWithSig(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelWithSig(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelWithSig() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithSig transaction")
}

func TestPerformLeaseCancelWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithProofs(t)
	err := to.tp.performLeaseWithProofs(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelWithProofs() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithProofs transaction")
}

func TestPerformCreateAliasWithSig(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithSig(t)
	err := to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasWithSig transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformCreateAliasWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithProofs(t)
	err := to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasWithProofs transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformDataWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createDataWithProofs(t, 1)
	entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
	tx.Entries = proto.DataEntries([]proto.DataEntry{entry})

	err := to.tp.performDataWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performDataWithProofs() failed")
	to.stor.flush(t)

	newEntry, err := to.stor.entities.accountsDataStor.retrieveNewestEntry(testGlobal.senderInfo.addr, entry.Key, true)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry, newEntry)
}

func TestPerformSponsorshipWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSponsorshipWithProofs(t)
	err := to.tp.performSponsorshipWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSponsorshipWithProofs() failed")

	isSponsored, err := to.stor.entities.sponsoredAssets.newestIsSponsored(tx.AssetID, true)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err := to.stor.entities.sponsoredAssets.newestAssetCost(tx.AssetID, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)

	isSponsored, err = to.stor.entities.sponsoredAssets.isSponsored(tx.AssetID, true)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)

	to.stor.flush(t)

	isSponsored, err = to.stor.entities.sponsoredAssets.newestIsSponsored(tx.AssetID, true)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err = to.stor.entities.sponsoredAssets.newestAssetCost(tx.AssetID, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)

	isSponsored, err = to.stor.entities.sponsoredAssets.isSponsored(tx.AssetID, true)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err = to.stor.entities.sponsoredAssets.assetCost(tx.AssetID, true)
	assert.NoError(t, err, "assetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)
}

func TestPerformSetScriptWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSetScriptWithProofs(t)
	err := to.tp.performSetScriptWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSetScriptWithProofs() failed")

	addr := testGlobal.senderInfo.addr

	// Test newest before flushing.
	accountHasScript, err := to.stor.entities.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err := to.stor.entities.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err := to.stor.entities.scriptsStorage.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.stor.entities.scriptsStorage.scriptByAddr(addr, true)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.stor.entities.scriptsStorage.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.stor.entities.scriptsStorage.scriptByAddr(addr, true)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
}

func TestPerformSetAssetScriptWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSetAssetScriptWithProofs(t)
	err := to.tp.performSetAssetScriptWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSetAssetScriptWithProofs() failed")

	assetID := tx.AssetID

	// Test newest before flushing.
	isSmartAsset, err := to.stor.entities.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err, "newestIsSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err := to.stor.entities.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.scriptByAsset(assetID, true)
	assert.Error(t, err, "scriptByAsset() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err, "newestIsSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.scriptByAsset(assetID, true)
	assert.NoError(t, err, "scriptByAsset() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.stor.entities.scriptsStorage.setAssetScript(assetID, proto.Script{}, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err, "newestIsSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.Error(t, err)

	// Test stable before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.scriptByAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err, "newestIsSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.Error(t, err)

	// Test stable after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.scriptByAsset(assetID, true)
	assert.Error(t, err)
}

func TestPerformUpdateAssetInfoWithProofs(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createUpdateAssetInfoWithProofs(t)
	err := to.tp.performUpdateAssetInfoWithProofs(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performUpdateAssetInfoWithProofs() failed")
	to.stor.flush(t)
	assetInfo.name = tx.Name
	assetInfo.description = tx.Description

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(tx.AssetID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing UpdateAssetInfo transaction")
}

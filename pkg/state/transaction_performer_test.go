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
	return &performerInfo{false, blockID0}
}

func TestPerformIssueV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createIssueV1(t, 1000)
	err := to.tp.performIssueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueV1() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:      tx.SenderPK,
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
		},
	}

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func TestPerformIssueV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createIssueV2(t, 1000)
	err := to.tp.performIssueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueV2() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:      tx.SenderPK,
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
		},
	}

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func TestPerformReissueV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueV1(t)
	err := to.tp.performReissueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV1() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func TestPerformReissueV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueV2(t)
	err := to.tp.performReissueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV2() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func TestPerformBurnV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnV1(t)
	err := to.tp.performBurnV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnV1() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV1 transaction")
}

func TestPerformBurnV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnV2(t)
	err := to.tp.performBurnV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnV2() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV2 transaction")
}

func TestPerformExchange(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createExchangeV1(t)
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

func TestPerformLeaseV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createLeaseV1(t)
	err := to.tp.performLeaseV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseV1 transaction")
}

func TestPerformLeaseV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createLeaseV2(t)
	err := to.tp.performLeaseV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseV1 transaction")
}

func TestPerformLeaseCancelV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseV1(t)
	err := to.tp.performLeaseV1(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelV1(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV1() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelV1 transaction")
}

func TestPerformLeaseCancelV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseV2(t)
	err := to.tp.performLeaseV2(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelV2(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV2() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelV2 transaction")
}

func TestPerformCreateAliasV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasV1(t)
	err := to.tp.performCreateAliasV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV1() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasV1 transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV1() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformCreateAliasV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasV2(t)
	err := to.tp.performCreateAliasV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV2() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasV2 transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV2() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformDataV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createDataV1(t, 1)
	entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
	tx.Entries = proto.DataEntries([]proto.DataEntry{entry})

	err := to.tp.performDataV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performDataV1() failed")
	to.stor.flush(t)

	newEntry, err := to.stor.entities.accountsDataStor.retrieveNewestEntry(testGlobal.senderInfo.addr, entry.Key, true)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry, newEntry)
}

func TestPerformSponsorshipV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSponsorshipV1(t)
	err := to.tp.performSponsorshipV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSponsorshipV1() failed")

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

func TestPerformSetScriptV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSetScriptV1(t)
	err := to.tp.performSetScriptV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSetScriptV1() failed")

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

func TestPerformSetAssetScriptV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)

	tx := createSetAssetScriptV1(t)
	err := to.tp.performSetAssetScriptV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performSetAssetScriptV1() failed")

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

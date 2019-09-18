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
	tx := createIssueV1(t)
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
	tx := createIssueV2(t)
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

func TestPerfromSetScriptV1(t *testing.T) {
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
	hasScript, err := to.stor.entities.accountsScripts.newestHasScript(addr, true)
	assert.NoError(t, err, "newestHasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err := to.stor.entities.accountsScripts.newestHasVerifier(addr, true)
	assert.NoError(t, err, "newestHasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err := to.stor.entities.accountsScripts.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	hasScript, err = to.stor.entities.accountsScripts.hasScript(addr, true)
	assert.NoError(t, err, "hasScript() failed")
	assert.Equal(t, false, hasScript)
	hasVerifier, err = to.stor.entities.accountsScripts.hasVerifier(addr, true)
	assert.NoError(t, err, "hasVerifier() failed")
	assert.Equal(t, false, hasVerifier)
	_, err = to.stor.entities.accountsScripts.scriptByAddr(addr, true)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	hasScript, err = to.stor.entities.accountsScripts.newestHasScript(addr, true)
	assert.NoError(t, err, "newestHasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err = to.stor.entities.accountsScripts.newestHasVerifier(addr, true)
	assert.NoError(t, err, "newestHasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err = to.stor.entities.accountsScripts.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	hasScript, err = to.stor.entities.accountsScripts.hasScript(addr, true)
	assert.NoError(t, err, "hasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err = to.stor.entities.accountsScripts.hasVerifier(addr, true)
	assert.NoError(t, err, "hasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err = to.stor.entities.accountsScripts.scriptByAddr(addr, true)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
}

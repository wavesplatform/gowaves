package state

import (
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerTestObjects struct {
	stor *testStorageObjects
	tp   *transactionPerformer
}

func createPerformerTestObjects(t *testing.T) *performerTestObjects {
	stor := createStorageObjects(t, true)
	tp, err := newTransactionPerformer(stor.entities, settings.MainNetSettings)
	require.NoError(t, err, "newTransactionPerformer() failed")
	return &performerTestObjects{stor, tp}
}

func defaultPerformerInfo() *performerInfo {
	return &performerInfo{0, blockID0}
}

func TestPerformIssueWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createIssueWithSig(t, 1000)
	err := to.tp.performIssueWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performIssueWithSig() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			tail:     proto.DigestTail(*tx.ID),
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
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(*tx.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformIssueWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createIssueWithProofs(t, 1000)

	err := to.tp.performIssueWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performIssueWithProofs() failed")
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			tail:     proto.DigestTail(*tx.ID),
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
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(*tx.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformReissueWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueWithSig(t, 1000)
	err := to.tp.performReissueWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performReissueWithSig() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueWithSig transaction")
}

func TestPerformReissueWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueWithProofs(t, 1000)
	err := to.tp.performReissueWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performReissueWithProofs() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueWithSig transaction")
}

func TestPerformBurnWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithSig(t)
	err := to.tp.performBurnWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performBurnWithSig() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithSig transaction")
}

func TestPerformBurnWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithProofs(t)
	err := to.tp.performBurnWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performBurnWithProofs() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithProofs transaction")
}

func TestPerformExchange(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createExchangeWithSig(t)
	err := to.tp.performExchange(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performExchange() failed")

	sellOrderId, err := tx.GetOrder2().GetID()
	assert.NoError(t, err)

	filledFee, err := to.stor.entities.ordersVolumes.newestFilledFee(sellOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)

	filledAmount, err := to.stor.entities.ordersVolumes.newestFilledAmount(sellOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	buyOrderId, err := tx.GetOrder1().GetID()
	assert.NoError(t, err)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(buyOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(buyOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	to.stor.flush(t)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(sellOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(sellOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	filledFee, err = to.stor.entities.ordersVolumes.newestFilledFee(buyOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)

	filledAmount, err = to.stor.entities.ordersVolumes.newestFilledAmount(buyOrderId)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetAmount(), filledAmount)
}

func TestPerformLeaseWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithSig(t)
	err := to.tp.performLeaseWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		OriginTransactionID: tx.ID,
		Status:              LeaseActive,
		Amount:              tx.Amount,
		Recipient:           *tx.Recipient.Address(),
		Sender:              testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithProofs(t)
	err := to.tp.performLeaseWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		OriginTransactionID: tx.ID,
		Status:              LeaseActive,
		Amount:              tx.Amount,
		Recipient:           *tx.Recipient.Address(),
		Sender:              testGlobal.senderInfo.addr,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseCancelWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithSig(t)
	err := to.tp.performLeaseWithSig(leaseTx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	tx := createLeaseCancelWithSig(t, *leaseTx.ID)
	leasingInfo := &leasing{
		OriginTransactionID: leaseTx.ID,
		Status:              LeaseCanceled,
		Amount:              leaseTx.Amount,
		Recipient:           *leaseTx.Recipient.Address(),
		Sender:              testGlobal.senderInfo.addr,
		CancelTransactionID: tx.ID,
	}
	err = to.tp.performLeaseCancelWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseCancelWithSig() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithSig transaction")
}

func TestPerformLeaseCancelWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithProofs(t)
	err := to.tp.performLeaseWithProofs(leaseTx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)
	leasingInfo := &leasing{
		OriginTransactionID: leaseTx.ID,
		Status:              LeaseCanceled,
		Amount:              leaseTx.Amount,
		Recipient:           *leaseTx.Recipient.Address(),
		Sender:              testGlobal.senderInfo.addr,
		CancelTransactionID: tx.ID,
	}
	err = to.tp.performLeaseCancelWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performLeaseCancelWithProofs() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithProofs transaction")
}

func TestPerformCreateAliasWithSig(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithSig(t)
	err := to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasWithSig transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformCreateAliasWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithProofs(t)
	err := to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasWithProofs transaction")

	// Test stealing aliases.
	err = to.tp.performCreateAliasWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformDataWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)

	tx := createDataWithProofs(t, 1)
	entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
	tx.Entries = []proto.DataEntry{entry}

	err := to.tp.performDataWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performDataWithProofs() failed")
	to.stor.flush(t)

	newEntry, err := to.stor.entities.accountsDataStor.retrieveNewestEntry(testGlobal.senderInfo.addr, entry.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry, newEntry)
}

func TestPerformSponsorshipWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)

	tx := createSponsorshipWithProofs(t, 1000)
	err := to.tp.performSponsorshipWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performSponsorshipWithProofs() failed")

	assetID := proto.AssetIDFromDigest(tx.AssetID)

	isSponsored, err := to.stor.entities.sponsoredAssets.newestIsSponsored(assetID)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err := to.stor.entities.sponsoredAssets.newestAssetCost(assetID)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)

	isSponsored, err = to.stor.entities.sponsoredAssets.isSponsored(proto.AssetIDFromDigest(tx.AssetID))
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)

	to.stor.flush(t)

	isSponsored, err = to.stor.entities.sponsoredAssets.newestIsSponsored(assetID)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err = to.stor.entities.sponsoredAssets.newestAssetCost(assetID)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)

	isSponsored, err = to.stor.entities.sponsoredAssets.isSponsored(proto.AssetIDFromDigest(tx.AssetID))
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, true)

	assetCost, err = to.stor.entities.sponsoredAssets.assetCost(proto.AssetIDFromDigest(tx.AssetID))

	assert.NoError(t, err, "assetCost() failed")
	assert.Equal(t, assetCost, tx.MinAssetFee)
}

func TestPerformSetScriptWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)

	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func call() = nil

		@Verifier(tx)
		func verify() = sigVerify(tx.bodyBytes, tx.proofs[0], tx.senderPublicKey)
	*/
	const scriptWithVerifierAndDApp = "BgIECAISAAABAWkBBGNhbGwABQNuaWwBAnR4AQZ2ZXJpZnkACQD0AwMIBQJ0eAlib2R5Qnl0ZXMJAJEDAggFAnR4BnByb29mcwAACAUCdHgPc2VuZGVyUHVibGljS2V5HBD+0Q=="
	scriptBytes, err := base64.StdEncoding.DecodeString(scriptWithVerifierAndDApp)
	require.NoError(t, err)
	expectedScriptAST, err := scriptBytesToTree(scriptBytes)
	require.NoError(t, err)

	tx := createSetScriptWithProofs(t, scriptBytes)
	err = to.tp.performSetScriptWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performSetScriptWithProofs() failed")

	addr := testGlobal.senderInfo.addr

	// Test newest before flushing.
	accountHasScript, err := to.stor.entities.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountIsDApp, err := to.stor.entities.scriptsStorage.newestAccountIsDApp(addr)
	assert.NoError(t, err, "newestAccountIsDApp() failed")
	assert.Equal(t, true, accountIsDApp)
	accountHasVerifier, err := to.stor.entities.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAST, err := to.stor.entities.scriptsStorage.newestScriptByAddr(addr)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, expectedScriptAST, scriptAST)

	// Test stable before flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountIsDApp, err = to.stor.entities.scriptsStorage.accountIsDApp(addr)
	assert.NoError(t, err, "accountIsDApp() failed")
	assert.Equal(t, false, accountIsDApp)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.stor.entities.scriptsStorage.scriptByAddr(addr)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountIsDApp, err = to.stor.entities.scriptsStorage.newestAccountIsDApp(addr)
	assert.NoError(t, err, "newestAccountIsDApp() failed")
	assert.Equal(t, true, accountIsDApp)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAST, err = to.stor.entities.scriptsStorage.newestScriptByAddr(addr)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, expectedScriptAST, scriptAST)

	// Test stable after flushing.
	accountHasScript, err = to.stor.entities.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountIsDApp, err = to.stor.entities.scriptsStorage.accountIsDApp(addr)
	assert.NoError(t, err, "accountIsDApp() failed")
	assert.Equal(t, true, accountIsDApp)
	accountHasVerifier, err = to.stor.entities.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAST, err = to.stor.entities.scriptsStorage.scriptByAddr(addr)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, expectedScriptAST, scriptAST)
}

func TestPerformSetAssetScriptWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	to.stor.addBlock(t, blockID0)

	tx := createSetAssetScriptWithProofs(t)
	err := to.tp.performSetAssetScriptWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performSetAssetScriptWithProofs() failed")

	fullAssetID := tx.AssetID
	shortAssetID := proto.AssetIDFromDigest(fullAssetID)

	// Test newest before flushing.
	isSmartAsset, err := to.stor.entities.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err := to.stor.entities.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.scriptByAsset(shortAssetID)
	assert.Error(t, err, "scriptByAsset() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.scriptByAsset(shortAssetID)
	assert.NoError(t, err, "scriptByAsset() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.stor.entities.scriptsStorage.setAssetScript(fullAssetID, proto.Script{}, crypto.PublicKey{}, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.Error(t, err)

	// Test stable before flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.stor.entities.scriptsStorage.scriptByAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.Error(t, err)

	// Test stable after flushing.
	isSmartAsset, err = to.stor.entities.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.stor.entities.scriptsStorage.scriptByAsset(shortAssetID)
	assert.Error(t, err)
}

func TestPerformUpdateAssetInfoWithProofs(t *testing.T) {
	to := createPerformerTestObjects(t)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createUpdateAssetInfoWithProofs(t)
	err := to.tp.performUpdateAssetInfoWithProofs(tx, defaultPerformerInfo())
	assert.NoError(t, err, "performUpdateAssetInfoWithProofs() failed")
	to.stor.flush(t)
	assetInfo.name = tx.Name
	assetInfo.description = tx.Description

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(tx.AssetID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing UpdateAssetInfo transaction")
}

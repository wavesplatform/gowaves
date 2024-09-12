package state

import (
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerTestObjects struct {
	stor *testStorageObjects
	th   *transactionHandler
}

func createPerformerTestObjects(t *testing.T, checkerInfo *checkerInfo) *performerTestObjects {
	sets := settings.MustMainNetSettings()
	stor := createStorageObjects(t, true)
	snapshotApplier := newBlockSnapshotsApplier(
		newBlockSnapshotsApplierInfo(checkerInfo, sets.AddressSchemeCharacter),
		newSnapshotApplierStorages(stor.entities, stor.rw),
	)
	buildAPIData, err := stor.stateDB.stateStoresApiData()
	require.NoError(t, err)
	blockID := stor.settings.Genesis.BlockID()
	th, err := newTransactionHandler(blockID, stor.entities, sets, &snapshotApplier, buildAPIData)
	require.NoError(t, err)
	return &performerTestObjects{stor, th}
}

func defaultPerformerInfo() *performerInfo {
	return newPerformerInfo(0, blockID0, proto.WavesAddress{}, txCheckerData{})
}

func defaultCheckerInfoHeight0() *checkerInfo {
	return &checkerInfo{
		currentTimestamp: defaultTimestamp,
		parentTimestamp:  defaultTimestamp - settings.MustMainNetSettings().MaxTxTimeBackOffset/2,
		blockID:          blockID0,
		blockVersion:     1,
		blockchainHeight: 0,
	}
}

func TestPerformFailedTx(t *testing.T) {
	const (
		applicationStatus = false
		validatingUTX     = false
	)
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)
	assetID := testGlobal.asset0.asset.ID
	_ = to.stor.createAsset(t, assetID)
	tx := createReissueWithSig(t, 1000)
	snapshot, err := to.th.performTx(tx, defaultPerformerInfo(), validatingUTX, nil, applicationStatus, nil)
	assert.NoError(t, err, "performReissueWithSig() failed")

	// Check tx snapshot
	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.TransactionStatusSnapshot{
				Status: proto.TransactionFailed,
			},
		},
		internal: nil,
	}
	assert.Equal(t, expectedSnapshot, snapshot)
	to.stor.flush(t)

	// Check that tx was written
	info, err := to.stor.rw.transactionInfoByID(tx.ID.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, proto.TransactionFailed, info.txStatus)
}

func TestPerformFailedTxWhenValidatingUtx(t *testing.T) {
	const (
		applicationStatus = false
		validatingUTX     = true
	)
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)
	assetID := testGlobal.asset0.asset.ID
	_ = to.stor.createAsset(t, assetID)
	tx := createReissueWithSig(t, 1000)
	snapshot, err := to.th.performTx(tx, defaultPerformerInfo(), validatingUTX, nil, applicationStatus, nil)
	assert.NoError(t, err, "performReissueWithSig() failed")

	// Check tx snapshot
	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.TransactionStatusSnapshot{
				Status: proto.TransactionFailed,
			},
		},
		internal: nil,
	}
	assert.Equal(t, expectedSnapshot, snapshot)

	// Check that tx was written to in memory storage
	info, err := to.stor.rw.newestTransactionInfoByID(tx.ID.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, proto.TransactionFailed, info.txStatus)

	to.stor.flush(t)

	// Check that tx doesn't exist after flush
	_, err = to.stor.rw.transactionInfoByID(tx.ID.Bytes())
	assert.Error(t, err)
}

func TestPerformIssueWithSig(t *testing.T) {
	checkerInfo := defaultCheckerInfoHeight0()
	to := createPerformerTestObjects(t, checkerInfo)
	to.stor.addBlock(t, blockID0)
	tx := createIssueWithSig(t, 1000)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performIssueWithSig() failed")
	to.stor.flush(t)
	expectedAssetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:                 proto.DigestTail(*tx.ID),
			Issuer:               tx.SenderPK,
			Decimals:             tx.Decimals,
			IssueHeight:          1,
			IsNFT:                false,
			IssueSequenceInBlock: 1,
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
	assert.Equal(t, expectedAssetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformIssueWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfoHeight0()
	to := createPerformerTestObjects(t, checkerInfo)
	to.stor.addBlock(t, blockID0)
	tx := createIssueWithProofs(t, 1000)

	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performIssueWithProofs() failed")
	to.stor.flush(t)
	expectedAssetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:                 proto.DigestTail(*tx.ID),
			Issuer:               tx.SenderPK,
			Decimals:             tx.Decimals,
			IssueHeight:          1,
			IsNFT:                false,
			IssueSequenceInBlock: 1,
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
	assert.Equal(t, expectedAssetInfo, *info, "invalid asset info after performing IssueWithSig transaction")
}

func TestPerformReissueWithSig(t *testing.T) {

	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)
	assetID := testGlobal.asset0.asset.ID

	assetInfo := to.stor.createAsset(t, assetID)
	tx := createReissueWithSig(t, 1000)
	snapshot, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performReissueWithSig() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(assetID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueWithSig transaction")

	// Check tx snapshot
	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.AssetVolumeSnapshot{
				AssetID:       assetID,
				TotalQuantity: info.quantity,
				IsReissuable:  info.reissuable,
			},
			&proto.TransactionStatusSnapshot{
				Status: proto.TransactionSucceeded,
			},
		},
		internal: nil,
	}
	assert.Equal(t, expectedSnapshot, snapshot)

	to.stor.flush(t)

	// Check that tx was written
	txInf, err := to.stor.rw.transactionInfoByID(tx.ID.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, proto.TransactionSucceeded, txInf.txStatus)
}

func TestPerformReissueWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createReissueWithProofs(t, 1000)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
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
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithSig(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performBurnWithSig() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithSig transaction")
}

func TestPerformBurnWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createBurnWithProofs(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performBurnWithProofs() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(testGlobal.asset0.asset.ID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnWithProofs transaction")
}

func TestPerformExchange(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	tx := createExchangeWithSig(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performExchange() failed")

	sellOrderID, err := tx.GetOrder2().GetID()
	assert.NoError(t, err)

	filledAmount, filledFee, err := to.stor.entities.ordersVolumes.newestFilled(sellOrderID)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	buyOrderID, err := tx.GetOrder1().GetID()
	assert.NoError(t, err)

	filledAmount, filledFee, err = to.stor.entities.ordersVolumes.newestFilled(buyOrderID)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	to.stor.flush(t)

	filledAmount, filledFee, err = to.stor.entities.ordersVolumes.newestFilled(sellOrderID)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetSellMatcherFee(), filledFee)
	assert.Equal(t, tx.GetAmount(), filledAmount)

	filledAmount, filledFee, err = to.stor.entities.ordersVolumes.newestFilled(buyOrderID)
	assert.NoError(t, err)
	assert.Equal(t, tx.GetBuyMatcherFee(), filledFee)
	assert.Equal(t, tx.GetAmount(), filledAmount)
}

func TestPerformLeaseWithSig(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithSig(t)
	pi := defaultPerformerInfo()
	_, err := to.th.performTx(tx, pi, false, nil, true, nil)
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		OriginTransactionID: tx.ID,
		OriginHeight:        pi.blockHeight(),
		Status:              LeaseActive,
		Amount:              tx.Amount,
		RecipientAddr:       *tx.Recipient.Address(),
		SenderPK:            testGlobal.senderInfo.pk,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	tx := createLeaseWithProofs(t)
	pi := defaultPerformerInfo()
	_, err := to.th.performTx(tx, pi, false, nil, true, nil)
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	leasingInfo := &leasing{
		OriginTransactionID: tx.ID,
		OriginHeight:        pi.blockHeight(),
		Status:              LeaseActive,
		Amount:              tx.Amount,
		RecipientAddr:       *tx.Recipient.Address(),
		SenderPK:            testGlobal.senderInfo.pk,
	}

	info, err := to.stor.entities.leases.leasingInfo(*tx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseWithSig transaction")
}

func TestPerformLeaseCancelWithSig(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithSig(t)
	pi := defaultPerformerInfo()
	_, err := to.th.performTx(leaseTx, pi, false, nil, true, nil)
	assert.NoError(t, err, "performLeaseWithSig() failed")
	to.stor.flush(t)
	tx := createLeaseCancelWithSig(t, *leaseTx.ID)
	leasingInfo := &leasing{
		OriginTransactionID: leaseTx.ID,
		OriginHeight:        pi.blockHeight(),
		Status:              LeaseCancelled,
		Amount:              leaseTx.Amount,
		RecipientAddr:       *leaseTx.Recipient.Address(),
		SenderPK:            testGlobal.senderInfo.pk,
		CancelTransactionID: tx.ID,
		CancelHeight:        pi.blockHeight(),
	}
	_, err = to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performLeaseCancelWithSig() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithSig transaction")
}

func TestPerformLeaseCancelWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	leaseTx := createLeaseWithProofs(t)
	pi := defaultPerformerInfo()
	_, err := to.th.performTx(leaseTx, pi, false, nil, true, nil)
	assert.NoError(t, err, "performLeaseWithProofs() failed")
	to.stor.flush(t)
	tx := createLeaseCancelWithProofs(t, *leaseTx.ID)
	leasingInfo := &leasing{
		OriginTransactionID: leaseTx.ID,
		OriginHeight:        pi.blockHeight(),
		Status:              LeaseCancelled,
		Amount:              leaseTx.Amount,
		RecipientAddr:       *leaseTx.Recipient.Address(),
		SenderPK:            testGlobal.senderInfo.pk,
		CancelTransactionID: tx.ID,
		CancelHeight:        pi.blockHeight(),
	}
	_, err = to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performLeaseCancelWithProofs() failed")
	to.stor.flush(t)
	info, err := to.stor.entities.leases.leasingInfo(*leaseTx.ID)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelWithProofs transaction")
}

func TestPerformCreateAliasWithSig(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithSig(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, addr, "invalid address by alias after performing CreateAliasWithSig transaction")

	// Test stealing aliases.
	_, err = to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performCreateAliasWithSig() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases(blockID0)
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformCreateAliasWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	tx := createCreateAliasWithProofs(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	addr, err := to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, addr, "invalid address by alias after performing CreateAliasWithProofs transaction")

	// Test stealing aliases.
	_, err = to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performCreateAliasWithProofs() failed")
	to.stor.flush(t)
	err = to.stor.entities.aliases.disableStolenAliases(blockID0)
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.stor.flush(t)
	_, err = to.stor.entities.aliases.addrByAlias(tx.Alias.Alias)
	assert.Equal(t, errAliasDisabled, err)
}

func TestPerformDataWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)

	tx := createDataWithProofs(t, 1)
	entry := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(666)}
	tx.Entries = []proto.DataEntry{entry}

	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performDataWithProofs() failed")
	to.stor.flush(t)

	newEntry, err := to.stor.entities.accountsDataStor.retrieveNewestEntry(testGlobal.senderInfo.addr, entry.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry, newEntry)
}

func TestPerformSponsorshipWithProofs(t *testing.T) {
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)

	tx := createSponsorshipWithProofs(t, 1000)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
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
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

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
	pi := *defaultPerformerInfo()
	pi.checkerData.scriptEstimation = &scriptEstimation{}
	_, err = to.th.performTx(tx, &pi, false, nil, true, nil)

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
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)

	tx := createSetAssetScriptWithProofs(t)
	pi := *defaultPerformerInfo()

	currentEstimatorVersion := 4
	tree, err := serialization.Parse(tx.Script)
	require.NoError(t, err)
	estimation, err := ride.EstimateTree(tree, currentEstimatorVersion)
	require.NoError(t, err)

	pi.checkerData.scriptEstimation = &scriptEstimation{
		currentEstimatorVersion: currentEstimatorVersion,
		scriptIsEmpty:           false,
		estimation:              estimation,
	}
	checkerInfo.blockID = blockID0
	_, err = to.th.performTx(tx, &pi, false, nil, true, nil)
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
	err = to.stor.entities.scriptsStorage.setAssetScript(fullAssetID, proto.Script{}, blockID0)
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
	checkerInfo := defaultCheckerInfo()
	to := createPerformerTestObjects(t, checkerInfo)
	// because update asset info tx has only protobuf binary representation
	to.stor.activateFeature(t, int16(settings.BlockV5))
	to.stor.rw.setProtobufActivated()

	assetInfo := to.stor.createAsset(t, testGlobal.asset0.asset.ID)
	tx := createUpdateAssetInfoWithProofs(t)
	_, err := to.th.performTx(tx, defaultPerformerInfo(), false, nil, true, nil)
	assert.NoError(t, err, "performUpdateAssetInfoWithProofs() failed")
	to.stor.flush(t)
	assetInfo.name = tx.Name
	assetInfo.description = tx.Description
	assetInfo.lastNameDescChangeHeight = checkerInfo.blockchainHeight + 1

	// Check asset info.
	info, err := to.stor.entities.assets.assetInfo(proto.AssetIDFromDigest(tx.AssetID))
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing UpdateAssetInfo transaction")
}

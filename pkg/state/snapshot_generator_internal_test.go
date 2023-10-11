package state

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func defaultAssetInfoTransfer(tail [12]byte, reissuable bool,
	amount int64, issuer crypto.PublicKey,
	name string) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:     tail,
			issuer:   issuer,
			decimals: 2,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(amount),
			name:                     name,
			description:              "description",
			lastNameDescChangeHeight: 1,
			reissuable:               reissuable,
		},
	}
}

func defaultPerformerInfoWithChecker(checkerData txCheckerData) *performerInfo {
	return &performerInfo{0, blockID0, proto.WavesAddress{}, new(proto.StateActionsCounter), checkerData}
}

func customCheckerInfo() *checkerInfo {
	defaultBlockInfo := defaultBlockInfo()
	return &checkerInfo{
		currentTimestamp: defaultBlockInfo.Timestamp,
		parentTimestamp:  defaultTimestamp - settings.MainNetSettings.MaxTxTimeBackOffset/2,
		blockID:          blockID0,
		blockVersion:     defaultBlockInfo.Version,
		height:           defaultBlockInfo.Height,
	}
}

func createCheckerCustomTestObjects(t *testing.T, differ *differTestObjects) *checkerTestObjects {
	tc, err := newTransactionChecker(proto.NewBlockIDFromSignature(genSig), differ.stor.entities, settings.MainNetSettings)
	require.NoError(t, err, "newTransactionChecker() failed")
	return &checkerTestObjects{differ.stor, tc, differ.tp, differ.stateActionsCounter}
}

func TestDefaultTransferWavesAndAssetSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	err := to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedTransferWithSig(testGlobal.issuerInfo.pk,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), defaultTimestamp,
		defaultAmount*1000*2, uint64(FeeUnit), testGlobal.recipientInfo.Recipient(), nil)
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign transfer tx")

	ch, err := to.td.createDiffTransferWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffTransferWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performTransferWithSig(tx,
		defaultPerformerInfo(to.stateActionsCounter), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform transfer tx")
	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299700000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			Balance: 200000,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

// TODO send only txBalanceChanges to perfomer
func TestDefaultIssueTransactionSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	tx := proto.NewUnsignedIssueWithSig(testGlobal.issuerInfo.pk,
		"asset0", "description", defaultQuantity, defaultDecimals,
		true, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign issue tx")

	ch, err := to.td.createDiffIssueWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffIssueWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performIssueWithSig(tx,
		defaultPerformerInfo(to.stateActionsCounter), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform issue tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.StaticAssetInfoSnapshot{
			AssetID:             *tx.ID,
			SourceTransactionID: *tx.ID,
			IssuerPublicKey:     testGlobal.issuerInfo.pk,
			Decimals:            defaultDecimals,
			IsNFT:               false},
		&proto.AssetDescriptionSnapshot{
			AssetID:          *tx.ID,
			AssetName:        "asset0",
			AssetDescription: "description",
			ChangeHeight:     1,
		},
		&proto.AssetVolumeSnapshot{
			AssetID:       *tx.ID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity)),
			IsReissuable:  true,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: *tx.ID,
			Balance: 1000,
		},
		&proto.AssetScriptSnapshot{
			AssetID:            *tx.ID,
			Script:             proto.Script{},
			SenderPK:           crypto.PublicKey{},
			VerifierComplexity: 0,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultReissueSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID),
		defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID),
			true, 1000, testGlobal.issuerInfo.pk, "asset0"), blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.issuerInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset0.assetID), 1000, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedReissueWithSig(testGlobal.issuerInfo.pk,
		testGlobal.asset0.assetID, 50,
		false, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign reissue tx")

	ch, err := to.td.createDiffReissueWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffReissueWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performReissueWithSig(tx,
		defaultPerformerInfo(to.stateActionsCounter), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform reissue tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 1050,
		},
		&proto.AssetVolumeSnapshot{
			AssetID:       testGlobal.asset0.assetID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity + 50)),
			IsReissuable:  false,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultBurnSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID),
		defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID),
			false, 950, testGlobal.issuerInfo.pk, "asset0"), blockID0)

	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.issuerInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset0.assetID), 1000, blockID0)
	assert.NoError(t, err, "failed to set asset balance")

	tx := proto.NewUnsignedBurnWithSig(testGlobal.issuerInfo.pk,
		testGlobal.asset0.assetID, 50, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffBurnWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performBurnWithSig(tx,
		defaultPerformerInfo(to.stateActionsCounter), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 950,
		},
		&proto.AssetVolumeSnapshot{
			AssetID:       testGlobal.asset0.assetID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity - 100)),
			IsReissuable:  false,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultExchangeTransaction(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	// issue assets
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID),
		defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID),
			true, 1000, testGlobal.senderInfo.pk, "asset0"), blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset1.assetID),
		defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset1.assetID),
			true, 1000, testGlobal.recipientInfo.pk, "asset1"), blockID0)
	assert.NoError(t, err, "failed to issue asset")

	// set waves balance for the seller and the buyer
	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 2000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	// set waves balance for the matcher account
	err = to.stor.entities.balances.setWavesBalance(testGlobal.matcherInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 3000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	// set asset balance for the seller and the buyer
	err = to.stor.entities.balances.setAssetBalance(testGlobal.senderInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset1.assetID), 500, blockID0)
	assert.NoError(t, err, "failed to set asset balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.recipientInfo.addr.ID(),
		proto.AssetIDFromDigest(testGlobal.asset0.assetID), 600, blockID0)
	assert.NoError(t, err, "failed to set asset balance")

	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk,
		*testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy,
		10e8, 10, 0, 0, 3)
	err = bo.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk,
		*testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell,
		10e8, 10, 0, 0, 3)
	err = so.Sign(proto.TestNetScheme, testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithSig(bo, so, bo.Price, bo.Amount, 1, 2, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.matcherInfo.sk)

	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performExchange(tx, defaultPerformerInfo(to.stateActionsCounter),
		nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299999999,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			Balance: 599999998,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.matcherInfo.addr,
			Balance: 899900003,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 10,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 590,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: testGlobal.asset1.assetID,
			Balance: 400,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			AssetID: testGlobal.asset1.assetID,
			Balance: 100,
		},
		&proto.FilledVolumeFeeSnapshot{
			OrderID:      *bo.ID,
			FilledVolume: 10,
			FilledFee:    1,
		},
		&proto.FilledVolumeFeeSnapshot{
			OrderID:      *so.ID,
			FilledVolume: 10,
			FilledFee:    2,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedLeaseWithSig(testGlobal.senderInfo.pk, testGlobal.recipientInfo.Recipient(),
		50, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffLeaseWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performLeaseWithSig(tx, defaultPerformerInfo(to.stateActionsCounter),
		nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.LeaseStateSnapshot{
			LeaseID: *tx.ID,
			Status: proto.LeaseStateStatus{
				Value: proto.LeaseActive,
			},
			Amount:              50,
			Sender:              testGlobal.senderInfo.addr,
			Recipient:           testGlobal.recipientInfo.addr,
			OriginTransactionID: tx.ID,
			Height:              0,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 50,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  50,
			LeaseOut: 0,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseCancelSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	leaseID := testGlobal.asset0.assetID
	leasing := &leasing{
		Sender:              testGlobal.senderInfo.addr,
		Recipient:           testGlobal.recipientInfo.addr,
		Amount:              50,
		Height:              1,
		Status:              proto.LeaseActive,
		OriginTransactionID: &leaseID,
	}
	err := to.stor.entities.leases.addLeasing(leaseID, leasing, blockID0)
	assert.NoError(t, err, "failed to add leasing")

	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3,
			leaseIn: 0, leaseOut: 50}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3, leaseIn: 50, leaseOut: 0}},
		blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedLeaseCancelWithSig(testGlobal.senderInfo.pk, leaseID, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffLeaseCancelWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performLeaseCancelWithSig(tx, defaultPerformerInfo(to.stateActionsCounter),
		nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.LeaseStateSnapshot{
			LeaseID: leaseID,
			Status: proto.LeaseStateStatus{
				Value:               proto.LeaseCanceled,
				CancelHeight:        0,
				CancelTransactionID: tx.ID,
			},
			Amount:              50,
			Sender:              testGlobal.senderInfo.addr,
			Recipient:           testGlobal.recipientInfo.addr,
			OriginTransactionID: &leaseID,
			Height:              1,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
	}
	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultCreateAliasSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	alias := proto.NewAlias(proto.TestNetScheme, "aliasForSender")
	tx := proto.NewUnsignedCreateAliasWithSig(testGlobal.senderInfo.pk, *alias, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffCreateAliasWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo(to.stateActionsCounter),
		nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.AliasSnapshot{
			Address: testGlobal.senderInfo.addr,
			Alias:   *proto.NewAlias(proto.TestNetScheme, "aliasForSender"),
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultDataSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(
		testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}},
		blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedDataWithProofs(1, testGlobal.senderInfo.pk, uint64(1*FeeUnit), defaultTimestamp)
	stringEntry := &proto.StringDataEntry{Key: "key_str", Value: "value_str"}
	intEntry := &proto.IntegerDataEntry{Key: "key_int", Value: 2}
	err = tx.AppendEntry(stringEntry)
	require.NoError(t, err)
	err = tx.AppendEntry(intEntry)
	require.NoError(t, err)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffDataWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performDataWithProofs(tx, defaultPerformerInfo(to.stateActionsCounter),
		nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.DataEntriesSnapshot{
			Address: testGlobal.senderInfo.addr,
			DataEntries: []proto.DataEntry{&proto.StringDataEntry{Key: "key_str", Value: "value_str"},
				&proto.IntegerDataEntry{Key: "key_int", Value: 2}},
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSponsorshipSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSponsorshipWithProofs(1, testGlobal.senderInfo.pk,
		testGlobal.asset0.assetID, uint64(5*FeeUnit), uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffSponsorshipWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSponsorshipWithProofs(tx,
		defaultPerformerInfo(to.stateActionsCounter), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.SponsorshipSnapshot{
			AssetID:         testGlobal.asset0.assetID,
			MinSponsoredFee: 500000,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetScriptSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSetScriptWithProofs(1, testGlobal.senderInfo.pk,
		testGlobal.scriptBytes, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign set script tx")

	co := createCheckerCustomTestObjects(t, to)
	co.stor = to.stor
	checkerData, err := co.tc.checkSetScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetScriptWithProofs(tx,
		defaultPerformerInfoWithChecker(checkerData), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.AccountScriptSnapshot{
			SenderPublicKey:    testGlobal.senderInfo.pk,
			Script:             testGlobal.scriptBytes,
			VerifierComplexity: 340,
		},
		&InternalDAppComplexitySnapshot{
			ScriptAddress: testGlobal.senderInfo.addr,
			Estimation:    ride.TreeEstimation{Estimation: 340, Verifier: 340},
			Update:        false,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetEmptyScriptSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSetScriptWithProofs(1, testGlobal.senderInfo.pk,
		nil, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign set script tx")

	co := createCheckerCustomTestObjects(t, to)
	co.stor = to.stor
	checkerData, err := co.tc.checkSetScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetScriptWithProofs(tx,
		defaultPerformerInfoWithChecker(checkerData), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&proto.AccountScriptSnapshot{
			SenderPublicKey:    testGlobal.senderInfo.pk,
			Script:             nil,
			VerifierComplexity: 0,
		},
		&InternalDAppComplexitySnapshot{
			ScriptAddress: testGlobal.senderInfo.addr,
			Estimation:    ride.TreeEstimation{Estimation: 0, Verifier: 0},
			Update:        false,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})
	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetAssetScriptSnapshot(t *testing.T) {
	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	var err error
	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	err = to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID),
		defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID),
			true, 1000, testGlobal.senderInfo.pk, "asset0"), blockID0)
	assert.NoError(t, err, "failed to issue asset")

	err = to.stor.entities.scriptsStorage.setAssetScript(testGlobal.asset0.assetID,
		testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "failed to issue asset")

	tx := proto.NewUnsignedSetAssetScriptWithProofs(1, testGlobal.senderInfo.pk,
		testGlobal.asset0.assetID, testGlobal.scriptBytes, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")

	co := createCheckerCustomTestObjects(t, to)
	co.stor = to.stor
	checkerData, err := co.tc.checkSetAssetScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetAssetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetAssetScriptWithProofs(tx,
		defaultPerformerInfoWithChecker(checkerData), nil, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},

		&proto.AssetScriptSnapshot{
			AssetID:            testGlobal.asset0.assetID,
			Script:             testGlobal.scriptBytes,
			VerifierComplexity: 340,
			SenderPK:           tx.SenderPK,
		},
	}

	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func setScript(t *testing.T, to *differTestObjects, addr proto.WavesAddress, pk crypto.PublicKey, script proto.Script) {
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	estimation, err := ride.EstimateTree(tree, 1)
	require.NoError(t, err)
	scriptEst := scriptEstimation{currentEstimatorVersion: 1, scriptIsEmpty: false, estimation: estimation}
	err = to.stor.entities.scriptsComplexity.saveComplexitiesForAddr(addr,
		scriptEst, blockID0)
	assert.NoError(t, err, "failed to save complexity for address")
	err = to.stor.entities.scriptsStorage.setAccountScript(addr, script, pk, blockID0)
	assert.NoError(t, err, "failed to set account script")
}

func TestDefaultInvokeScriptSnapshot(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func call() = {
		  [
		    BooleanEntry("bool", true),
		    IntegerEntry("int", 1),
		    StringEntry("str", "")
		  ]
		}
	*/
	script := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAARMAAAAAgkBAAAA" +
		"DEJvb2xlYW5FbnRyeQAAAAICAAAABGJvb2wGCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAAN" +
		"pbnQAAAAAAAAAAAEJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAADc3RyAgAAAAAFAAAAA25pbAAAAADr9Rv/"
	scriptsBytes, err := base64.StdEncoding.DecodeString(script)
	assert.NoError(t, err, "failed to set decode base64 script")

	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.Ride4DApps))
	// to.stor.activateFeature(t, int16(settings.RideV5))

	setScript(t, to, testGlobal.recipientInfo.addr, testGlobal.recipientInfo.pk, scriptsBytes)

	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	functionCall := proto.NewFunctionCall("call", nil)
	invokeFee = FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	feeAsset = proto.NewOptionalAssetWaves()

	tx := proto.NewUnsignedInvokeScriptWithProofs(1, testGlobal.senderInfo.pk,
		proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), functionCall,
		[]proto.ScriptPayment{}, feeAsset, invokeFee, defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign invoke script tx")

	co := createCheckerCustomTestObjects(t, to)
	co.stor = to.stor
	checkerData, err := co.tc.checkInvokeScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check invoke script tx")

	ch, err := to.td.createDiffInvokeScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffInvokeScriptWithProofs() failed")

	actions := []proto.ScriptAction{
		&proto.DataEntryScriptAction{
			Entry:  &proto.BooleanDataEntry{Key: "bool", Value: true},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.IntegerDataEntry{Key: "int", Value: 1},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.StringDataEntry{Key: "int", Value: ""},
			Sender: &testGlobal.recipientInfo.pk},
	}

	invocationResult := &invocationResult{actions: actions}

	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performInvokeScriptWithProofs(tx, defaultPerformerInfoWithChecker(checkerData),
		invocationResult, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform invoke script tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 200000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299500000,
		},
		&proto.DataEntriesSnapshot{
			Address: testGlobal.recipientInfo.addr,
			DataEntries: []proto.DataEntry{
				&proto.BooleanDataEntry{Key: "bool", Value: true},
				&proto.StringDataEntry{Key: "int", Value: ""}, // IntegerEntry("int", 1) - will be overwritten
			},
		},
		&InternalDAppComplexitySnapshot{
			ScriptAddress: testGlobal.recipientInfo.addr,
			Estimation: ride.TreeEstimation{
				Estimation: 16,
				Verifier:   0,
				Functions:  map[string]int{"call": 16},
			},
			Update: true,
		},
	}

	var expectedDataEntrySnapshot proto.DataEntriesSnapshot
	idxExpectedDataSnapshot := 0
	for idx, atomicSnapshot := range expectedSnapshot {
		if dataEntryS, ok := atomicSnapshot.(*proto.DataEntriesSnapshot); ok {
			idxExpectedDataSnapshot = idx
			expectedDataEntrySnapshot = *dataEntryS
		}
	}
	var transactionDataEntrySnapshot proto.DataEntriesSnapshot
	idxDataSnapshot := 0
	for idx, atomicSnapshot := range transactionSnapshot {
		if dataEntryS, ok := atomicSnapshot.(*proto.DataEntriesSnapshot); ok {
			idxDataSnapshot = idx
			transactionDataEntrySnapshot = *dataEntryS
		}
	}
	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedDataEntrySnapshot.DataEntries, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedDataEntrySnapshot.DataEntries[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedDataEntrySnapshot.DataEntries[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionDataEntrySnapshot.DataEntries, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionDataEntrySnapshot.DataEntries[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionDataEntrySnapshot.DataEntries[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedDataEntrySnapshot, transactionDataEntrySnapshot)

	expectedSnapshot = remove(expectedSnapshot, idxExpectedDataSnapshot)
	transactionSnapshot = remove(transactionSnapshot, idxDataSnapshot)

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultInvokeScriptRepeatableActionsSnapshot(t *testing.T) {
	/*
			{-# STDLIB_VERSION 5 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}

			@Callable(i)
			func call() = {
			  [
				BooleanEntry("bool", true),
				IntegerEntry("int", 1),
		        IntegerEntry("int", 1),
		        IntegerEntry("int2", 2),
				StringEntry("str", "1"),
		        StringEntry("str", "1"),
		        StringEntry("str2", "2")
			  ]
			}
	*/
	script := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2Fsb" +
		"AAAAAAJAARMAAAAAgkBAAAADEJvb2xlYW5FbnRyeQAAAAICAAAABGJvb2wGCQAE" +
		"TAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQAAAAAAAAAAAEJAARMAAAA" +
		"AgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2ludAAAAAAAAAAAAQkABEwAAAACCQEAAA" +
		"AMSW50ZWdlckVudHJ5AAAAAgIAAAAEaW50MgAAAAAAAAAAAgkABEwAAAACCQEAAAALU3RyaW5nRW" +
		"50cnkAAAACAgAAAANzdHICAAAAATEJAARMAAAAAgkBAAAAC1N0cmluZ0VudHJ5AAAAAgIAAAADc3RyAgAAAA" +
		"ExCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAICAAAABHN0cjICAAAAATIFAAAAA25pbAAAAACkN9Gf"
	scriptsBytes, err := base64.StdEncoding.DecodeString(script)
	assert.NoError(t, err, "failed to set decode base64 script")

	checkerInfo := customCheckerInfo()
	to := createDifferTestObjects(t, checkerInfo)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.Ride4DApps))
	// to.stor.activateFeature(t, int16(settings.RideV5))

	setScript(t, to, testGlobal.recipientInfo.addr, testGlobal.recipientInfo.pk, scriptsBytes)

	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	functionCall := proto.NewFunctionCall("call", nil)
	invokeFee = FeeUnit * feeConstants[proto.InvokeScriptTransaction]
	feeAsset = proto.NewOptionalAssetWaves()

	tx := proto.NewUnsignedInvokeScriptWithProofs(1, testGlobal.senderInfo.pk,
		proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), functionCall,
		[]proto.ScriptPayment{}, feeAsset, invokeFee, defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign invoke script tx")

	co := createCheckerCustomTestObjects(t, to)
	co.stor = to.stor
	checkerData, err := co.tc.checkInvokeScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check invoke script tx")

	ch, err := to.td.createDiffInvokeScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffInvokeScriptWithProofs() failed")

	actions := []proto.ScriptAction{
		&proto.DataEntryScriptAction{
			Entry:  &proto.BooleanDataEntry{Key: "bool", Value: true},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.IntegerDataEntry{Key: "int", Value: 1},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.IntegerDataEntry{Key: "int", Value: 1},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.IntegerDataEntry{Key: "int2", Value: 2},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.StringDataEntry{Key: "str", Value: "1"},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.StringDataEntry{Key: "str", Value: "1"},
			Sender: &testGlobal.recipientInfo.pk},
		&proto.DataEntryScriptAction{
			Entry:  &proto.StringDataEntry{Key: "str2", Value: "2"},
			Sender: &testGlobal.recipientInfo.pk},
	}

	invocationResult := &invocationResult{actions: actions}

	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performInvokeScriptWithProofs(tx, defaultPerformerInfoWithChecker(checkerData),
		invocationResult, applicationRes.changes.diff)
	assert.NoError(t, err, "failed to perform invoke script tx")

	expectedSnapshot := proto.TransactionSnapshot{
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 200000,
		},
		&proto.WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299500000,
		},
		// The order is not deterministic.
		&proto.DataEntriesSnapshot{
			Address: testGlobal.recipientInfo.addr,
			DataEntries: []proto.DataEntry{
				&proto.BooleanDataEntry{Key: "bool", Value: true},
				&proto.IntegerDataEntry{Key: "int", Value: 1},
				&proto.IntegerDataEntry{Key: "int2", Value: 2},
				&proto.StringDataEntry{Key: "str", Value: "1"},
				&proto.StringDataEntry{Key: "str2", Value: "2"},
			},
		},
		&InternalDAppComplexitySnapshot{
			ScriptAddress: testGlobal.recipientInfo.addr,
			Estimation: ride.TreeEstimation{
				Estimation: 36,
				Verifier:   0,
				Functions:  map[string]int{"call": 36},
			},
			Update: true,
		},
	}

	var expectedDataEntrySnapshot proto.DataEntriesSnapshot
	idxExpectedDataSnapshot := 0
	for idx, atomicSnapshot := range expectedSnapshot {
		if dataEntryS, ok := atomicSnapshot.(*proto.DataEntriesSnapshot); ok {
			idxExpectedDataSnapshot = idx
			expectedDataEntrySnapshot = *dataEntryS
		}
	}
	var transactionDataEntrySnapshot proto.DataEntriesSnapshot
	idxDataSnapshot := 0
	for idx, atomicSnapshot := range transactionSnapshot {
		if dataEntryS, ok := atomicSnapshot.(*proto.DataEntriesSnapshot); ok {
			idxDataSnapshot = idx
			transactionDataEntrySnapshot = *dataEntryS
		}
	}
	var snapshotI []byte
	var snapshotJ []byte
	sort.Slice(expectedDataEntrySnapshot.DataEntries, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedDataEntrySnapshot.DataEntries[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedDataEntrySnapshot.DataEntries[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionDataEntrySnapshot.DataEntries, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionDataEntrySnapshot.DataEntries[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionDataEntrySnapshot.DataEntries[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedDataEntrySnapshot, transactionDataEntrySnapshot)

	expectedSnapshot = remove(expectedSnapshot, idxExpectedDataSnapshot)
	transactionSnapshot = remove(transactionSnapshot, idxDataSnapshot)

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err = json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err = json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func remove(slice []proto.AtomicSnapshot, s int) []proto.AtomicSnapshot {
	return append(slice[:s], slice[s+1:]...)
}

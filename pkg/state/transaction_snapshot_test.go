package state

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"math/big"
	"sort"
	"testing"
)

func defaultAssetInfoTransfer(tail [12]byte, reissuable bool, amount int64, issuer crypto.PublicKey, name string) *assetInfo {
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

func TestDefaultTransferWavesAndAssetSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	err := to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedTransferWithSig(testGlobal.issuerInfo.pk, proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), defaultTimestamp, defaultAmount*1000*2, uint64(FeeUnit), testGlobal.recipientInfo.Recipient(), nil)
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign transfer tx")

	ch, err := to.td.createDiffTransferWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffTransferWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performTransferWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform transfer tx")
	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299700000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			Balance: 200000,
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

// TODO send only txBalanceChanges to perfomer
func TestDefaultIssueTransactionSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	tx := proto.NewUnsignedIssueWithSig(testGlobal.issuerInfo.pk, "asset0", "description", defaultQuantity, defaultDecimals, true, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign issue tx")

	ch, err := to.td.createDiffIssueWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffIssueWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performIssueWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform issue tx")

	expectedSnapshot := TransactionSnapshot{
		&StaticAssetInfoSnapshot{
			AssetID:             *tx.ID,
			SourceTransactionID: *tx.ID,
			IssuerPublicKey:     testGlobal.issuerInfo.pk,
			Decimals:            defaultDecimals,
			IsNFT:               false},
		&AssetDescriptionSnapshot{
			AssetID:          *tx.ID,
			AssetName:        "asset0",
			AssetDescription: "description",
			ChangeHeight:     1,
		},
		&AssetVolumeSnapshot{
			AssetID:       *tx.ID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity)),
			IsReissuable:  true,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: *tx.ID,
			Balance: 1000,
		},
	}
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultReissueSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID), true, 1000, testGlobal.issuerInfo.pk, "asset0"), blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.issuerInfo.addr.ID(), proto.AssetIDFromDigest(testGlobal.asset0.assetID), 1000, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedReissueWithSig(testGlobal.issuerInfo.pk, testGlobal.asset0.assetID, 50, false, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign reissue tx")

	ch, err := to.td.createDiffReissueWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffReissueWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performReissueWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform reissue tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 1050,
		},
		&AssetVolumeSnapshot{
			AssetID:       testGlobal.asset0.assetID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity + 50)),
			IsReissuable:  false,
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultBurnSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID), true, 1000, testGlobal.issuerInfo.pk, "asset0"), blockID0)

	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.issuerInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.issuerInfo.addr.ID(), proto.AssetIDFromDigest(testGlobal.asset0.assetID), 1000, blockID0)
	assert.NoError(t, err, "failed to set asset balance")

	tx := proto.NewUnsignedBurnWithSig(testGlobal.issuerInfo.pk, testGlobal.asset0.assetID, 50, defaultTimestamp, uint64(1*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.issuerInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffBurnWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performBurnWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			Balance: 299900000,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 950,
		},
		&AssetVolumeSnapshot{
			AssetID:       testGlobal.asset0.assetID,
			TotalQuantity: *big.NewInt(int64(defaultQuantity - 50)),
			IsReissuable:  true,
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultExchangeTransaction(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	// issue assets
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID), true, 1000, testGlobal.senderInfo.pk, "asset0"), blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset1.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset1.assetID), true, 1000, testGlobal.recipientInfo.pk, "asset1"), blockID0)
	assert.NoError(t, err, "failed to issue asset")

	// set waves balance for the seller and the buyer
	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 2000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	// set waves balance for the matcher account
	err = to.stor.entities.balances.setWavesBalance(testGlobal.matcherInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 3000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	// set asset balance for the seller and the buyer
	err = to.stor.entities.balances.setAssetBalance(testGlobal.senderInfo.addr.ID(), proto.AssetIDFromDigest(testGlobal.asset1.assetID), 500, blockID0)
	assert.NoError(t, err, "failed to set asset balance")
	err = to.stor.entities.balances.setAssetBalance(testGlobal.recipientInfo.addr.ID(), proto.AssetIDFromDigest(testGlobal.asset0.assetID), 600, blockID0)
	assert.NoError(t, err, "failed to set asset balance")

	bo := proto.NewUnsignedOrderV1(testGlobal.senderInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Buy, 10e8, 10, 0, 0, 3)
	err = bo.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "bo.Sign() failed")
	so := proto.NewUnsignedOrderV1(testGlobal.recipientInfo.pk, testGlobal.matcherInfo.pk, *testGlobal.asset0.asset, *testGlobal.asset1.asset, proto.Sell, 10e8, 10, 0, 0, 3)
	err = so.Sign(proto.TestNetScheme, testGlobal.recipientInfo.sk)
	assert.NoError(t, err, "so.Sign() failed")
	tx := proto.NewUnsignedExchangeWithSig(bo, so, bo.Price, bo.Amount, 1, 2, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.matcherInfo.sk)

	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffExchange(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performExchange(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299999999,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			Balance: 599999998,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.matcherInfo.addr,
			Balance: 899900003,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 10,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			AssetID: testGlobal.asset0.assetID,
			Balance: 590,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: testGlobal.asset1.assetID,
			Balance: 400,
		},
		&AssetBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			AssetID: testGlobal.asset1.assetID,
			Balance: 100,
		},
		&FilledVolumeFeeSnapshot{
			OrderID:      *bo.ID,
			FilledVolume: 10,
			FilledFee:    1,
		},
		&FilledVolumeFeeSnapshot{
			OrderID:      *so.ID,
			FilledVolume: 10,
			FilledFee:    2,
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedLeaseWithSig(testGlobal.senderInfo.pk, testGlobal.recipientInfo.Recipient(), 50, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffLeaseWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performLeaseWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&LeaseStateSnapshot{
			LeaseID:             *tx.ID,
			Status:              LeaseActive,
			Amount:              50,
			Sender:              testGlobal.senderInfo.addr,
			Recipient:           testGlobal.recipientInfo.addr,
			OriginTransactionID: *tx.ID,
			Height:              0,
		},
		&LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 50,
		},
		&LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  50,
			LeaseOut: 0,
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseCancelSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	leaseID := testGlobal.asset0.assetID
	leasing := &leasing{
		Sender:              testGlobal.senderInfo.addr,
		Recipient:           testGlobal.recipientInfo.addr,
		Amount:              50,
		Height:              1,
		Status:              LeaseActive,
		RecipientAlias:      testGlobal.recipientInfo.Recipient().Alias(),
		OriginTransactionID: &leaseID,
	}
	err := to.stor.entities.leases.addLeasing(leaseID, leasing, blockID0)
	assert.NoError(t, err, "failed to add leasing")

	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3, leaseIn: 0, leaseOut: 50}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")
	err = to.stor.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3, leaseIn: 50, leaseOut: 0}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedLeaseCancelWithSig(testGlobal.senderInfo.pk, leaseID, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffLeaseCancelWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performLeaseCancelWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&LeaseStateSnapshot{
			LeaseID:             leaseID,
			Status:              LeaseCanceled,
			Amount:              50,
			Sender:              testGlobal.senderInfo.addr,
			Recipient:           testGlobal.recipientInfo.addr,
			OriginTransactionID: leaseID,
			Height:              1,
		},
		&LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
		&LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
	}
	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultCreateAliasSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	alias := proto.NewAlias(proto.TestNetScheme, "aliasForSender")
	tx := proto.NewUnsignedCreateAliasWithSig(testGlobal.senderInfo.pk, *alias, uint64(1*FeeUnit), defaultTimestamp)
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")
	ch, err := to.td.createDiffCreateAliasWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performCreateAliasWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := TransactionSnapshot{
		&WavesBalanceSnapshot{
			Address: testGlobal.minerInfo.addr,
			Balance: 40000,
		},
		&WavesBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			Balance: 299900000,
		},
		&AliasSnapshot{
			Address: testGlobal.senderInfo.addr,
			Alias:   *proto.NewAlias(proto.TestNetScheme, "aliasForSender"),
		},
	}

	sort.Slice(expectedSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(expectedSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(expectedSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	sort.Slice(transactionSnapshot, func(i, j int) bool {
		snapshotI, err := json.Marshal(transactionSnapshot[i])
		assert.NoError(t, err, "failed to marshal snapshots")
		snapshotJ, err := json.Marshal(transactionSnapshot[j])
		assert.NoError(t, err, "failed to marshal snapshots")
		return string(snapshotI) < string(snapshotJ)
	})

	assert.Equal(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

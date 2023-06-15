package state

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"math/big"
	"sort"
	"testing"
)

func defaultAssetInfoTransfer(tail [12]byte, reissuable bool, amount int64) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:     tail,
			issuer:   testGlobal.issuerInfo.pk,
			decimals: 2,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(amount),
			name:                     "asset",
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
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID), true, 1000), blockID0)
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
	err := to.stor.entities.assets.issueAsset(proto.AssetIDFromDigest(testGlobal.asset0.assetID), defaultAssetInfoTransfer(proto.DigestTail(testGlobal.asset0.assetID), true, 1000), blockID0)

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

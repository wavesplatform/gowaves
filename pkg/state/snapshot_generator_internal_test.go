package state

import (
	"encoding/base64"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type snapshotGeneratorTestObjects struct {
	stor *testStorageObjects
	tc   *transactionChecker
	tp   transactionPerformer
	td   *transactionDiffer
}

func createSnapshotGeneratorTestObjects(t *testing.T) *snapshotGeneratorTestObjects {
	stor := createStorageObjects(t, true)
	sg := newSnapshotGenerator(stor.entities, stor.settings.AddressSchemeCharacter)
	genID := proto.NewBlockIDFromSignature(genSig)
	tc, err := newTransactionChecker(genID, stor.entities, stor.settings)
	require.NoError(t, err)
	td, err := newTransactionDiffer(stor.entities, stor.settings)
	require.NoError(t, err)
	return &snapshotGeneratorTestObjects{stor, tc, sg, td}
}

func defaultAssetInfoTransfer(tail [12]byte, reissuable bool,
	amount int64, issuer crypto.PublicKey,
	name string) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:     tail,
			Issuer:   issuer,
			Decimals: 2,
			IsNFT:    false,
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
	return &performerInfo{0, blockID0, proto.WavesAddress{}, checkerData}
}

func customCheckerInfo() *checkerInfo {
	defaultBlockInfo := defaultBlockInfo()
	return &checkerInfo{
		currentTimestamp: defaultBlockInfo.Timestamp,
		parentTimestamp:  defaultTimestamp - settings.MustMainNetSettings().MaxTxTimeBackOffset/2,
		blockID:          blockID0,
		blockVersion:     defaultBlockInfo.Version,
		blockchainHeight: defaultBlockInfo.Height,
	}
}

func txSnapshotsEqual(t *testing.T, expected, actual txSnapshot) {
	_ = assert.ElementsMatch(t, expected.regular, actual.regular)
	_ = assert.ElementsMatch(t, expected.internal, actual.internal)
}

func TestDefaultTransferWavesAndAssetSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performTransferWithSig(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform transfer tx")
	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

// TODO send only txBalanceChanges to performer

func TestDefaultIssueTransactionSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performIssueWithSig(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform issue tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.NewAssetSnapshot{
				AssetID:         *tx.ID,
				IssuerPublicKey: testGlobal.issuerInfo.pk,
				Decimals:        defaultDecimals,
				IsNFT:           false},
			&proto.AssetDescriptionSnapshot{
				AssetID:          *tx.ID,
				AssetName:        "asset0",
				AssetDescription: "description",
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultReissueSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performReissueWithSig(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform reissue tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultBurnSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performBurnWithSig(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultExchangeTransaction(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performExchange(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	pi := defaultPerformerInfo()
	transactionSnapshot, err := to.tp.performLeaseWithSig(tx, pi, applicationRes.changes.diff.balancesChanges())
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: 40000,
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 299900000,
			},
			&proto.NewLeaseSnapshot{
				LeaseID:       *tx.ID,
				Amount:        50,
				SenderPK:      testGlobal.senderInfo.pk,
				RecipientAddr: testGlobal.recipientInfo.addr,
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
		},
		internal: []internalSnapshot{
			&InternalNewLeaseInfoSnapshot{
				LeaseID:             *tx.ID,
				OriginHeight:        pi.blockHeight(),
				OriginTransactionID: tx.ID,
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultLeaseCancelSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))

	leaseID := testGlobal.asset0.assetID
	leasing := &leasing{
		SenderPK:            testGlobal.senderInfo.pk,
		RecipientAddr:       testGlobal.recipientInfo.addr,
		Amount:              50,
		OriginHeight:        1,
		Status:              LeaseActive,
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
	pi := defaultPerformerInfo()
	transactionSnapshot, err := to.tp.performLeaseCancelWithSig(tx, pi, applicationRes.changes.diff.balancesChanges())
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: 40000,
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 299900000,
			},
			&proto.CancelledLeaseSnapshot{
				LeaseID: leaseID,
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
		},
		internal: []internalSnapshot{
			&InternalCancelledLeaseInfoSnapshot{
				LeaseID:             leaseID,
				CancelHeight:        pi.blockHeight(),
				CancelTransactionID: tx.ID,
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultCreateAliasSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performCreateAliasWithSig(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
				Alias:   "aliasForSender",
			},
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultDataSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performDataWithProofs(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSponsorshipSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
	transactionSnapshot, err := to.tp.performSponsorshipWithProofs(
		tx,
		defaultPerformerInfo(),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: nil,
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetDappScriptSnapshot(t *testing.T) {
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
	to := createSnapshotGeneratorTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.RideV5))
	err = to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSetScriptWithProofs(1, testGlobal.senderInfo.pk,
		scriptsBytes, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign set script tx")

	co := createCheckerTestObjectsWithStor(t, checkerInfo, to.stor)
	co.stor = to.stor
	checkerData, err := co.tc.checkSetScriptWithProofs(tx, checkerInfo)
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
				Script:             scriptsBytes,
				VerifierComplexity: 0,
			},
		},
		internal: []internalSnapshot{
			&InternalDAppComplexitySnapshot{
				ScriptAddress: testGlobal.senderInfo.addr,
				Estimation:    ride.TreeEstimation{Estimation: 36, Verifier: 0, Functions: map[string]int{"call": 36}},
				ScriptIsEmpty: false,
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetScriptSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSetScriptWithProofs(1, testGlobal.senderInfo.pk,
		testGlobal.scriptBytes, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign set script tx")

	checkerData, err := to.tc.checkSetScriptWithProofs(tx, customCheckerInfo())
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: []internalSnapshot{
			&InternalDAppComplexitySnapshot{
				ScriptAddress: testGlobal.senderInfo.addr,
				Estimation:    ride.TreeEstimation{Estimation: 340, Verifier: 340},
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetEmptyScriptSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(),
		wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to set waves balance")

	tx := proto.NewUnsignedSetScriptWithProofs(1, testGlobal.senderInfo.pk,
		nil, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign set script tx")

	checkerData, err := to.tc.checkSetScriptWithProofs(tx, customCheckerInfo())
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
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
		},
		internal: []internalSnapshot{
			&InternalDAppComplexitySnapshot{
				ScriptAddress: testGlobal.senderInfo.addr,
				Estimation:    ride.TreeEstimation{Estimation: 0, Verifier: 0},
				ScriptIsEmpty: true,
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultSetAssetScriptSnapshot(t *testing.T) {
	to := createSnapshotGeneratorTestObjects(t)

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
		testGlobal.scriptBytes, blockID0)
	assert.NoError(t, err, "failed to issue asset")

	tx := proto.NewUnsignedSetAssetScriptWithProofs(1, testGlobal.senderInfo.pk,
		testGlobal.asset0.assetID, testGlobal.scriptBytes, uint64(1*FeeUnit), defaultTimestamp)

	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign burn tx")

	checkerData, err := to.tc.checkSetAssetScriptWithProofs(tx, customCheckerInfo())
	assert.NoError(t, err, "failed to check set script tx")

	ch, err := to.td.createDiffSetAssetScriptWithProofs(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffBurnWithSig() failed")
	applicationRes := &applicationResult{changes: ch, checkerData: txCheckerData{}}
	transactionSnapshot, err := to.tp.performSetAssetScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform burn tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: 40000,
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 299900000,
			},
			&proto.AssetScriptSnapshot{
				AssetID: testGlobal.asset0.assetID,
				Script:  testGlobal.scriptBytes,
			},
		},
		internal: []internalSnapshot{
			&InternalAssetScriptComplexitySnapshot{
				AssetID: testGlobal.asset0.assetID,
				Estimation: ride.TreeEstimation{
					Estimation: 340,
					Verifier:   340,
					Functions:  nil,
				},
				ScriptIsEmpty: false,
			},
		},
	}

	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	to.stor.flush(t)
}

func TestDefaultInvokeScriptSnapshot(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	info := to.fallibleValidationParams(t)
	to.setDApp(t, "default_dapp_snapshots.base64", testGlobal.recipientInfo)
	amount := uint64(1000)
	startBalance := amount + 1

	wavesBalSender := wavesValue{
		profile: balanceProfile{
			balance: startBalance + invokeFee,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	wavesBalMiner := wavesValue{
		profile: balanceProfile{
			balance: startBalance,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	err := to.state.stor.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), wavesBalSender, blockID0)
	assert.NoError(t, err)
	err = to.state.stor.balances.setWavesBalance(testGlobal.minerInfo.addr.ID(), wavesBalMiner, blockID0)
	assert.NoError(t, err)

	// activate ReducedNFTFee feature for NFT flag = true
	// though because asset issued as reissuable [Issue("Asset", "", 1, 0, true, unit, 0)] it can't be NFT anyway
	// so NFT flag will be false
	// With [Reissue(assetId, 1, false)] asset will be non-reissuable, but it's still not NFT, because asset has been
	// issued as reissuable
	to.activateFeature(t, int16(settings.ReducedNFTFee))

	snapshotApplierInfo := newBlockSnapshotsApplierInfo(info.checkerInfo, to.state.settings.AddressSchemeCharacter)
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(snapshotApplierInfo)
	defer cleanup()

	fc := proto.NewFunctionCall("call", []proto.Argument{})
	testData := invokeApplierTestData{

		payments: []proto.ScriptPayment{},
		fc:       fc,
		info:     info,
	}

	tx := createInvokeScriptWithProofs(t, testData.payments, testData.fc, feeAsset, invokeFee)
	assert.NoError(t, err, "failed to sign invoke script tx")

	_, applicationRes := to.applyAndSaveInvoke(t, tx, testData.info, false)

	transactionSnapshot, err := to.state.appender.txHandler.tp.performInvokeScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(applicationRes.checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform invoke script tx")

	var dataEntrySnapshot *proto.DataEntriesSnapshot
	var dataEntrySnapshoIdx int
	var assetID crypto.Digest
	for i, snap := range transactionSnapshot.regular {
		if assetScriptSnapshot, ok := snap.(*proto.NewAssetSnapshot); ok {
			assetID = assetScriptSnapshot.AssetID
		}
		if dataEntrySnap, ok := snap.(*proto.DataEntriesSnapshot); ok {
			dataEntrySnapshot = dataEntrySnap
			dataEntrySnapshoIdx = i
		}
	}
	transactionSnapshot.regular = remove(transactionSnapshot.regular, dataEntrySnapshoIdx)

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: startBalance + calculateCurrentBlockTxFee(invokeFee, true), // because ng is activated
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 1001,
			},
			&proto.AssetBalanceSnapshot{
				Address: testGlobal.recipientInfo.addr,
				AssetID: assetID,
				Balance: 1,
			},
			&proto.AssetDescriptionSnapshot{
				AssetID:          assetID,
				AssetName:        "Asset",
				AssetDescription: "",
			},
			&proto.AssetVolumeSnapshot{
				AssetID:       assetID,
				TotalQuantity: *big.NewInt(1),
				IsReissuable:  false,
			},
			&proto.NewAssetSnapshot{
				AssetID:         assetID,
				IssuerPublicKey: testGlobal.recipientInfo.pk,
				Decimals:        0,
				IsNFT:           false, // see comment above
			},
		},
		internal: nil,
	}
	expectedDataEntry := &proto.DataEntriesSnapshot{
		Address: testGlobal.recipientInfo.addr,
		DataEntries: []proto.DataEntry{
			&proto.BinaryDataEntry{Key: "bin", Value: []byte{}},
			&proto.BooleanDataEntry{Key: "bool", Value: true},
			&proto.IntegerDataEntry{Key: "int", Value: 1},
			// &proto.StringDataEntry{Key: "int", Value: ""}, // This entry will be overwritten by delete data entry
			&proto.DeleteDataEntry{Key: "str"},
		},
	}
	assert.Equal(t, expectedDataEntry.Address, dataEntrySnapshot.Address)
	assert.ElementsMatch(t, expectedDataEntry.DataEntries, dataEntrySnapshot.DataEntries)
	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	flushErr := to.state.stor.flush()
	assert.NoError(t, flushErr)
}

// Check if the snapshot generator doesn't generate
// extra asset description and static asset info snapshots after reissue and burn.
func TestNoExtraStaticAssetInfoSnapshot(t *testing.T) {
	to := createInvokeApplierTestObjects(t)
	info := to.fallibleValidationParams(t)
	to.setDApp(t, "issue_reissue_dapp_snapshots.base64", testGlobal.recipientInfo)
	amount := uint64(1000)
	startBalance := amount + 1

	wavesBalSender := wavesValue{
		profile: balanceProfile{
			balance: startBalance + invokeFee,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	wavesBalMiner := wavesValue{
		profile: balanceProfile{
			balance: startBalance,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	err := to.state.stor.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), wavesBalSender, blockID0)
	assert.NoError(t, err)
	err = to.state.stor.balances.setWavesBalance(testGlobal.minerInfo.addr.ID(), wavesBalMiner, blockID0)
	assert.NoError(t, err)

	var asset crypto.Digest
	err = asset.UnmarshalBinary([]byte("GAzAEjApmjMYZKPzri2g2VUXNvTiQGF7"))
	assert.NoError(t, err)
	assetID := proto.AssetIDFromDigest(asset)
	err = to.state.stor.assets.issueAsset(assetID, &assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:                 proto.DigestTail(asset),
			Issuer:               testGlobal.recipientInfo.pk,
			Decimals:             0,
			IssueHeight:          0,
			IsNFT:                false,
			IssueSequenceInBlock: 1,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(10),
			name:                     "asset",
			description:              "",
			lastNameDescChangeHeight: 0,
			reissuable:               true,
		},
	}, blockID0)
	assert.NoError(t, err)

	snapshotApplierInfo := newBlockSnapshotsApplierInfo(info.checkerInfo, to.state.settings.AddressSchemeCharacter)
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(snapshotApplierInfo)
	defer cleanup()

	fc := proto.NewFunctionCall("call", []proto.Argument{})
	testData := invokeApplierTestData{

		payments: []proto.ScriptPayment{},
		fc:       fc,
		info:     info,
	}

	tx := createInvokeScriptWithProofs(t, testData.payments, testData.fc, feeAsset, invokeFee)
	assert.NoError(t, err, "failed to sign invoke script tx")

	_, applicationRes := to.applyAndSaveInvoke(t, tx, testData.info, false)

	transactionSnapshot, err := to.state.appender.txHandler.tp.performInvokeScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(applicationRes.checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform invoke script tx")

	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: startBalance + calculateCurrentBlockTxFee(invokeFee, true), // because ng is activated
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 1001,
			},
			&proto.AssetBalanceSnapshot{
				Address: testGlobal.recipientInfo.addr,
				AssetID: asset,
				Balance: 4,
			},
			&proto.AssetVolumeSnapshot{
				AssetID:       asset,
				TotalQuantity: *big.NewInt(14),
				IsReissuable:  false,
			},
		},
		internal: nil,
	}
	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)
	flushErr := to.state.stor.flush()
	assert.NoError(t, flushErr)
}

func TestLeaseAndLeaseCancelInTheSameInvokeTx(t *testing.T) {
	const (
		script = `
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		
		let addr = Address(base58'3PD8uesEwWoKu63ujwbJXeJdk7jygdimJST')
		
		@Callable(i)
		func call() = {
			let lease = Lease(addr, 1000000)
			let leaseID = calculateLeaseId(lease)
			[lease, LeaseCancel(leaseID)]
		}`
		leaseAmount        = 1000000
		leaseRecipientAddr = "3PD8uesEwWoKu63ujwbJXeJdk7jygdimJST"
	)
	scriptBytes, errs := compiler.Compile(script, false, true)
	require.NoError(t, errors.Join(errs...))

	to := createInvokeApplierTestObjects(t)
	info := to.fallibleValidationParams(t)

	dAppInfo := testGlobal.recipientInfo
	to.setScript(t, testGlobal.recipientInfo.addr, dAppInfo.pk, scriptBytes)

	amount := uint64(1000)
	startBalance := amount + 1

	wavesBalSender := wavesValue{
		profile: balanceProfile{
			balance: startBalance + invokeFee,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	wavesBalMiner := wavesValue{
		profile: balanceProfile{
			balance: startBalance,
		},
		leaseChange:   false,
		balanceChange: false,
	}
	err := to.state.stor.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), wavesBalSender, blockID0)
	assert.NoError(t, err)
	err = to.state.stor.balances.setWavesBalance(testGlobal.minerInfo.addr.ID(), wavesBalMiner, blockID0)
	assert.NoError(t, err)

	snapshotApplierInfo := newBlockSnapshotsApplierInfo(info.checkerInfo, to.state.settings.AddressSchemeCharacter)
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(snapshotApplierInfo)
	defer cleanup()

	testData := invokeApplierTestData{
		payments: []proto.ScriptPayment{},
		fc:       proto.NewFunctionCall("call", []proto.Argument{}),
		info:     info,
	}

	tx := createInvokeScriptWithProofs(t, testData.payments, testData.fc, feeAsset, invokeFee)
	assert.NoError(t, err, "failed to sign invoke script tx")

	_, applicationRes := to.applyAndSaveInvoke(t, tx, testData.info, false)

	transactionSnapshot, err := to.state.appender.txHandler.tp.performInvokeScriptWithProofs(
		tx,
		defaultPerformerInfoWithChecker(applicationRes.checkerData),
		applicationRes.changes.diff.balancesChanges(),
	)
	assert.NoError(t, err, "failed to perform invoke script tx")

	lRcpAddr := proto.MustAddressFromString(leaseRecipientAddr)
	lID := proto.GenerateLeaseScriptActionID(proto.NewRecipientFromAddress(lRcpAddr), leaseAmount, 0, *tx.ID)
	expectedSnapshot := txSnapshot{
		regular: []proto.AtomicSnapshot{
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.minerInfo.addr,
				Balance: startBalance + calculateCurrentBlockTxFee(invokeFee, true), // because ng is activated
			},
			&proto.WavesBalanceSnapshot{
				Address: testGlobal.senderInfo.addr,
				Balance: 1001,
			},
			&proto.NewLeaseSnapshot{
				LeaseID:       lID,
				Amount:        leaseAmount,
				SenderPK:      dAppInfo.pk,
				RecipientAddr: lRcpAddr,
			},
			&proto.CancelledLeaseSnapshot{LeaseID: lID},
			&proto.LeaseBalanceSnapshot{LeaseIn: 0, LeaseOut: 0, Address: lRcpAddr},
			&proto.LeaseBalanceSnapshot{LeaseIn: 0, LeaseOut: 0, Address: dAppInfo.addr},
		},
		internal: []internalSnapshot{
			&InternalNewLeaseInfoSnapshot{
				LeaseID:             lID,
				OriginHeight:        info.blockInfo.Height,
				OriginTransactionID: tx.ID,
			},
			&InternalCancelledLeaseInfoSnapshot{
				LeaseID:             lID,
				CancelHeight:        info.blockInfo.Height,
				CancelTransactionID: tx.ID,
			},
		},
	}
	txSnapshotsEqual(t, expectedSnapshot, transactionSnapshot)

	flushErr := to.state.stor.flush()
	assert.NoError(t, flushErr)
}

func remove(slice []proto.AtomicSnapshot, s int) []proto.AtomicSnapshot {
	return append(slice[:s], slice[s+1:]...)
}

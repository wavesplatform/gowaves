package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// sponsorshipChangeTestObjects is the state with the asset issued and sponsored by the dApp, which changes
// the sponsorship of the asset by the invoke paying the fee in the same asset.
type sponsorshipChangeTestObjects struct {
	*invokeApplierTestObjects
	asset       crypto.Digest
	dApp        *testWavesAddrData
	sender      *testWavesAddrData
	miner       *testWavesAddrData
	startWaves  uint64
	startAssets uint64
}

const (
	sponsorshipChangeOldRate = 1000 // the asset units per 0.001 Waves before the invoke
	sponsorshipChangeNewRate = 10   // the asset units per 0.001 Waves set by the invoke
	sponsorshipChangeFee     = 5000 // the fee in the asset units for both transactions
)

func createSponsorshipChangeTestObjects(t *testing.T) *sponsorshipChangeTestObjects {
	const script = `
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func sponsor(asset: ByteVector, rate: Int) = [SponsorFee(asset, rate)]`
	scriptBytes, errs := compiler.Compile(script, false, true)
	require.NoError(t, errors.Join(errs...))

	to := &sponsorshipChangeTestObjects{
		invokeApplierTestObjects: createInvokeApplierTestObjects(t),
		asset:                    testGlobal.asset0.assetID,
		dApp:                     testGlobal.recipientInfo,
		sender:                   testGlobal.senderInfo,
		miner:                    testGlobal.minerInfo,
		startWaves:               100 * proto.PriceConstant,
		startAssets:              100 * sponsorshipChangeFee,
	}
	// Sponsorship takes effect right after the activation of the feature, without the second activation window.
	to.state.stor.sponsoredAssets.settings.SponsorshipSingleActivationPeriod = true
	for _, f := range []settings.Feature{
		settings.FeeSponsorship, settings.BlockV5, settings.RideV5, settings.SmartAccountTrading,
	} {
		to.activateFeature(t, int16(f))
	}
	to.setScript(t, to.dApp.addr, to.dApp.pk, scriptBytes)

	ai := defaultAssetInfo(proto.DigestTail(to.asset), true)
	ai.Issuer = to.dApp.pk
	ai.quantity.SetUint64(to.startAssets)
	stor := to.state.stor
	require.NoError(t, stor.assets.issueAsset(proto.AssetIDFromDigest(to.asset), ai, blockID0))
	require.NoError(t, stor.sponsoredAssets.sponsorAsset(to.asset, sponsorshipChangeOldRate, blockID0))
	require.NoError(t, stor.balances.setAssetBalance(to.sender.addr.ID(), proto.AssetIDFromDigest(to.asset),
		to.startAssets, blockID0))
	for _, addr := range []proto.WavesAddress{to.dApp.addr, to.sender.addr, to.miner.addr} {
		require.NoError(t, stor.balances.setWavesBalance(addr.ID(),
			wavesValue{profile: balanceProfile{Balance: to.startWaves}}, blockID0))
	}
	require.NoError(t, to.state.flush())
	to.state.reset()
	return to
}

func (to *sponsorshipChangeTestObjects) feeAsset() proto.OptionalAsset {
	return *proto.NewOptionalAssetFromDigest(to.asset)
}

// sponsorInvoke returns the invoke, which sets the new rate of the sponsored asset and pays the fee in the same asset.
func (to *sponsorshipChangeTestObjects) sponsorInvoke(t *testing.T) *proto.InvokeScriptWithProofs {
	fc := proto.NewFunctionCall("sponsor", proto.Arguments{
		&proto.BinaryArgument{Value: to.asset.Bytes()},
		proto.NewIntegerArgument(sponsorshipChangeNewRate),
	})
	tx := proto.NewUnsignedInvokeScriptWithProofs(2, to.sender.pk, proto.NewRecipientFromAddress(to.dApp.addr), fc,
		nil, to.feeAsset(), sponsorshipChangeFee, defaultTimestamp)
	require.NoError(t, tx.Sign(proto.MainNetScheme, to.sender.sk))
	return tx
}

// sponsoredTransfer returns the transfer, which pays the fee in the sponsored asset.
func (to *sponsorshipChangeTestObjects) sponsoredTransfer(t *testing.T) *proto.TransferWithProofs {
	waves := proto.NewOptionalAssetWaves()
	tx := proto.NewUnsignedTransferWithProofs(2, to.sender.pk, waves, to.feeAsset(), defaultTimestamp+1, 1,
		sponsorshipChangeFee, proto.NewRecipientFromAddress(to.miner.addr), nil)
	require.NoError(t, tx.Sign(proto.MainNetScheme, to.sender.sk))
	return tx
}

func (to *sponsorshipChangeTestObjects) wavesBalance(t *testing.T, addr proto.WavesAddress) uint64 {
	b, err := to.state.NewestWavesBalance(proto.NewRecipientFromAddress(addr))
	require.NoError(t, err)
	return b
}

func (to *sponsorshipChangeTestObjects) checkMinerFee(t *testing.T, expectedTotal uint64) {
	distr := to.state.appender.blockDiffer.curDistr
	assert.Equal(t, expectedTotal, distr.totalWavesFees, "total fee")
	assert.Equal(t, calculateCurrentBlockTxFee(expectedTotal, true), distr.currentWavesBlockFees,
		"fee of the current block's generator")
	assert.Empty(t, distr.totalFees, "sponsored fee must be converted to Waves")
	assert.Empty(t, distr.currentBlockFees, "sponsored fee must be converted to Waves")
}

func (to *sponsorshipChangeTestObjects) checkRate(t *testing.T, expected uint64) {
	rate, err := to.state.stor.sponsoredAssets.newestAssetCost(proto.AssetIDFromDigest(to.asset))
	require.NoError(t, err)
	assert.Equal(t, expected, rate)
}

func TestAppendTxSponsorshipChangedByInvoke(t *testing.T) {
	to := createSponsorshipChangeTestObjects(t)
	params := defaultAppendTxParams()
	params.checkerInfo.rideV5Activated = true
	params.blockV5Activated = true
	params.rideV5Activated = true
	params.acceptFailed = true
	params.currentMinerPK = to.miner.pk
	params.chans = launchVerifier(t.Context(), 1, proto.MainNetScheme)
	defer func() { require.NoError(t, params.chans.closeAndWait()) }()
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(
		newBlockSnapshotsApplierInfo(params.checkerInfo, proto.MainNetScheme))
	defer cleanup()

	oldRateFee := sponsorshipChangeFee * FeeUnit / uint64(sponsorshipChangeOldRate)
	newRateFee := sponsorshipChangeFee * FeeUnit / uint64(sponsorshipChangeNewRate)

	_, err := to.state.appender.appendTx(to.sponsorInvoke(t), params)
	require.NoError(t, err)
	to.checkRate(t, sponsorshipChangeNewRate)
	// The sponsor is charged and the miner is paid at the rate in effect before the invoke.
	assert.Equal(t, to.startWaves-oldRateFee, to.wavesBalance(t, to.dApp.addr), "sponsor's balance")
	assert.Equal(t, to.startWaves+calculateCurrentBlockTxFee(oldRateFee, true), to.wavesBalance(t, to.miner.addr),
		"miner's balance")
	to.checkMinerFee(t, oldRateFee)

	// The next transaction is charged at the new rate.
	_, err = to.state.appender.appendTx(to.sponsoredTransfer(t), params)
	require.NoError(t, err)
	assert.Equal(t, to.startWaves-oldRateFee-newRateFee, to.wavesBalance(t, to.dApp.addr), "sponsor's balance")
	to.checkMinerFee(t, oldRateFee+newRateFee)
}

func TestApplySnapshotInLightNodeSponsorshipChangedByInvoke(t *testing.T) {
	to := createSponsorshipChangeTestObjects(t)
	checkerInfo := defaultCheckerInfo()
	cleanup := to.state.appender.txHandler.sa.SetApplierInfo(
		newBlockSnapshotsApplierInfo(checkerInfo, proto.MainNetScheme))
	defer cleanup()

	oldRateFee := sponsorshipChangeFee * FeeUnit / uint64(sponsorshipChangeOldRate)
	newRateFee := sponsorshipChangeFee * FeeUnit / uint64(sponsorshipChangeNewRate)

	params := &appendBlockParams{
		transactions: []proto.Transaction{to.sponsorInvoke(t), to.sponsoredTransfer(t)},
		block:        defaultBlock(),
	}
	snapshot := proto.BlockSnapshot{TxSnapshots: [][]proto.AtomicSnapshot{
		{
			&proto.SponsorshipSnapshot{AssetID: to.asset, MinSponsoredFee: sponsorshipChangeNewRate},
			&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
		},
		{
			&proto.TransactionStatusSnapshot{Status: proto.TransactionSucceeded},
		},
	}}
	hasher, err := newTxSnapshotHasherDefault()
	require.NoError(t, err)
	defer hasher.Release()

	_, err = to.state.appender.applySnapshotInLightNode(params, defaultBlockInfo(), snapshot, crypto.Digest{}, hasher)
	require.NoError(t, err)
	to.checkRate(t, sponsorshipChangeNewRate)
	// The fee of the invoke is converted at the rate before the invoke, the fee of the transfer at the new rate.
	to.checkMinerFee(t, oldRateFee+newRateFee)
}

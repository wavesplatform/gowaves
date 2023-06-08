package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type blockDifferTestObjects struct {
	stor        *testStorageObjects
	blockDiffer *blockDiffer
	gsp         consensus.GenerationSignatureProvider
}

func createBlockDiffer(t *testing.T) *blockDifferTestObjects {
	return createBlockDifferWithSettings(t, settings.TestNetSettings)
}

func createBlockDifferWithSettings(t *testing.T, sets *settings.BlockchainSettings) *blockDifferTestObjects {
	stor := createStorageObjectsWithOptions(t, testStorageObjectsOptions{Amend: false, Settings: sets})
	handler, err := newTransactionHandler(sets.Genesis.BlockID(), stor.entities, sets)
	require.NoError(t, err, "newTransactionHandler() failed")
	blockDiffer, err := newBlockDiffer(handler, stor.entities, sets)
	require.NoError(t, err, "newBlockDiffer() failed")
	return &blockDifferTestObjects{stor, blockDiffer, consensus.NXTGenerationSignatureProvider}
}

func genBlocks(t *testing.T, to *blockDifferTestObjects) (*proto.Block, *proto.Block) {
	// Create and sign parent block.
	txs := proto.Transactions{createTransferWithSig(t)}
	randSig := genRandBlockIds(t, 1)[0]
	genSig := crypto.MustBytesFromBase58(defaultGenSig)
	parent, err := proto.CreateBlock(txs, 1565694219644, randSig, testGlobal.matcherInfo.pk, proto.NxtConsensus{BaseTarget: 65, GenSignature: genSig}, proto.NgBlockVersion, nil, -1, proto.TestNetScheme)
	require.NoError(t, err, "CreateBlock() failed")
	err = parent.Sign(proto.TestNetScheme, testGlobal.matcherInfo.sk)
	require.NoError(t, err, "Block.Sign() failed")

	// Create and sign child block.
	txs = []proto.Transaction{createIssueWithSig(t, 1000)}
	genSig, err = to.gsp.GenerationSignature(testGlobal.minerInfo.pk, parent.GenSignature[:])
	require.NoError(t, err, "GeneratorSignature() failed")
	child, err := proto.CreateBlock(txs, 1565694219944, parent.BlockID(), testGlobal.minerInfo.pk, proto.NxtConsensus{BaseTarget: 66, GenSignature: genSig}, proto.NgBlockVersion, nil, -1, proto.TestNetScheme)
	require.NoError(t, err, "CreateBlock() failed")
	err = child.Sign(proto.TestNetScheme, testGlobal.minerInfo.sk)
	require.NoError(t, err, "Block.Sign() failed")
	return parent, child
}

func TestCreateBlockDiffWithoutNg(t *testing.T) {
	to := createBlockDiffer(t)

	block, _ := genBlocks(t, to)
	minerDiff, err := to.blockDiffer.createMinerDiff(&block.BlockHeader, true)
	require.NoError(t, err, "createMinerDiff() failed")
	// Empty miner diff before NG activation.
	assert.Equal(t, txDiff{}, minerDiff)
}

func TestCreateBlockDiffNg(t *testing.T) {
	to := createBlockDiffer(t)

	parent, child := genBlocks(t, to)
	// Activate NG first of all.
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.addBlock(t, parent.BlockID())
	to.stor.addBlock(t, child.BlockID())

	// Create diff from parent block.
	txs := parent.Transactions
	for _, tx := range txs {
		err := to.blockDiffer.countMinerFee(tx)
		require.NoError(t, err, "countMinerFee() failed")
	}
	err := to.blockDiffer.saveCurFeeDistr(&parent.BlockHeader)
	require.NoError(t, err, "saveCurFeeDistr() failed")
	parentFeeTotal := int64(txs[0].GetFee())
	parentFeePrevBlock := parentFeeTotal / 5 * 2
	parentFeeNextBlock := parentFeeTotal - parentFeePrevBlock

	// Create diff from child block.
	minerDiff, err := to.blockDiffer.createMinerDiff(&child.BlockHeader, true)
	require.NoError(t, err, "createMinerDiff() failed")
	// Verify child block miner's diff.
	correctMinerAssetBalanceDiff := newBalanceDiff(parentFeeNextBlock, 0, 0, false)
	correctMinerAssetBalanceDiff.blockID = child.BlockID()
	correctMinerDiff := txDiff{
		testGlobal.minerInfo.assetKeys[0]: correctMinerAssetBalanceDiff,
	}
	assert.Equal(t, correctMinerDiff, minerDiff)
}

func TestCreateBlockDiffSponsorship(t *testing.T) {
	to := createBlockDiffer(t)

	parent, child := genBlocks(t, to)
	// Create asset.
	to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	// Activate NG and FeeSponsorship first of all.
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateSponsorship(t)

	// Sponsor asset.
	assetCost := uint64(100500)
	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, parent.BlockID())
	to.stor.addBlock(t, child.BlockID())
	err := to.stor.entities.sponsoredAssets.sponsorAsset(testGlobal.asset0.asset.ID, assetCost, blockID0)
	require.NoError(t, err, "sponsorAsset() failed")

	// Create diff from parent block.
	txs := parent.Transactions
	for _, tx := range txs {
		err = to.blockDiffer.countMinerFee(tx)
		require.NoError(t, err, "countMinerFee() failed")
	}
	err = to.blockDiffer.saveCurFeeDistr(&parent.BlockHeader)
	require.NoError(t, err, "saveCurFeeDistr() failed")
	_, err = to.blockDiffer.createMinerDiff(&parent.BlockHeader, false)
	require.NoError(t, err, "createMinerDiff() failed")
	parentFeeTotal := int64(txs[0].GetFee() * FeeUnit / assetCost)
	parentFeePrevBlock := parentFeeTotal / 5 * 2
	parentFeeNextBlock := parentFeeTotal - parentFeePrevBlock

	// Create diff from child block.
	minerDiff, err := to.blockDiffer.createMinerDiff(&child.BlockHeader, true)
	require.NoError(t, err, "createMinerDiff() failed")
	// Verify child block miner's diff.
	correctMinerWavesBalanceDiff := newBalanceDiff(parentFeeNextBlock, 0, 0, false)
	correctMinerWavesBalanceDiff.blockID = child.BlockID()
	correctMinerDiff := txDiff{
		testGlobal.minerInfo.wavesKey: correctMinerWavesBalanceDiff,
	}
	assert.Equal(t, correctMinerDiff, minerDiff)
}

func genTransferWithWavesFee(t *testing.T) *proto.TransferWithProofs {
	waves := proto.NewOptionalAssetWaves()
	tx := proto.NewUnsignedTransferWithProofs(2, testGlobal.senderInfo.pk, waves, waves, defaultTimestamp, defaultAmount, defaultFee, proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), []byte("attachment"))
	err := tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	require.NoError(t, err)
	return tx
}

func genBlockWithSingleTransaction(t *testing.T, prevID proto.BlockID, prevGenSig []byte, to *blockDifferTestObjects) *proto.Block {
	txs := proto.Transactions{genTransferWithWavesFee(t)}
	genSig, err := to.gsp.GenerationSignature(testGlobal.minerInfo.pk, prevGenSig)
	require.NoError(t, err)
	block, err := proto.CreateBlock(txs, 1565694219944, prevID, testGlobal.minerInfo.pk, proto.NxtConsensus{BaseTarget: 66, GenSignature: genSig}, proto.RewardBlockVersion, nil, -1, proto.TestNetScheme)
	require.NoError(t, err)
	block.BlockHeader.Version = proto.RewardBlockVersion
	block.BlockHeader.RewardVote = 700000000
	err = block.Sign(proto.TestNetScheme, testGlobal.minerInfo.sk)
	require.NoError(t, err)
	return block
}

func TestCreateBlockDiffWithReward(t *testing.T) {
	to := createBlockDiffer(t)

	// Activate NG and BlockReward
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.BlockReward))

	sig := genRandBlockIds(t, 1)[0]
	gs := crypto.MustBytesFromBase58(defaultGenSig)

	// First block
	block1 := genBlockWithSingleTransaction(t, sig, gs, to)
	to.stor.addBlock(t, block1.BlockID())
	txs := block1.Transactions
	for _, tx := range txs {
		err := to.blockDiffer.countMinerFee(tx)
		require.NoError(t, err)
	}
	err := to.blockDiffer.saveCurFeeDistr(&block1.BlockHeader)
	require.NoError(t, err)

	// Second block
	block2 := genBlockWithSingleTransaction(t, block1.BlockID(), block1.GenSignature, to)
	to.stor.addBlock(t, block2.BlockID())
	minerDiff, err := to.blockDiffer.createMinerDiff(&block2.BlockHeader, true)
	require.NoError(t, err)

	fee := defaultFee - defaultFee/5*2
	correctMinerWavesBalanceDiff := newBalanceDiff(int64(fee+to.blockDiffer.settings.FunctionalitySettings.InitialBlockReward), 0, 0, false)
	correctMinerWavesBalanceDiff.blockID = block2.BlockID()
	correctMinerDiff := txDiff{testGlobal.minerInfo.wavesKey: correctMinerWavesBalanceDiff}
	assert.Equal(t, correctMinerDiff, minerDiff)
}

func TestBlockRewardDistributionWithTwoAddresses(t *testing.T) {
	sets := *settings.TestNetSettings
	// Add some addresses for reward distribution
	sets.RewardAddresses = []proto.WavesAddress{testGlobal.senderInfo.addr, testGlobal.recipientInfo.addr}
	sets.InitialBlockReward = 800000000
	to := createBlockDifferWithSettings(t, &sets)

	// Activate NG and BlockReward
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.BlockReward))
	to.stor.activateFeature(t, int16(settings.BlockRewardDistribution))

	sig := genRandBlockIds(t, 1)[0]
	gs := crypto.MustBytesFromBase58(defaultGenSig)

	// First block
	block1 := genBlockWithSingleTransaction(t, sig, gs, to)
	to.stor.addBlock(t, block1.BlockID())
	txs := block1.Transactions
	for _, tx := range txs {
		err := to.blockDiffer.countMinerFee(tx)
		require.NoError(t, err)
	}
	err := to.blockDiffer.saveCurFeeDistr(&block1.BlockHeader)
	require.NoError(t, err)

	// Second block
	block2 := genBlockWithSingleTransaction(t, block1.BlockID(), block1.GenSignature, to)
	to.stor.addBlock(t, block2.BlockID())
	minerDiff, err := to.blockDiffer.createMinerDiff(&block2.BlockHeader, true)
	require.NoError(t, err)

	fee := int64(defaultFee - defaultFee/5*2)
	reward := int64(to.blockDiffer.settings.FunctionalitySettings.InitialBlockReward)
	additionalAddressReward := reward / 3
	correctFirstRewardAddressBalanceDiff := newBalanceDiff(additionalAddressReward, 0, 0, false)
	correctSecondRewardAddressBalanceDiff := newBalanceDiff(additionalAddressReward, 0, 0, false)
	correctMinerWavesBalanceDiff := newBalanceDiff(fee+reward-2*additionalAddressReward, 0, 0, false)
	correctMinerWavesBalanceDiff.blockID = block2.BlockID()
	correctFirstRewardAddressBalanceDiff.blockID = block2.BlockID()
	correctSecondRewardAddressBalanceDiff.blockID = block2.BlockID()
	correctDiff := txDiff{
		testGlobal.minerInfo.wavesKey:     correctMinerWavesBalanceDiff,
		testGlobal.senderInfo.wavesKey:    correctFirstRewardAddressBalanceDiff,
		testGlobal.recipientInfo.wavesKey: correctSecondRewardAddressBalanceDiff,
	}
	assert.Equal(t, correctDiff, minerDiff)
}

func TestBlockRewardDistributionWithOneAddress(t *testing.T) {
	sets := *settings.TestNetSettings
	// Add some addresses for reward distribution
	sets.RewardAddresses = []proto.WavesAddress{testGlobal.senderInfo.addr}
	to := createBlockDifferWithSettings(t, &sets)

	// Activate NG and BlockReward
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateFeature(t, int16(settings.BlockReward))
	to.stor.activateFeature(t, int16(settings.BlockRewardDistribution))

	sig := genRandBlockIds(t, 1)[0]
	gs := crypto.MustBytesFromBase58(defaultGenSig)

	// First block
	block1 := genBlockWithSingleTransaction(t, sig, gs, to)
	to.stor.addBlock(t, block1.BlockID())
	txs := block1.Transactions
	for _, tx := range txs {
		err := to.blockDiffer.countMinerFee(tx)
		require.NoError(t, err)
	}
	err := to.blockDiffer.saveCurFeeDistr(&block1.BlockHeader)
	require.NoError(t, err)

	// Second block
	block2 := genBlockWithSingleTransaction(t, block1.BlockID(), block1.GenSignature, to)
	to.stor.addBlock(t, block2.BlockID())
	minerDiff, err := to.blockDiffer.createMinerDiff(&block2.BlockHeader, true)
	require.NoError(t, err)

	fee := defaultFee - defaultFee/5*2
	correctMinerWavesBalanceDiff := newBalanceDiff(int64(fee+(to.blockDiffer.settings.FunctionalitySettings.InitialBlockReward/2)), 0, 0, false)
	correctRewardAddressBalanceDiff := newBalanceDiff(int64(to.blockDiffer.settings.FunctionalitySettings.InitialBlockReward/2), 0, 0, false)
	correctMinerWavesBalanceDiff.blockID = block2.BlockID()
	correctRewardAddressBalanceDiff.blockID = block2.BlockID()
	correctDiff := txDiff{
		testGlobal.minerInfo.wavesKey:  correctMinerWavesBalanceDiff,
		testGlobal.senderInfo.wavesKey: correctRewardAddressBalanceDiff,
	}
	assert.Equal(t, correctDiff, minerDiff)
}

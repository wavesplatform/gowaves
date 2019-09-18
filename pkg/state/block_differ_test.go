package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type blockDifferTestObjects struct {
	stor        *testStorageObjects
	blockDiffer *blockDiffer
}

func createBlockDiffer(t *testing.T) (*blockDifferTestObjects, []string) {
	sets := settings.MainNetSettings
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	genesis, err := sets.GenesisGetter.Get()
	assert.NoError(t, err, "GenesisGetter.Get() failed")
	handler, err := newTransactionHandler(genesis.BlockSignature, stor.entities, sets)
	assert.NoError(t, err, "newTransactionHandler() failed")
	blockDiffer, err := newBlockDiffer(handler, stor.entities, sets)
	assert.NoError(t, err, "newBlockDiffer() failed")
	return &blockDifferTestObjects{stor, blockDiffer}, path
}

func genBlocks(t *testing.T, to *blockDifferTestObjects) (*proto.Block, *proto.Block) {
	// Create and sign parent block.
	txsRepr := proto.NewReprFromTransactions([]proto.Transaction{createTransferV1(t)})
	randSig := genRandBlockIds(t, 1)[0]
	genSig, err := crypto.NewDigestFromBase58(defaultGenSig)
	assert.NoError(t, err, "NewDigestFromString() failed")
	parent, err := proto.CreateBlock(txsRepr, 1565694219644, randSig, testGlobal.matcherInfo.pk, proto.NxtConsensus{BaseTarget: 65, GenSignature: genSig})
	assert.NoError(t, err, "CreateBlock() failed")
	err = parent.Sign(testGlobal.matcherInfo.sk)
	assert.NoError(t, err, "Block.Sign() failed")

	// Create and sign child block.
	txsRepr = proto.NewReprFromTransactions([]proto.Transaction{createIssueV1(t)})
	genSig, err = consensus.GeneratorSignature(parent.GenSignature, testGlobal.minerInfo.pk)
	assert.NoError(t, err, "GeneratorSignature() failed")
	child, err := proto.CreateBlock(txsRepr, 1565694219944, parent.BlockSignature, testGlobal.minerInfo.pk, proto.NxtConsensus{BaseTarget: 66, GenSignature: genSig})
	assert.NoError(t, err, "CreateBlock() failed")
	err = child.Sign(testGlobal.minerInfo.sk)
	assert.NoError(t, err, "Block.Sign() failed")
	return parent, child
}

func TestCreateBlockDiffWithoutNg(t *testing.T) {
	to, path := createBlockDiffer(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	block, _ := genBlocks(t, to)
	minerDiff, err := to.blockDiffer.createMinerDiff(&block.BlockHeader, true)
	assert.NoError(t, err, "createMinerDiff() failed")
	// Empty miner diff before NG activation.
	assert.Equal(t, txDiff{}, minerDiff)
}

func TestCreateBlockDiffNg(t *testing.T) {
	to, path := createBlockDiffer(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	parent, child := genBlocks(t, to)
	// Activate NG first of all.
	to.stor.activateFeature(t, int16(settings.NG))

	// Create diff from parent block.
	txs, err := parent.Transactions.Transactions()
	assert.NoError(t, err, "Transactions() failed")
	for _, tx := range txs {
		err = to.blockDiffer.countMinerFee(tx)
		assert.NoError(t, err, "countMinerFee() failed")
	}
	err = to.blockDiffer.saveCurFeeDistr(&parent.BlockHeader)
	assert.NoError(t, err, "saveCurFeeDistr() failed")
	parentFeeTotal := int64(txs[0].GetFee())
	parentFeePrevBlock := parentFeeTotal / 5 * 2
	parentFeeNextBlock := parentFeeTotal - parentFeePrevBlock

	// Create diff from child block.
	minerDiff, err := to.blockDiffer.createMinerDiff(&child.BlockHeader, true)
	assert.NoError(t, err, "createMinerDiff() failed")
	// Verify child block miner's diff.
	correctMinerAssetBalanceDiff := newBalanceDiff(parentFeeNextBlock, 0, 0, false)
	correctMinerAssetBalanceDiff.blockID = child.BlockSignature
	correctMinerDiff := txDiff{
		testGlobal.minerInfo.assetKey: correctMinerAssetBalanceDiff,
	}
	assert.Equal(t, correctMinerDiff, minerDiff)
}

func TestCreateBlockDiffSponsorship(t *testing.T) {
	to, path := createBlockDiffer(t)

	defer func() {
		to.stor.close(t)

		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	parent, child := genBlocks(t, to)
	// Create asset.
	to.stor.createAsset(t, testGlobal.asset0.asset.ID)

	// Activate NG and FeeSponsorship first of all.
	to.stor.activateFeature(t, int16(settings.NG))
	to.stor.activateSponsorship(t)

	// Sponsor asset.
	assetCost := uint64(100500)
	to.stor.addBlock(t, blockID0)
	err := to.stor.entities.sponsoredAssets.sponsorAsset(testGlobal.asset0.asset.ID, assetCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")

	// Create diff from parent block.
	txs, err := parent.Transactions.Transactions()
	assert.NoError(t, err, "Transactions() failed")
	for _, tx := range txs {
		err = to.blockDiffer.countMinerFee(tx)
		assert.NoError(t, err, "countMinerFee() failed")
	}
	err = to.blockDiffer.saveCurFeeDistr(&parent.BlockHeader)
	assert.NoError(t, err, "saveCurFeeDistr() failed")
	_, err = to.blockDiffer.createMinerDiff(&parent.BlockHeader, false)
	assert.NoError(t, err, "createMinerDiff() failed")
	parentFeeTotal := int64(txs[0].GetFee() * FeeUnit / assetCost)
	parentFeePrevBlock := parentFeeTotal / 5 * 2
	parentFeeNextBlock := parentFeeTotal - parentFeePrevBlock

	// Create diff from child block.
	minerDiff, err := to.blockDiffer.createMinerDiff(&child.BlockHeader, true)
	assert.NoError(t, err, "createMinerDiff() failed")
	// Verify child block miner's diff.
	correctMinerWavesBalanceDiff := newBalanceDiff(parentFeeNextBlock, 0, 0, false)
	correctMinerWavesBalanceDiff.blockID = child.BlockSignature
	correctMinerDiff := txDiff{
		testGlobal.minerInfo.wavesKey: correctMinerWavesBalanceDiff,
	}
	assert.Equal(t, correctMinerDiff, minerDiff)
}

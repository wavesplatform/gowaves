package state

import (
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type blockDifferTestObjects struct {
	stor        *storageObjects
	entities    *blockchainEntitiesStorage
	blockDiffer *blockDiffer
}

func createBlockDiffer(t *testing.T) (*blockDifferTestObjects, []string) {
	sets := settings.MainNetSettings
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, stor.stateDB, sets)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	genesis, err := sets.GenesisGetter.Get()
	assert.NoError(t, err, "GenesisGetter.Get() failed")
	handler, err := newTransactionHandler(genesis.BlockSignature, entities, sets)
	assert.NoError(t, err, "newTransactionHandler() failed")
	blockDiffer, err := newBlockDiffer(handler, entities, sets)
	assert.NoError(t, err, "newBlockDiffer() failed")
	return &blockDifferTestObjects{stor, entities, blockDiffer}, path
}

func genBlocks(t *testing.T, to *blockDifferTestObjects) (*proto.Block, *proto.Block) {
	// Create and sign parent block.
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	minerSk0, minerPk0 := crypto.GenerateKeyPair(seed)

	txsRepr := proto.NewReprFromTransactions([]proto.Transaction{createTransferV1(t)})
	randSig := genRandBlockIds(t, 1)[0]
	genSig, err := crypto.NewDigestFromBase58(defaultGenSig)
	assert.NoError(t, err, "NewDigestFromString() failed")
	parent, err := proto.CreateBlock(txsRepr, 1565694219644, randSig, minerPk0, proto.NxtConsensus{BaseTarget: 65, GenSignature: genSig})
	assert.NoError(t, err, "CreateBlock() failed")
	err = parent.Sign(minerSk0)
	assert.NoError(t, err, "Block.Sign() failed")

	// Create and sign child block.
	seed1, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk2ac")
	minerSk1, minerPk1 := crypto.GenerateKeyPair(seed1)

	txsRepr = proto.NewReprFromTransactions([]proto.Transaction{createIssueV1(t)})
	genSig, err = consensus.GeneratorSignature(parent.GenSignature, minerPk1)
	assert.NoError(t, err, "GeneratorSignature() failed")
	child, err := proto.CreateBlock(txsRepr, 1565694219944, parent.BlockSignature, minerPk1, proto.NxtConsensus{BaseTarget: 66, GenSignature: genSig})
	assert.NoError(t, err, "CreateBlock() failed")
	err = child.Sign(minerSk1)
	assert.NoError(t, err, "Block.Sign() failed")
	return parent, child
}

func TestCreateBlockDiffWithoutNg(t *testing.T) {
	to, path := createBlockDiffer(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	block, _ := genBlocks(t, to)
	txs, err := block.Transactions.Transactions()
	assert.NoError(t, err, "Transactions() failed")
	diff, err := to.blockDiffer.createBlockDiff(txs, &block.BlockHeader, true, true)
	assert.NoError(t, err, "createBlockDiff() failed")
	// Empty miner diff before NG activation.
	assert.Equal(t, txDiff{}, diff.minerDiff)
}

func TestCreateBlockDiffNg(t *testing.T) {
	to, path := createBlockDiffer(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	parent, child := genBlocks(t, to)
	// Activate NG first of all.
	activateFeature(t, to.entities.features, to.stor, int16(settings.NG))

	// Create diff from parent block.
	txs, err := parent.Transactions.Transactions()
	assert.NoError(t, err, "Transactions() failed")
	_, err = to.blockDiffer.createBlockDiff(txs, &parent.BlockHeader, true, false)
	assert.NoError(t, err, "createBlockDiff() failed")
	parentFeeTotal := int64(txs[0].GetFee())

	// Create diff from child block.
	txs, err = child.Transactions.Transactions()
	assert.NoError(t, err, "Transactions() failed")
	diff, err := to.blockDiffer.createBlockDiff(txs, &child.BlockHeader, true, true)
	assert.NoError(t, err, "createBlockDiff() failed")
	// Verify child block miner's diff.
	minerAssetKey, err := assetKeyFromPk(child.GenPublicKey, testGlobal.asset0.assetID)
	assert.NoError(t, err, "assetKeyFromPk() failed")
	correctMinerAssetBalanceDiff := newBalanceDiff(parentFeeTotal/5*3, 0, 0, false)
	correctMinerAssetBalanceDiff.blockID = child.BlockSignature
	correctMinerDiff := txDiff{
		minerAssetKey: correctMinerAssetBalanceDiff,
	}
	assert.Equal(t, correctMinerDiff, diff.minerDiff)
}

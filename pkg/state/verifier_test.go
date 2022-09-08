package state

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func verifyTransactions(transactions []proto.Transaction, chans *verifierChans) error {
	for _, tx := range transactions {
		task := &verifyTask{
			taskType:   verifyTx,
			tx:         tx,
			checkTxSig: true,
		}
		if err := chans.trySend(task); err != nil {
			return err
		}
	}
	return chans.closeAndWait()
}

func verifyBlocks(blocks []proto.Block, chans *verifierChans) error {
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		task := &verifyTask{
			taskType: verifyBlock,
			parentID: blocks[i-1].BlockID(),
			block:    &block,
		}
		if err := chans.trySend(task); err != nil {
			return err
		}
	}
	return chans.closeAndWait()
}

func TestVerifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Read real blocks.
	height := uint64(75)
	blocks, err := readBlocksFromTestPath(int(height + 1))
	assert.NoError(t, err, "readBlocksFromTestPath() failed")
	last := blocks[len(blocks)-1]
	// Get real block's transactions.
	txs := last.Transactions

	// Test valid blocks.
	chans := launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	// Test valid transactions.
	err = verifyTransactions(txs, chans)
	assert.NoError(t, err, "verifyTransactions() failed with valid transactions")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	// Spoil block parent.
	backup := blocks[len(blocks)/2]
	blocks[len(blocks)/2].Parent = proto.NewBlockIDFromSignature(crypto.Signature{})
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong parent")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	// Spoil block signature.
	blocks[len(blocks)/2].BlockSignature = crypto.Signature{}
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong signature")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	// Test unsigned tx failure.
	spk, err := crypto.NewPublicKeyFromBase58(testPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	recipient, err := proto.NewAddressFromString(testAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	unsignedTx := proto.NewUnsignedPayment(spk, recipient, 100, 1, 0)
	unsignedTx.ID = &crypto.Signature{} // stub to avoid segfault in verifier goroutine
	txs = []proto.Transaction{unsignedTx}
	err = verifyTransactions(txs, chans)
	assert.Error(t, err, "verifyTransactions() did not fail with unsigned tx")
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	// Test invalid tx failure.
	invalidTx := proto.NewUnsignedGenesis(recipient, 0, 0)
	txs = []proto.Transaction{invalidTx}
	err = verifyTransactions(txs, chans)
	assert.Error(t, err, "verifyTransactions() did not fail with invalid tx")
}

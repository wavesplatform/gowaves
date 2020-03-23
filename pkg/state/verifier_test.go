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
		select {
		case verifyError := <-chans.errChan:
			return verifyError
		case chans.tasksChan <- task:
		}
	}
	close(chans.tasksChan)
	verifyError := <-chans.errChan
	return verifyError
}

func verifyBlocks(blocks []proto.Block, chans *verifierChans) error {
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		task := &verifyTask{
			taskType: verifyBlock,
			parentID: blocks[i-1].BlockID(),
			block:    &block,
		}
		select {
		case verifyError := <-chans.errChan:
			return verifyError
		case chans.tasksChan <- task:
		}
	}
	close(chans.tasksChan)
	verifyError := <-chans.errChan
	return verifyError
}

func TestVerifier(t *testing.T) {
	// Read real blocks.
	height := uint64(75)
	blocks, err := readBlocksFromTestPath(int(height + 1))
	assert.NoError(t, err, "readBlocksFromTestPath() failed")
	last := blocks[len(blocks)-1]
	// Get real block's transactions.
	txs := last.Transactions

	// Test valid blocks.
	chans := newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	// Test valid transactions.
	err = verifyTransactions(txs, chans)
	assert.NoError(t, err, "verifyTransactions() failed with valid transactions")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	// Spoil block parent.
	backup := blocks[len(blocks)/2]
	blocks[len(blocks)/2].Parent = proto.NewBlockIDFromSignature(crypto.Signature{})
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong parent")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	// Spoil block signature.
	blocks[len(blocks)/2].BlockSignature = crypto.Signature{}
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong signature")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	// Test unsigned tx failure.
	spk, err := crypto.NewPublicKeyFromBase58(testPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	recipient, err := proto.NewAddressFromString(testAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	unsignedTx := proto.NewUnsignedPayment(spk, recipient, 100, 1, 0)
	txs = []proto.Transaction{unsignedTx}
	err = verifyTransactions(txs, chans)
	assert.Error(t, err, "verifyTransactions() did not fail with unsigned tx")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU(), proto.MainNetScheme)
	// Test invalid tx failure.
	invalidTx := proto.NewUnsignedGenesis(recipient, 0, 0)
	txs = []proto.Transaction{invalidTx}
	err = verifyTransactions(txs, chans)
	assert.Error(t, err, "verifyTransactions() did not fail with invalid tx")
}

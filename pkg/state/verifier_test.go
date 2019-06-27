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
			taskType: verifyTx,
			tx:       tx,
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
		blockBytes, err := block.MarshalBinary()
		if err != nil {
			return err
		}
		task := &verifyTask{
			taskType:   verifyBlock,
			parentSig:  blocks[i-1].BlockSignature,
			block:      &block,
			blockBytes: blockBytes[:len(blockBytes)-crypto.SignatureSize],
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
	blocks, err := readRealBlocks(t, blocksPath(t), int(height+1))
	assert.NoError(t, err, "readRealBlocks() failed")
	last := blocks[len(blocks)-1]
	// Get real block's transactions.
	txs, err := proto.BytesToTransactions(last.TransactionCount, last.Transactions)
	assert.NoError(t, err, "BytesToTransactions() failed")

	// Test valid blocks.
	chans := newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	// Test valid transactions.
	err = verifyTransactions(txs, chans)
	assert.NoError(t, err, "verifyTransactions() failed with valid transactions")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	// Spoil block parent.
	backup := blocks[len(blocks)/2]
	blocks[len(blocks)/2].Parent = crypto.Signature{}
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong parent")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	// Spoil block signature.
	blocks[len(blocks)/2].BlockSignature = crypto.Signature{}
	err = verifyBlocks(blocks, chans)
	assert.Error(t, err, "verifyBlocks() did not fail with wrong signature")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	blocks[len(blocks)/2] = backup
	err = verifyBlocks(blocks, chans)
	assert.NoError(t, err, "verifyBlocks() failed with valid blocks")
	chans = newVerifierChans()
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
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
	go launchVerifier(context.Background(), chans, runtime.NumCPU())
	// Test invalid tx failure.
	invalidTx := proto.NewUnsignedGenesis(recipient, 0, 0)
	txs = []proto.Transaction{invalidTx}
	err = verifyTransactions(txs, chans)
	assert.Error(t, err, "verifyTransactions() did not fail with invalid tx")
}

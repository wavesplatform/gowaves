package state

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	ctx := t.Context()
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
	// Test self-challenged block.
	chans = launchVerifier(ctx, runtime.NumCPU(), proto.TestNetScheme)
	prevBlock := blocks[len(blocks)/2-1]
	block := blocks[len(blocks)/2]
	block.ChallengedHeader = &proto.ChallengedHeader{GeneratorPublicKey: block.GeneratorPublicKey}
	err = verifyBlocks([]proto.Block{prevBlock, block}, chans)
	assert.EqualError(t, err, fmt.Sprintf("State: handleTask: block '%s' is self-challenged", block.ID.String()),
		"verifyBlocks() did not fail with self-challenged block",
	)
	//
	// Test transactions
	//
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

func TestExchangeVerifier(t *testing.T) {
	const mainnetTxJSON = `
		{
		  "type": 7,
		  "id": "64Mhr6pJeX21guka35pGFb2dqppwBMGGagNF57Zq8qvD",
		  "fee": 300000,
		  "feeAssetId": null,
		  "timestamp": 1779301779221,
		  "version": 3,
		  "chainId": 87,
		  "sender": "3PHJZGJkQHDSTe1J1uoHdshhDoaBhZCzjns",
		  "senderPublicKey": "J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi",
		  "proofs": [
			"4gBCaG6rLBTQudHte8tycT72W96G4p4LCQLBnbbvHkRnmsYqrxt6cmjCfHJFn35iJBVoQ3vqWxEBUkxdUZfs4Gqn"
		  ],
		  "order1": {
			"version": 3,
			"id": "6yDEoyQ2fm59rZ7fS6jWTw9ApwFchjPnC5m8mhJU3h7C",
			"sender": "3PHJZGJkQHDSTe1J1uoHdshhDoaBhZCzjns",
			"senderPublicKey": "J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi",
			"matcherPublicKey": "J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi",
			"assetPair": {
			  "amountAsset": "4QMfJbtFQ6iKJLMvZ1BbE7Zqb6dho6zh2na8myzUGn1T",
			  "priceAsset": null
			},
			"orderType": "buy",
			"amount": 10,
			"price": 100000,
			"timestamp": 1779301779221,
			"expiration": 1779305379220,
			"matcherFee": 300000,
			"signature": "kJnGej4wz6Kmb3WUYSJWLkPjSAthML73wrJACx9w4g6EBZd4QV2CrqJ6RwvEWFSNgCEFDRQX8NpSaTQTWRFrnc6",
			"proofs": [
			  "kJnGej4wz6Kmb3WUYSJWLkPjSAthML73wrJACx9w4g6EBZd4QV2CrqJ6RwvEWFSNgCEFDRQX8NpSaTQTWRFrnc6"
			],
			"matcherFeeAssetId": null
		  },
		  "order2": {
			"version": 3,
			"id": "59KLuWTV2G6upUKVjnpfVneWSWwtqEC156ekWYQ79Nup",
			"sender": "3PHJZGJkQHDSTe1J1uoHdshhDoaBhZCzjns",
			"senderPublicKey": "J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi",
			"matcherPublicKey": "J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi",
			"assetPair": {
			  "amountAsset": "4QMfJbtFQ6iKJLMvZ1BbE7Zqb6dho6zh2na8myzUGn1T",
			  "priceAsset": null
			},
			"orderType": "sell",
			"amount": 10,
			"price": 100000,
			"timestamp": 1779301779220,
			"expiration": 1779305379220,
			"matcherFee": 300000,
			"signature": "4emUeYCTG3XDkvxNNi8bHFh6AhC45DbkM3nh1VwuckKe1UU19TqdUXUrQHbaTFDds48Nw5w3JoX1hZWdDCope14n",
			"proofs": [
			  "4emUeYCTG3XDkvxNNi8bHFh6AhC45DbkM3nh1VwuckKe1UU19TqdUXUrQHbaTFDds48Nw5w3JoX1hZWdDCope14n"
			],
			"matcherFeeAssetId": null
		  },
		  "amount": 10,
		  "price": 100000,
		  "buyMatcherFee": 300000,
		  "sellMatcherFee": 300000,
		  "height": 5232511,
		  "applicationStatus": "succeeded",
		  "spentComplexity": 0
		}`
	var tx proto.ExchangeWithProofs
	err := json.Unmarshal([]byte(mainnetTxJSON), &tx)
	require.NoError(t, err)
	const scheme = proto.MainNetScheme

	t.Run("verify_tx", func(t *testing.T) {
		err = verifyExchangeTransaction(&tx, scheme, true, true)
		assert.NoError(t, err)
	})
	t.Run("verify_first_oder", func(t *testing.T) {
		o := tx.GetOrder1()
		ok, vErr := o.Verify(scheme)
		assert.True(t, ok)
		assert.NoError(t, vErr)
	})
	t.Run("verify_second_order", func(t *testing.T) {
		o := tx.GetOrder2()
		ok, vErr := o.Valid()
		assert.True(t, ok)
		assert.NoError(t, vErr)

		id, idErr := o.GetID()
		require.NoError(t, idErr)
		gErr := o.GenerateID(scheme)
		require.NoError(t, gErr)
		idG, idErr := o.GetID()
		require.NoError(t, idErr)
		assert.Equal(t, id, idG)

		ok, vErr = o.Verify(scheme)
		assert.True(t, ok)
		assert.NoError(t, vErr)
	})
	t.Run("build_verify_second_order", func(t *testing.T) {
		var (
			orderID   = crypto.MustDigestFromBase58("59KLuWTV2G6upUKVjnpfVneWSWwtqEC156ekWYQ79Nup")
			senderPK  = crypto.MustPublicKeyFromBase58("J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi")
			matcherPK = crypto.MustPublicKeyFromBase58("J3qN131GYh3swtQZcgtRDguVdJ8LyqfSpnUccNxWNegi")
			assetPair = proto.AssetPair{
				AmountAsset: proto.NewOptionalAsset(
					true, crypto.MustDigestFromBase58("4QMfJbtFQ6iKJLMvZ1BbE7Zqb6dho6zh2na8myzUGn1T"),
				),
				PriceAsset: proto.NewOptionalAssetWaves(),
			}
			matcherFeeAsset = proto.NewOptionalAssetWaves()
			sig             = crypto.MustSignatureFromBase58(
				"4emUeYCTG3XDkvxNNi8bHFh6AhC45DbkM3nh1VwuckKe1UU19TqdUXUrQHbaTFDds48Nw5w3JoX1hZWdDCope14n",
			)
		)
		const (
			orderType  = proto.Sell
			price      = 100000
			amount     = 10
			timestamp  = 1779301779220
			expiration = 1779305379220
			matcherFee = 300000
		)
		ov3 := proto.NewUnsignedOrderV3(
			senderPK,
			matcherPK,
			assetPair.AmountAsset,
			assetPair.PriceAsset,
			orderType,
			price,
			amount,
			timestamp,
			expiration,
			matcherFee,
			matcherFeeAsset,
		)
		ov3.Proofs = proto.NewProofsFromSignature(&sig)

		idErr := ov3.GenerateID(scheme)
		require.NoError(t, idErr)
		require.Equal(t, orderID.String(), ov3.ID.String())

		ok, vErr := ov3.Verify(scheme)
		assert.True(t, ok)
		assert.NoError(t, vErr)
	})
}

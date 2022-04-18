package genesis_generator

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type GenesisTransactionInfo struct {
	Address   proto.WavesAddress
	Amount    uint64
	Timestamp uint64
}

var (
	genesisBlockParent  = proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("67rpwLCuS5DGA8KGZXKsVQ7dnPb9goRLoKfgGbLfQg9WoLUgNY77E2jT11fem3coV9nAkguBACzrU1iyZM4B8roQ"))
	genesisKeyPair      = proto.MustKeyPair([]byte{})
	genesisGenSignature = crypto.MustBytesFromBase58("11111111111111111111111111111111")
)

// GenerateGenesisBlock creates a new genesis block with a new signature. The signature will be different each time.
// This function should be used to create genesis block for integration tests and alike.
func GenerateGenesisBlock(scheme proto.Scheme, transactions []GenesisTransactionInfo, baseTarget uint64, timestamp proto.Timestamp) (*proto.Block, error) {
	txs, err := makeTransactions(scheme, transactions)
	if err != nil {
		return nil, err
	}
	return newGenesisBlock(scheme, txs, baseTarget, timestamp)
}

// RecreateGenesisBlock builds the GenesisBlock and sets it signature to the given signature.
// Use this function to reproduce existing genesis blocks for known networks.
func RecreateGenesisBlock(scheme proto.Scheme, transactions []GenesisTransactionInfo, baseTarget uint64, timestamp proto.Timestamp, signature crypto.Signature) (*proto.Block, error) {
	block, err := GenerateGenesisBlock(scheme, transactions, baseTarget, timestamp)
	if err != nil {
		return nil, err
	}
	block.BlockSignature = signature
	return block, nil
}

func newGenesisBlock(scheme proto.Scheme, transactions proto.Transactions, baseTarget, timestamp proto.Timestamp) (*proto.Block, error) {
	consensus := proto.NxtConsensus{BaseTarget: baseTarget, GenSignature: genesisGenSignature}
	block, err := proto.CreateBlock(transactions, timestamp, genesisBlockParent, genesisKeyPair.Public, consensus,
		proto.GenesisBlockVersion, nil, 0, scheme)
	if err != nil {
		return nil, err
	}
	err = block.Sign(scheme, genesisKeyPair.Secret)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func makeTransactions(scheme proto.Scheme, transactions []GenesisTransactionInfo) (proto.Transactions, error) {
	txs := make(proto.Transactions, len(transactions))
	for i := range transactions {
		tx := proto.NewUnsignedGenesis(transactions[i].Address, transactions[i].Amount, transactions[i].Timestamp)
		err := tx.GenerateSigID(scheme)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}

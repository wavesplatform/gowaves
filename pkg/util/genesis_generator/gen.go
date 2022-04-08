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

func newGenesisBlock(scheme proto.Scheme, transactions proto.Transactions, baseTarget, timestamp proto.Timestamp) (*proto.Block, error) {
	id := proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("67rpwLCuS5DGA8KGZXKsVQ7dnPb9goRLoKfgGbLfQg9WoLUgNY77E2jT11fem3coV9nAkguBACzrU1iyZM4B8roQ"))
	block, err := proto.CreateBlock(
		transactions,
		timestamp,
		id,
		crypto.PublicKey{},
		proto.NxtConsensus{
			BaseTarget:   baseTarget,
			GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
		},
		proto.GenesisBlockVersion,
		nil,
		0,
		scheme,
	)

	if err != nil {
		return nil, err
	}

	kp := proto.MustKeyPair([]byte{})
	err = block.Sign(scheme, kp.Secret)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func Generate(scheme byte, transactions []GenesisTransactionInfo, baseTarget, timestamp proto.Timestamp) (*proto.Block, error) {
	txs := make(proto.Transactions, len(transactions))
	for i := range transactions {
		tx := proto.NewUnsignedGenesis(transactions[i].Address, transactions[i].Amount, transactions[i].Timestamp)
		err := tx.GenerateSigID(scheme)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return newGenesisBlock(scheme, txs, baseTarget, timestamp)
}

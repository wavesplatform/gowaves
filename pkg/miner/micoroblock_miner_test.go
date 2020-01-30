package miner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestMineMicroblock(t *testing.T) {

	now := proto.NewTimestampFromTime(time.Now())
	noSig := crypto.Signature{}

	keyPair, _ := proto.NewKeyPair([]byte("test"))

	nxt := proto.NxtConsensus{
		BaseTarget:   153722867,
		GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
	}

	keyBlock, err := proto.CreateBlock(
		proto.NewReprFromTransactions(nil),
		now,
		noSig,
		keyPair.Public,
		nxt,
		proto.RewardBlockVersion,
		nil,
		-1,
	)
	require.NoError(t, err)
	err = keyBlock.Sign(keyPair.Secret)
	require.NoError(t, err)

	transferV1 := byte_helpers.TransferV1.Transaction.Clone()

	micro, err := createMicroBlock(keyBlock, proto.NewReprFromTransactions([]proto.Transaction{transferV1}), keyPair, proto.MainNetScheme)
	require.NoError(t, err)

	t.Logf("%+v", keyBlock)
	t.Logf("%+v", micro)

}

func createMicroBlock(keyBlock *proto.Block, tr *proto.TransactionsRepresentation, keyPair proto.KeyPair, chainID proto.Scheme) (*proto.MicroBlock, error) {
	blockApplyOn := keyBlock
	bts_, err := blockApplyOn.Transactions.Bytes()
	if err != nil {
		return nil, err
	}
	bts := make([]byte, len(bts_))
	copy(bts, bts_)

	transactions, err := blockApplyOn.Transactions.Join(tr)
	if err != nil {
		return nil, err
	}

	newBlock, err := proto.CreateBlock(
		transactions,
		blockApplyOn.Timestamp,
		blockApplyOn.Parent,
		blockApplyOn.GenPublicKey,
		blockApplyOn.NxtConsensus,
		blockApplyOn.Version,
		blockApplyOn.Features,
		blockApplyOn.RewardVote,
	)
	if err != nil {
		return nil, err
	}

	priv := keyPair.Secret
	err = newBlock.Sign(keyPair.Secret)
	if err != nil {
		return nil, err
	}

	micro := proto.MicroBlock{
		VersionField:          3,
		SenderPK:              keyPair.Public,
		Transactions:          tr,
		TransactionCount:      uint32(tr.Count()),
		PrevResBlockSigField:  keyBlock.BlockSignature,
		TotalResBlockSigField: newBlock.BlockSignature,
	}

	err = micro.Sign(priv)
	if err != nil {
		return nil, err
	}

	return &micro, nil
}

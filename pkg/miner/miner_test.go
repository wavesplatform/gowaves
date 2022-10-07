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
		proto.Transactions(nil),
		now,
		proto.NewBlockIDFromSignature(noSig),
		keyPair.Public,
		nxt,
		proto.RewardBlockVersion,
		nil,
		-1,
		proto.TestNetScheme,
	)
	require.NoError(t, err)
	err = keyBlock.Sign(proto.TestNetScheme, keyPair.Secret)
	require.NoError(t, err)

	transferWithSig := byte_helpers.TransferWithSig.Transaction.Clone()

	_, err = createMicroBlock(keyBlock, []proto.Transaction{transferWithSig}, keyPair, proto.TestNetScheme)
	require.NoError(t, err)
}

func createMicroBlock(keyBlock *proto.Block, tr proto.Transactions, keyPair proto.KeyPair, scheme proto.Scheme) (*proto.MicroBlock, error) {
	blockApplyOn := keyBlock
	transactions := blockApplyOn.Transactions.Join(tr)

	newBlock, err := proto.CreateBlock(
		transactions,
		blockApplyOn.Timestamp,
		blockApplyOn.Parent,
		blockApplyOn.GeneratorPublicKey,
		blockApplyOn.NxtConsensus,
		blockApplyOn.Version,
		blockApplyOn.Features,
		blockApplyOn.RewardVote,
		scheme,
	)
	if err != nil {
		return nil, err
	}

	sk := keyPair.Secret
	err = newBlock.Sign(proto.TestNetScheme, keyPair.Secret)
	if err != nil {
		return nil, err
	}

	micro := proto.MicroBlock{
		VersionField:          3,
		SenderPK:              keyPair.Public,
		Transactions:          tr,
		TransactionCount:      uint32(tr.Count()),
		Reference:             keyBlock.BlockID(),
		TotalResBlockSigField: newBlock.BlockSignature,
	}

	err = micro.Sign(proto.TestNetScheme, sk)
	if err != nil {
		return nil, err
	}

	return &micro, nil
}

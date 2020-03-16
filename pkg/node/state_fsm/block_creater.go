package state_fsm

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlockCreaterImpl struct {
}

func (a BlockCreaterImpl) FromMicroblockRow(seq types.MicroblockRow) (*proto.Block, error) {
	keyBlock := seq.KeyBlock
	t := keyBlock.Transactions
	BlockSignature := keyBlock.BlockSignature
	for _, row := range seq.MicroBlocks {
		t = t.Join(row.Transactions)
		BlockSignature = row.TotalResBlockSigField
	}

	block, err := proto.CreateBlock(
		t,
		keyBlock.Timestamp,
		keyBlock.Parent,
		keyBlock.GenPublicKey,
		keyBlock.NxtConsensus,
		keyBlock.Version,
		keyBlock.Features,
		keyBlock.RewardVote)
	if err != nil {
		return nil, err
	}
	block.BlockSignature = BlockSignature
	return block, nil
}

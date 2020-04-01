package proto

type MicroblockRow struct {
	KeyBlock    *Block
	MicroBlocks []*MicroBlock
}

func (row MicroblockRow) LastSignature() BlockID {
	if len(row.MicroBlocks) > 0 {
		return NewBlockIDFromSignature(row.MicroBlocks[len(row.MicroBlocks)-1].Signature)
	} else {
		return row.KeyBlock.BlockID()
	}
}

type BlockCreatorImpl struct {
	s Scheme
}

func NewBlockCreator(s Scheme) *BlockCreatorImpl {
	return &BlockCreatorImpl{s: s}
}

func (a BlockCreatorImpl) FromMicroblockRow(seq MicroblockRow) (*Block, error) {
	keyBlock := seq.KeyBlock
	t := keyBlock.Transactions
	BlockSignature := keyBlock.BlockSignature
	for _, row := range seq.MicroBlocks {
		t = t.Join(row.Transactions)
		BlockSignature = row.TotalResBlockSigField
	}

	block, err := CreateBlock(
		t,
		keyBlock.Timestamp,
		keyBlock.Parent,
		keyBlock.GenPublicKey,
		keyBlock.NxtConsensus,
		keyBlock.Version,
		keyBlock.Features,
		keyBlock.RewardVote,
		a.s)
	if err != nil {
		return nil, err
	}
	block.BlockSignature = BlockSignature
	return block, nil
}

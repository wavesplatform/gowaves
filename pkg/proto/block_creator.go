package proto

type MicroblockRow struct {
	KeyBlock    *Block
	MicroBlocks []*MicroBlock
}

type BlockCreatorImpl struct {
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
		keyBlock.RewardVote)
	if err != nil {
		return nil, err
	}
	block.BlockSignature = BlockSignature
	return block, nil
}

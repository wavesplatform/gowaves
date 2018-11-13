package server

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockBatch struct {
	pendingBlocksHave map[proto.BlockID]bool
	orphanedBlocks    map[proto.BlockID]*proto.Block
	inPlaceBlocks     map[proto.BlockID]*proto.Block
	ancestor          proto.BlockID
}

func (b *blockBatch) haveAll() bool {
	for _, v := range b.pendingBlocksHave {
		if !v {
			return v
		}
	}

	return true
}

func (b *blockBatch) addPendingBlock(block proto.BlockID, have bool) {
	b.pendingBlocksHave[block] = have
}

func (b *blockBatch) addOrphaned(block *proto.Block) {
	b.orphanedBlocks[block.Parent] = block
}

func (b *blockBatch) contains(block proto.BlockID) bool {
	_, ok := b.pendingBlocksHave[block]
	return ok
}

func (b *blockBatch) addBlock(block *proto.Block) error {
	if !b.contains(block.BlockSignature) {
		return errors.New("batch does not contain block")
	}
LOOP:
	for {
		switch {
		case block.BlockSignature == b.ancestor:
			b.pendingBlocksHave[block.BlockSignature] = true
			b.inPlaceBlocks[block.BlockSignature] = block
		default:
			_, ok := b.inPlaceBlocks[block.Parent]
			if !ok {
				b.orphanedBlocks[block.Parent] = block
				break LOOP
			}
			b.inPlaceBlocks[block.BlockSignature] = block
		}

		b.pendingBlocksHave[block.BlockSignature] = true
		orphan, ok := b.orphanedBlocks[block.BlockSignature]
		if !ok {
			break
		}
		delete(b.orphanedBlocks, block.BlockSignature)
		b.inPlaceBlocks[orphan.BlockSignature] = orphan
		block = orphan
	}

	return nil
}

func (b *blockBatch) orderedBatch() ([]*proto.Block, error) {
	batch := make([]*proto.Block, 0, len(b.inPlaceBlocks))
	byParent := make(map[proto.BlockID]*proto.Block)

	for id, block := range b.inPlaceBlocks {
		if id == b.ancestor {
			continue
		}

		byParent[block.Parent] = block
	}

	begin := b.ancestor

	block, ok := b.inPlaceBlocks[b.ancestor]
	if !ok {
		return nil, errors.New("not found")
	}
	batch = append(batch, block)
	for {
		block, ok := byParent[begin]
		if !ok {
			break
		}
		begin = block.BlockSignature
		batch = append(batch, block)
	}

	return batch, nil
}

func NewBatch(batch []proto.BlockID) (*blockBatch, error) {
	if len(batch) == 0 {
		return nil, errors.New("empty batch")
	}

	b := &blockBatch{
		ancestor:          batch[0],
		pendingBlocksHave: make(map[proto.BlockID]bool),
		orphanedBlocks:    make(map[proto.BlockID]*proto.Block),
		inPlaceBlocks:     make(map[proto.BlockID]*proto.Block),
	}

	for _, block := range batch {
		b.addPendingBlock(block, false)
	}
	return b, nil
}

package server

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var batchIncomplete error = errors.New("batch incomplete")
var noSuchBlock error = errors.New("no such block")
var batchEmpty error = errors.New("batch empty")

type blockBatch struct {
	pendingBlocksHave map[crypto.Signature]bool
	orphanedBlocks    map[crypto.Signature]*proto.Block
	inPlaceBlocks     map[crypto.Signature]*proto.Block
	ancestor          crypto.Signature
}

func (b *blockBatch) haveAll() bool {
	for _, v := range b.pendingBlocksHave {
		if !v {
			return v
		}
	}

	return true
}

func (b *blockBatch) addPendingBlock(block crypto.Signature, have bool) {
	b.pendingBlocksHave[block] = have
}

func (b *blockBatch) addOrphaned(block *proto.Block) {
	b.orphanedBlocks[block.Parent] = block
}

func (b *blockBatch) contains(block crypto.Signature) bool {
	_, ok := b.pendingBlocksHave[block]
	return ok
}

func (b *blockBatch) addBlock(block *proto.Block) error {
	if !b.contains(block.BlockSignature) {
		return noSuchBlock
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
	byParent := make(map[crypto.Signature]*proto.Block)

	for id, block := range b.inPlaceBlocks {
		if id == b.ancestor {
			continue
		}

		byParent[block.Parent] = block
	}

	begin := b.ancestor

	block, ok := b.inPlaceBlocks[b.ancestor]
	if !ok {
		return nil, batchIncomplete
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

	if len(batch) != len(b.pendingBlocksHave) {
		return nil, batchIncomplete
	}

	return batch, nil
}

func NewBatch(batch []crypto.Signature) (*blockBatch, error) {
	if len(batch) == 0 {
		return nil, batchEmpty
	}

	b := &blockBatch{
		ancestor:          batch[0],
		pendingBlocksHave: make(map[crypto.Signature]bool),
		orphanedBlocks:    make(map[crypto.Signature]*proto.Block),
		inPlaceBlocks:     make(map[crypto.Signature]*proto.Block),
	}

	for _, block := range batch {
		b.addPendingBlock(block, false)
	}
	return b, nil
}

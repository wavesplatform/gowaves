package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blocksCache struct {
	blocks map[proto.BlockID]proto.Block
}

func (c *blocksCache) put(block *proto.Block) int {
	c.blocks[block.ID] = *block
	return len(c.blocks)
}

func (c *blocksCache) clear() {
	c.blocks = map[proto.BlockID]proto.Block{}
}

func (c *blocksCache) get(blockID proto.BlockID) (*proto.Block, bool) {
	block, ok := c.blocks[blockID]
	if !ok {
		return nil, false
	}
	return &block, true
}

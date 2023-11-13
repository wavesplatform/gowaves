package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type blockCache struct {
	blocks map[proto.BlockID]proto.Block
}

func (c *blockCache) put(block *proto.Block) {
	c.blocks[block.ID] = *block
}

func (c *blockCache) clear() {
	c.blocks = map[proto.BlockID]proto.Block{}
}

func (c *blockCache) get(blockID proto.BlockID) (*proto.Block, bool) {
	block, ok := c.blocks[blockID]
	if !ok {
		return nil, false
	}
	return &block, true
}

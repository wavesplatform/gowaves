package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

const defaultCacheSize = 24

type microBlockCache interface {
	put(blockID proto.BlockID, micro *proto.MicroBlock)
	get(proto.BlockID) (*proto.MicroBlock, bool)
}

type microBlockInvCache interface {
	put(blockID proto.BlockID, micro *proto.MicroBlockInv)
	get(proto.BlockID) (*proto.MicroBlockInv, bool)
	exist(blockID proto.BlockID) bool
}

type defaultMicroBlockCache struct {
	cache *fifo_cache.FIFOCache
}

func newDefaultMicroblockCache() *defaultMicroBlockCache {
	return &defaultMicroBlockCache{
		cache: fifo_cache.New(defaultCacheSize),
	}
}

func (c *defaultMicroBlockCache) put(blockID proto.BlockID, micro *proto.MicroBlock) {
	c.cache.Add2(blockID.Bytes(), micro)
}

func (c *defaultMicroBlockCache) get(sig proto.BlockID) (*proto.MicroBlock, bool) {
	rs, ok := c.cache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.MicroBlock), true
}

type defaultMicroBlockInvCache struct {
	cache *fifo_cache.FIFOCache
}

func newDefaultMicroblockInvCache() *defaultMicroBlockInvCache {
	return &defaultMicroBlockInvCache{
		cache: fifo_cache.New(defaultCacheSize),
	}
}

func (c *defaultMicroBlockInvCache) put(blockID proto.BlockID, micro *proto.MicroBlockInv) {
	c.cache.Add2(blockID.Bytes(), micro)
}

func (c *defaultMicroBlockInvCache) get(blockID proto.BlockID) (*proto.MicroBlockInv, bool) {
	rs, ok := c.cache.Get(blockID.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.MicroBlockInv), true
}

func (c *defaultMicroBlockInvCache) exist(blockID proto.BlockID) bool {
	return c.cache.Exists(blockID.Bytes())
}

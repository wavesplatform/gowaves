package microblock_cache

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

type MicroblockCache struct {
	cache *fifo_cache.FIFOCache
}

func NewMicroblockCache() *MicroblockCache {
	return &MicroblockCache{
		cache: fifo_cache.New(24),
	}
}

func (a *MicroblockCache) Add(blockID proto.BlockID, micro *proto.MicroBlock) {
	a.cache.Add2(blockID.Bytes(), micro)
}

func (a *MicroblockCache) Get(sig proto.BlockID) (*proto.MicroBlock, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.MicroBlock), true
}

type MicroblockInvCache struct {
	cache *fifo_cache.FIFOCache
}

func NewMicroblockInvCache() *MicroblockInvCache {
	return &MicroblockInvCache{
		cache: fifo_cache.New(24),
	}
}

func (a *MicroblockInvCache) Add(blockID proto.BlockID, micro *proto.MicroBlockInv) {
	a.cache.Add2(blockID.Bytes(), micro)
}

func (a *MicroblockInvCache) Get(sig proto.BlockID) (*proto.MicroBlockInv, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.MicroBlockInv), true
}

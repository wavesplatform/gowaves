package microblock_cache

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

const microBlockCacheSize = 24

type MicroBlockCache struct {
	blockCache    *fifo_cache.FIFOCache[proto.BlockID, *proto.MicroBlock]
	snapshotCache *fifo_cache.FIFOCache[proto.BlockID, *proto.BlockSnapshot]
}

func NewMicroBlockCache() *MicroBlockCache {
	return &MicroBlockCache{
		blockCache:    fifo_cache.New[proto.BlockID, *proto.MicroBlock](microBlockCacheSize),
		snapshotCache: fifo_cache.New[proto.BlockID, *proto.BlockSnapshot](microBlockCacheSize),
	}
}

func (a *MicroBlockCache) Add(blockID proto.BlockID, micro *proto.MicroBlock) {
	a.blockCache.Add2(blockID, micro)
}

func (a *MicroBlockCache) Get(sig proto.BlockID) (*proto.MicroBlock, bool) {
	rs, ok := a.blockCache.Get(sig)
	if !ok {
		return nil, false
	}
	return rs, true
}

func (a *MicroBlockCache) AddSnapshot(blockID proto.BlockID, snapshot *proto.BlockSnapshot) {
	a.snapshotCache.Add2(blockID, snapshot)
}
func (a *MicroBlockCache) GetSnapshot(sig proto.BlockID) (*proto.BlockSnapshot, bool) {
	rs, ok := a.snapshotCache.Get(sig)
	if !ok {
		return nil, false
	}
	return rs, true
}

type MicroblockInvCache struct {
	cache *fifo_cache.FIFOCache[proto.BlockID, *proto.MicroBlockInv]
}

func NewMicroblockInvCache() *MicroblockInvCache {
	const microBlockInvCacheSize = 24
	return &MicroblockInvCache{
		cache: fifo_cache.New[proto.BlockID, *proto.MicroBlockInv](microBlockInvCacheSize),
	}
}

func (a *MicroblockInvCache) Add(blockID proto.BlockID, micro *proto.MicroBlockInv) {
	a.cache.Add2(blockID, micro)
}

func (a *MicroblockInvCache) Get(sig proto.BlockID) (*proto.MicroBlockInv, bool) {
	rs, ok := a.cache.Get(sig)
	if !ok {
		return nil, false
	}
	return rs, true
}

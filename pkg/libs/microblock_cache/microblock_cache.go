package microblock_cache

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

const microBlockCacheSize = 24

type MicroBlockCache struct {
	blockCache    *fifo_cache.FIFOCache
	snapshotCache *fifo_cache.FIFOCache
}

func NewMicroBlockCache() *MicroBlockCache {
	return &MicroBlockCache{
		blockCache:    fifo_cache.New(microBlockCacheSize),
		snapshotCache: fifo_cache.New(microBlockCacheSize),
	}
}

func (a *MicroBlockCache) Add(blockID proto.BlockID, micro *proto.MicroBlock) {
	a.blockCache.Add2(blockID.Bytes(), micro)
}

func (a *MicroBlockCache) Get(sig proto.BlockID) (*proto.MicroBlock, bool) {
	rs, ok := a.blockCache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.MicroBlock), true
}

func (a *MicroBlockCache) AddSnapshot(blockID proto.BlockID, snapshot *proto.BlockSnapshot) {
	a.snapshotCache.Add2(blockID.Bytes(), snapshot)
}
func (a *MicroBlockCache) GetSnapshot(sig proto.BlockID) (*proto.BlockSnapshot, bool) {
	rs, ok := a.snapshotCache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*proto.BlockSnapshot), true
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

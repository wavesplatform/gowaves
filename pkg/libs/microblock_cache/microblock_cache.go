package microblock_cache

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

const microBlockCacheSize = 24

type microBlockWithSnapshot struct {
	microBlock *proto.MicroBlock    // always not nil
	snapshot   *proto.BlockSnapshot // can be nil
}

type MicroBlockCache struct {
	cache *fifo_cache.FIFOCache
}

func NewMicroBlockCache() *MicroBlockCache {
	return &MicroBlockCache{
		cache: fifo_cache.New(microBlockCacheSize),
	}
}

func (a *MicroBlockCache) AddMicroBlock(
	blockID proto.BlockID,
	micro *proto.MicroBlock,
) {
	a.cache.Add2(blockID.Bytes(), &microBlockWithSnapshot{
		microBlock: micro,
		snapshot:   nil, // intentionally nil
	})
}

func (a *MicroBlockCache) AddMicroBlockWithSnapshot(
	blockID proto.BlockID,
	micro *proto.MicroBlock,
	snapshot *proto.BlockSnapshot,
) {
	a.cache.Add2(blockID.Bytes(), &microBlockWithSnapshot{
		microBlock: micro,
		snapshot:   snapshot,
	})
}

func (a *MicroBlockCache) GetBlock(sig proto.BlockID) (*proto.MicroBlock, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	return rs.(*microBlockWithSnapshot).microBlock, true
}

func (a *MicroBlockCache) GetSnapshot(sig proto.BlockID) (*proto.BlockSnapshot, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if !ok {
		return nil, false
	}
	var (
		snapshot     = rs.(*microBlockWithSnapshot).snapshot
		existInCache = snapshot != nil
	)
	return snapshot, existInCache
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

package ng

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

type kvMicro struct {
	m *proto.MicroBlock
}

func (a kvMicro) Key() []byte {
	return a.m.TotalResBlockSigField.Bytes()
}

func (a kvMicro) Value() interface{} {
	return a.m
}

type kvInv struct {
	inv *proto.MicroBlockInv
}

func (a kvInv) Key() []byte {
	return a.inv.TotalBlockSig.Bytes()
}

func (a kvInv) Value() interface{} {
	return a.inv
}

type NotifyNewMicroblock interface {
	AddMicroblock(*proto.MicroBlock)
}

type MicroblockCache struct {
	cache *fifo_cache.FIFOCache
}

func NewMicroblockCache(cacheSize int) *MicroblockCache {
	return &MicroblockCache{
		cache: fifo_cache.New(cacheSize),
	}
}

func (a *MicroblockCache) AddMicroBlock(microBlock *proto.MicroBlock) {
	a.cache.Add(kvMicro{microBlock})
}

func (a MicroblockCache) MicroBlock(sig proto.MicroblockTotalSig) (*proto.MicroBlock, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if ok {
		return rs.(*proto.MicroBlock), ok
	}
	return nil, false
}

type InvCache struct {
	cache *fifo_cache.FIFOCache
}

func NewInvCache(cacheSize int) *InvCache {
	return &InvCache{
		cache: fifo_cache.New(cacheSize),
	}
}

func (a *InvCache) AddInv(inv *proto.MicroBlockInv) {
	a.cache.Add(kvInv{inv})
}

func (a *InvCache) Inv(sig proto.MicroblockTotalSig) (*proto.MicroBlockInv, bool) {
	rs, ok := a.cache.Get(sig.Bytes())
	if ok {
		return rs.(*proto.MicroBlockInv), ok
	}
	return nil, false
}

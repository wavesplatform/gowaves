package ng

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

// store only inv signatures to cache non requested
type InvRequesterImpl struct {
	cache *fifo_cache.FIFOCache
}

func NewInvRequester() *InvRequesterImpl {
	return &InvRequesterImpl{
		cache: fifo_cache.New(16),
	}
}

func (a *InvRequesterImpl) Add2Cache(id proto.BlockID) bool {
	idb := id.Bytes()
	if a.cache.Exists(idb) {
		return true
	}
	a.cache.Add2(idb, struct{}{})
	return false
}

func (a *InvRequesterImpl) Request(p types.MessageSender, id proto.BlockID) bool {
	existed := a.Add2Cache(id)
	if !existed {
		p.SendMessage(&proto.MicroBlockRequestMessage{
			TotalBlockSig: id,
		})
	}
	return existed
}

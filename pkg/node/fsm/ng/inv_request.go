package ng

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

// store only inv signatures to cache non requested
type InvRequesterImpl struct {
	cache *fifo_cache.FIFOCache[proto.BlockID, struct{}]
}

func NewInvRequester() *InvRequesterImpl {
	const invRequestsCacheSize = 16
	return &InvRequesterImpl{
		cache: fifo_cache.New[proto.BlockID, struct{}](invRequestsCacheSize),
	}
}

func (a *InvRequesterImpl) Add2Cache(id proto.BlockID) bool {
	if a.cache.Exists(id) {
		return true
	}
	a.cache.Add2(id, struct{}{})
	return false
}

func (a *InvRequesterImpl) Request(p types.MessageSender, id proto.BlockID, enableLightNode bool) bool {
	existed := a.Add2Cache(id)
	if !existed {
		idBytes := id.Bytes()
		p.SendMessage(&proto.MicroBlockRequestMessage{
			TotalBlockSig: idBytes,
		})
		if enableLightNode {
			p.SendMessage(&proto.MicroBlockSnapshotRequestMessage{
				BlockIDBytes: idBytes,
			})
		}
	}
	return existed
}

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

func (a *InvRequesterImpl) Request(p types.MessageSender, id []byte) {
	if a.cache.Exists(id) {
		return
	}
	a.cache.Add2(id, struct{}{})

	p.SendMessage(&proto.MicroBlockRequestMessage{
		TotalBlockSig: id,
	})
}

package ng

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/fifo_cache"
)

type InvRequesterImpl struct {
	mu    sync.Mutex
	cache *fifo_cache.FIFOCache
}

func NewInvRequester() *InvRequesterImpl {
	return &InvRequesterImpl{
		cache: fifo_cache.New(16),
	}
}

func (a *InvRequesterImpl) Request(p types.MessageSender, signature crypto.Signature) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cache.Exists(signature.Bytes()) {
		return
	}
	a.cache.Add2(signature.Bytes(), struct{}{})

	p.SendMessage(&proto.MicroBlockRequestMessage{
		Body: &proto.MicroBlockRequest{
			TotalBlockSig: signature,
		},
	})
}

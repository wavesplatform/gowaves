package ng

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"go.uber.org/zap"
)

type Runtime interface {
	MinedMicroblock(block *proto.MicroBlock, inv *proto.MicroBlockInv)
}

type RuntimeImpl struct {
	mu       sync.Mutex
	blocks   *MicroblockCache
	inv      *InvCache
	services services.Services
	ngState  *State

	// we send request for this microblock and waiting for it
	waitingOnMicroblock *crypto.Signature
}

func NewRuntime(services services.Services, ngState *State) *RuntimeImpl {
	return &RuntimeImpl{
		blocks:  NewMicroblockCache(32),
		inv:     NewInvCache(32),
		ngState: ngState,

		services: services,
	}
}

func (a *RuntimeImpl) MinedMicroblock(block *proto.MicroBlock, inv *proto.MicroBlockInv) {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, ok := a.blocks.MicroBlock(block.TotalResBlockSigField)
	if !ok {
		a.blocks.AddMicroBlock(block)
		a.inv.AddInv(inv)
		bts, err := inv.MarshalBinary()
		if err != nil {
			zap.S().Error(err)
			return
		}
		a.services.Peers.EachConnected(func(peer peer.Peer, i *proto.Score) {
			peer.SendMessage(&proto.MicroBlockInvMessage{
				Body: bts,
			})
		})
	}
}

func (a *RuntimeImpl) HandleInvMessage(p peer.Peer, mess *proto.MicroBlockInvMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	inv := proto.MicroBlockInv{}
	err := inv.UnmarshalBinary(mess.Body)
	if err != nil {
		zap.S().Error(err)
		return
	}

	_, ok := a.inv.Inv(inv.TotalBlockSig)
	if ok { //already exists
		return
	}

	_, ok = a.services.Peers.Connected(p)
	if !ok {
		return
	}

	a.waitingOnMicroblock = &inv.TotalBlockSig

	a.services.InvRequester.Request(p, inv.TotalBlockSig)
}

func (a *RuntimeImpl) HandleMicroBlockRequestMessage(p peer.Peer, message *proto.MicroBlockRequestMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()

	mess := proto.MicroBlockRequest{}
	err := mess.UnmarshalBinary(message.Body.(proto.Bytes))
	if err != nil {
		zap.S().Error(err)
		return
	}
	microBlock, ok := a.blocks.MicroBlock(mess.TotalBlockSig)
	if !ok {
		return
	}
	_, ok = a.services.Peers.Connected(p)
	if !ok {
		return
	}
	p.SendMessage(&proto.MicroBlockMessage{
		Body: microBlock,
	})
}

func (a *RuntimeImpl) HandleMicroBlockMessage(_ peer.Peer, message *proto.MicroBlockMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()

	microblock := &proto.MicroBlock{}

	switch t := message.Body.(type) {
	case proto.Bytes:
		err := microblock.UnmarshalBinary(t)
		if err != nil {
			zap.S().Error(err)
			return
		}
	case *proto.MicroBlock:
		microblock = t
	default:
		zap.S().Errorf("unknown *proto.MicroBlockMessage body type %T", t)
		return
	}

	zap.S().Debugf("received micro %s", microblock.Signature)

	if a.waitingOnMicroblock == nil {
		// we don't need microblocks
		return
	}

	if *a.waitingOnMicroblock != microblock.TotalResBlockSigField {
		// received microblock that we don't expect
		return
	}

	a.ngState.AddMicroblock(microblock)
	go a.services.Scheduler.Reschedule()
}

func (a *RuntimeImpl) HandleBlockMessage(_ peer.Peer, block *proto.Block) {
	zap.S().Debugf("NG State: HandleBlockMessage: New block %s", block.BlockSignature.String())
	a.ngState.AddBlock(block)
	go a.services.Scheduler.Reschedule()
}

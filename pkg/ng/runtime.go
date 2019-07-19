package ng

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type Runtime interface {
	MinedMicroblock(block *proto.MicroBlock, inv *proto.MicroBlockInv)
}

type RuntimeImpl struct {
	mu        sync.Mutex
	blocks    *MicroblockCache
	inv       *InvCache
	peers     peer_manager.PeerManager
	state     *State
	scheduler types.Scheduler

	// we send request for this microblock and waiting for it
	waitingOnMicroblock *crypto.Signature
}

func NewRuntime(peers peer_manager.PeerManager, ngState *State, scheduler types.Scheduler) *RuntimeImpl {
	return &RuntimeImpl{
		peers:     peers,
		blocks:    NewMicroblockCache(32),
		inv:       NewInvCache(32),
		state:     ngState,
		scheduler: scheduler,
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
		a.peers.EachConnected(func(peer peer.Peer, i *proto.Score) {
			peer.SendMessage(&proto.MicroBlockInvMessage{
				Body: bts,
			})
		})
	}
}

func (a *RuntimeImpl) HandleInvMessage(peerID string, mess *proto.MicroBlockInvMessage) {
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

	peer, ok := a.peers.Connected(peerID)
	if !ok {
		return
	}

	a.waitingOnMicroblock = &inv.TotalBlockSig

	peer.SendMessage(&proto.MicroBlockRequestMessage{
		Body: &proto.MicroBlockRequest{
			TotalBlockSig: inv.TotalBlockSig,
		},
	})
}

func (a *RuntimeImpl) HandleMicroBlockRequestMessage(s string, message *proto.MicroBlockRequestMessage) {
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

	peer, ok := a.peers.Connected(s)
	if !ok {
		return
	}
	peer.SendMessage(&proto.MicroBlockMessage{
		Body: microBlock,
	})
}

func (a *RuntimeImpl) HandleMicroBlockMessage(s string, message *proto.MicroBlockMessage) {
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

	if a.waitingOnMicroblock == nil {
		// we don't need microblocks
		return
	}

	if *a.waitingOnMicroblock != microblock.TotalResBlockSigField {
		// received microblock that we don't expect
		return
	}

	a.state.AddMicroblock(microblock)
	go a.scheduler.Reschedule()
}

func (a *RuntimeImpl) HandleBlockMessage(peerID string, block *proto.Block) {
	a.state.AddBlock(block)
	go a.scheduler.Reschedule()
}

package node

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util"
	"go.uber.org/zap"
)

type Config struct {
	AppName  string
	NodeName string
	Listen   string
	DeclAddr string
}

type Node struct {
	peers peer_manager.PeerManager
	state state.State
	//subscribe types.Subscribe
	declAddr  proto.TCPAddr
	bindAddr  proto.TCPAddr
	scheduler types.Scheduler
	utx       types.UtxPool
	//ng        *ng.RuntimeImpl
	services services.Services

	microblockCache *MicroblockCache

	//
	//mu sync.Mutex
}

func NewNode(services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr) *Node {
	if bindAddr.Empty() {
		bindAddr = declAddr
	}
	return &Node{
		state: services.State,
		peers: services.Peers,
		//subscribe: services.Subscribe,
		declAddr:  declAddr,
		bindAddr:  bindAddr,
		scheduler: services.Scheduler,
		utx:       services.UtxPool,
		services:  services,
	}
}

func (a *Node) State() state.State {
	return a.state
}

func (a *Node) PeerManager() peer_manager.PeerManager {
	return a.peers
}

//func (a *Node) HandleProtoMessage(mess peer.ProtoMessage) {
//	switch t := mess.Message.(type) {
//	case *proto.PeersMessage:
//		a.handlePeersMessage(mess.ID, t)
//	case *proto.GetPeersMessage:
//		a.handleGetPeersMessage(mess.ID, t)
//	case *proto.ScoreMessage:
//		//a.handleScoreMessage(mess.ID, t.Score)
//	case *proto.BlockMessage:
//		a.handleBlockMessage(mess.ID, t)
//	case *proto.GetBlockMessage:
//		a.handleBlockBySignatureMessage(mess.ID, t.BlockID)
//	case *proto.SignaturesMessage:
//		a.handleSignaturesMessage(mess.ID, t)
//	case *proto.GetSignaturesMessage:
//		a.handleGetSignaturesMessage(mess.ID, t)
//	case *proto.TransactionMessage:
//		a.handleTransactionMessage(mess.ID, t)
//	case *proto.MicroBlockInvMessage:
//		a.handleMicroblockInvMessage(mess.ID, t)
//	case *proto.MicroBlockRequestMessage:
//		a.handleMicroBlockRequestMessage(mess.ID, t)
//	case *proto.MicroBlockMessage:
//		a.handleMicroBlockMessage(mess.ID, t)
//	case *proto.PBBlockMessage:
//		a.handlePBBlockMessage(mess.ID, t)
//	case *proto.PBMicroBlockMessage:
//		a.handlePBMicroBlockMessage(mess.ID, t)
//	case *proto.PBTransactionMessage:
//		a.handlePBTransactionMessage(mess.ID, t)
//
//	default:
//		zap.S().Errorf("unknown proto Message %T", mess.Message)
//	}
//}

// TODO
func (a *Node) handlePBBlockMessage(p peer.Peer, mess *proto.PBBlockMessage) {
	//if !a.subscribe.Receive(p, mess) {
	//	b := &proto.Block{}
	//	err := b.UnmarshalFromProtobuf(mess.PBBlockBytes)
	//	if err != nil {
	//		zap.S().Debug(err)
	//		return
	//	}
	//	a.ng.HandleBlockMessage(p, b)
	//}
}

func (a *Node) handlePBMicroBlockMessage(p peer.Peer, mess *proto.PBMicroBlockMessage) {
	//a.ng.HandlePBMicroBlockMessage(p, mess)
	// TODO handle pb microblock mess
}

func (a *Node) handlePBTransactionMessage(_ peer.Peer, mess *proto.PBTransactionMessage) {
	t, err := proto.SignedTxFromProtobuf(mess.Transaction)
	if err != nil {
		zap.S().Debug(err)
		return
	}
	_ = a.utx.AddWithBytes(t, util.Dup(mess.Transaction))
}

func (a *Node) handleTransactionMessage(_ peer.Peer, mess *proto.TransactionMessage) {
	t, err := proto.BytesToTransaction(mess.Transaction, a.services.Scheme)
	if err != nil {
		zap.S().Debug(err)
		return
	}
	_ = a.utx.AddWithBytes(t, util.Dup(mess.Transaction))
}

func (a *Node) handlePeersMessage(_ peer.Peer, peers *proto.PeersMessage) {
	var prs []proto.TCPAddr
	for _, p := range peers.Peers {
		prs = append(prs, proto.NewTCPAddr(p.Addr, int(p.Port)))
	}
	err := a.peers.UpdateKnownPeers(prs)
	if err != nil {
		zap.S().Error(err)
	}
}

func (a *Node) handleGetPeersMessage(p peer.Peer, _ *proto.GetPeersMessage) {
	rs, err := a.peers.KnownPeers()
	if err != nil {
		zap.L().Error("failed got known peers", zap.Error(err))
		return
	}
	//_, ok := a.peers.Connected(p)
	//if !ok {
	//	// peer gone offline, skip
	//	return
	//}

	var out []proto.PeerInfo
	for _, r := range rs {
		out = append(out, proto.PeerInfo{
			Addr: net.IP(r.IP[:]),
			Port: uint16(r.Port),
		})
	}

	p.SendMessage(&proto.PeersMessage{Peers: out})
}

//func (a *Node) HandleInfoMessage(m peer.InfoMessage) {
//	switch t := m.Value.(type) {
//	case *peer.Connected:
//		a.handleNewConnection(t.Peer)
//	case error:
//		a.handlePeerError(m.Peer, t)
//	}
//}

//func (a *Node) HandleInfoMessage2(fsm state_fsm.FSM, m peer.InfoMessage) {
//	switch t := m.Value.(type) {
//	case *peer.Connected:
//		a.handleNewConnection(t.Peer)
//	case error:
//		a.handlePeerError(m.Peer, t)
//	}
//}

//func (a *Node) AskPeers() {
//	a.peers.AskPeers()
//}

//func (a *Node) handlePeerError(p peer.Peer, err error) {
//	zap.S().Debug(err)
//	a.peers.Suspend(p, err.Error())
//}

func (a *Node) Close() {
	a.peers.Close()
	locked := a.state.Mutex().Lock()
	a.state.Close()
	locked.Unlock()
}

//func (a *Node) handleNewConnection(p peer.Peer) {
//	err := a.peers.NewConnection(p)
//	if err != nil {
//		return
//	}
//
//	// send score to new connected
//	go func() {
//		locked := a.state.Mutex().RLock()
//		score, err := a.state.CurrentScore()
//		locked.Unlock()
//		if err != nil {
//			zap.S().Error(err)
//			return
//		}
//		p.SendMessage(&proto.ScoreMessage{
//			Score: score.Bytes(),
//		})
//	}()
//}

func (a *Node) handleBlockBySignatureMessage(p peer.Peer, sig crypto.Signature) {
	locked := a.state.Mutex().RLock()
	block, err := a.state.Block(sig)
	locked.Unlock()
	if err != nil {
		zap.S().Error(err)
		return
	}
	bm, err := proto.MessageByBlock(block, a.services.Scheme)
	if err != nil {
		zap.S().Error(err)
		return
	}
	p.SendMessage(bm)
}

//func (a *Node) handleScoreMessage(p peer.Peer, score []byte) {
//	b := new(big.Int)
//	b.SetBytes(score)
//	a.peers.UpdateScore(p, b)
//
//	go func() {
//		<-time.After(4 * time.Second)
//		a.sync.Sync()
//	}()
//
//}

//func (a *Node) handleBlockMessage(p peer.Peer, mess *proto.BlockMessage) {
//if !a.subscribe.Receive(p, mess) {
//	b := &proto.Block{}
//	err := b.UnmarshalBinary(mess.BlockBytes, a.services.Scheme)
//	if err != nil {
//		zap.S().Debug(err)
//		return
//	}
//	a.ng.HandleBlockMessage(p, b)
//}
//}

func (a *Node) handleGetSignaturesMessage(p peer.Peer, mess *proto.GetSignaturesMessage) {
	locked := a.state.Mutex().RLock()
	defer locked.Unlock()
	for _, sig := range mess.Blocks {
		block, err := a.state.Header(sig)
		if err != nil {
			continue
		}
		a.sendSignatures(block, p)
		return
	}
}

//func sendSignatures(p Peer, storage storage.State, sigs []crypto.Signature) {
//	for _, sig := range sigs {
//		block, err := storage.Header(sig)
//		if err != nil {
//			continue
//		}
//		_sendSignatures(block, storage, p)
//		break
//	}
//	return fsm, nil, nil
//}

func (a *Node) sendSignatures(block *proto.BlockHeader, p peer.Peer) {
	height, err := a.state.BlockIDToHeight(block.BlockSignature)
	if err != nil {
		zap.S().Error(err)
		return
	}

	var out []crypto.Signature
	out = append(out, block.BlockSignature)

	for i := 1; i < 101; i++ {
		b, err := a.state.HeaderByHeight(height + uint64(i))
		if err != nil {
			break
		}
		out = append(out, b.BlockSignature)
	}

	// if we put smth except first block
	if len(out) > 1 {
		p.SendMessage(&proto.SignaturesMessage{
			Signatures: out,
		})
	}
}

//func (a *Node) handleMicroblockInvMessage(p peer.Peer, mess *proto.MicroBlockInvMessage) {
//	a.ng.HandleInvMessage(p, mess)
//}

// TODO implement
func (a *Node) handleMicroBlockRequestMessage(p peer.Peer, mess *proto.MicroBlockRequestMessage) {
	//micro, ok := a.microblockCache.Get(mess.Body)
	//if ok {
	panic("not implemented")
	//p.SendMessage(&proto.MicroBlockMessage{})
	//}
}

func (a *Node) SpawnOutgoingConnections(ctx context.Context) {
	a.peers.SpawnOutgoingConnections(ctx)
}

func (a *Node) SpawnOutgoingConnection(ctx context.Context, addr proto.TCPAddr) error {
	return a.peers.Connect(ctx, addr)
}

func (a *Node) Serve(ctx context.Context) error {
	// if empty declared address, listen on port doesn't make sense
	if a.declAddr.Empty() {
		return nil
	}

	if a.bindAddr.Empty() {
		return nil
	}

	zap.S().Info("start listening on ", a.bindAddr.String())
	l, err := net.Listen("tcp", a.bindAddr.String())
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		go func() {
			if err := a.peers.SpawnIncomingConnection(ctx, conn); err != nil {
				zap.S().Error(err)
				return
			}
		}()
	}
}

//func (a *Node) handleMicroBlockMessage(p peer.Peer, message *proto.MicroBlockMessage) {
//	a.ng.HandleMicroBlockMessage(p, message)
//}

//func (a *Node) handleSignaturesMessage(p peer.Peer, message *proto.SignaturesMessage) {
//	a.subscribe.Receive(p, message)
//}

func (a *Node) Run(ctx context.Context, p peer.Parent) {
	//go a.sync.Run(ctx)

	go func() {
		for {
			a.SpawnOutgoingConnections(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
		}
	}()

	//go func() {
	//	select {
	//	case <-time.After(10 * time.Second):
	//	case <-ctx.Done():
	//		return
	//	}
	//
	//	a.AskPeers()
	//
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		case <-time.After(4 * time.Minute):
	//			a.AskPeers()
	//		}
	//	}
	//}()

	// info messages
	//go func() {
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		case m := <-p.InfoCh:
	//			n.HandleInfoMessage(m)
	//		}
	//	}
	//}()

	go func() {
		if err := a.Serve(ctx); err != nil {
			return
		}
	}()

	tasksCh := make(chan state_fsm.AsyncTask, 10)

	// TODO hardcode
	outDatePeriod := 3600 /* hour */ * 4 * 1000 /* milliseconds */
	fsm, async, err := state_fsm.NewFsm(
		a.state, a.peers, a.services.Time, uint64(outDatePeriod), a.services.Scheme,
		NewInvRequester(),
		state_fsm.BlockCreaterImpl{},
		blocks_applier.NewBlocksApplier())
	if err != nil {
		zap.S().Error(err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-tasksCh:
			fsm, async, err = fsm.Task(task)
		case m := <-p.InfoCh:
			//n.HandleInfoMessage(m)
			switch t := m.Value.(type) {
			case *peer.Connected:
				//TODO async
				fsm, async, err = fsm.NewPeer(t.Peer)
				//if err != nil {
				//	zap.S().Error(err)
				//}
				//zap.S().Debugf("fsm %T", fsm)
				//a.handleNewConnection(t.Peer)
			case error:
				// TODO handle error
				zap.S().Error(m.Peer, t)
				//a.handlePeerError(m.Peer, t)
				fsm, async, err = fsm.PeerError(m.Peer, t)
			}
		case mess := <-p.MessageCh:
			zap.S().Debugf("received proto Message %T", mess.Message)
			//a.services.LoggableRunner.Named(fmt.Sprintf("Node.Run.Handler.%T", m.Message), func() {
			//a.HandleProtoMessage(m)
			switch t := mess.Message.(type) {
			//case *proto.PeersMessage:
			//	a.handlePeersMessage(mess.ID, t)
			case *proto.GetPeersMessage:
				a.handleGetPeersMessage(mess.ID, t)
				//fsm, async, err = fsm.GetPeers(mess.ID)
			case *proto.ScoreMessage:
				//a.handleScoreMessage(mess.ID, t.Score)
				b := new(big.Int)
				b.SetBytes(t.Score)
				fsm, async, err = fsm.Score(mess.ID, b)
			case *proto.BlockMessage:
				b := &proto.Block{}
				err2 := b.UnmarshalBinary(t.BlockBytes, a.services.Scheme)
				if err2 != nil {
					zap.S().Debug(err2)
					continue
				}
				fsm, async, err = fsm.Block(mess.ID, b)
			case *proto.GetBlockMessage:
				a.handleBlockBySignatureMessage(mess.ID, t.BlockID)
			case *proto.SignaturesMessage:
				//a.handleSignaturesMessage(mess.ID, t)
				fsm, async, err = fsm.Signatures(mess.ID, t.Signatures)
				//if err != nil {
				//	zap.S().Error(err)
				//}
				//zap.S().Debugf("fsm %T", fsm)

			case *proto.GetSignaturesMessage:
				a.handleGetSignaturesMessage(mess.ID, t)
			//case *proto.TransactionMessage:
			//	a.handleTransactionMessage(mess.ID, t)
			case *proto.MicroBlockInvMessage:
				//a.handleMicroblockInvMessage(mess.ID, t)
				//t.UnmarshalBinary()
				//t.Body
				inv := &proto.MicroBlockInv{}
				err2 := inv.UnmarshalBinary(t.Body)
				if err2 != nil {
					zap.S().Error(err2)
					continue
				}
				fsm, async, err = fsm.MicroBlockInv(mess.ID, inv)

			case *proto.MicroBlockRequestMessage:
				a.handleMicroBlockRequestMessage(mess.ID, t)
			case *proto.MicroBlockMessage:
				//a.handleMicroBlockMessage(mess.ID, t)

				micro := &proto.MicroBlock{}
				err2 := micro.UnmarshalBinary(t.Body, a.services.Scheme)
				if err2 != nil {
					zap.S().Error(err2)
					continue
				}
				fsm, async, err = fsm.MicroBlock(mess.ID, micro)

			//case *proto.PBBlockMessage:
			//	a.handlePBBlockMessage(mess.ID, t)
			//case *proto.PBMicroBlockMessage:
			//	a.handlePBMicroBlockMessage(mess.ID, t)
			//case *proto.PBTransactionMessage:
			//	a.handlePBTransactionMessage(mess.ID, t)

			default:
				zap.S().Errorf("unknown proto Message %T", mess.Message)
			}
			//})
		}
		if err != nil {
			zap.S().Error(err)
		}
		for _, t := range async {
			a.services.LoggableRunner.Named(fmt.Sprintf("Async Task %T", t), func() {
				err := t.Run(ctx, tasksCh)
				if err != nil {
					zap.S().Errorf("Async Task %T, error %q", t, err)
				}
			})
		}
		zap.S().Debugf("fsm %T", fsm)
	}
}

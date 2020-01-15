package node

import (
	"context"
	"math/big"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
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
	peers            peer_manager.PeerManager
	state            state.State
	subscribe        types.Subscribe
	sync             types.StateSync
	declAddr         proto.TCPAddr
	bindAddr         proto.TCPAddr
	scheduler        types.Scheduler
	minerInterrupter types.MinerInterrupter
	utx              types.UtxPool
	ng               *ng.RuntimeImpl
}

func NewNode(services services.Services, declAddr proto.TCPAddr, bindAddr proto.TCPAddr, ng *ng.RuntimeImpl, interrupter types.MinerInterrupter, sync types.StateSync) *Node {
	if bindAddr.Empty() {
		bindAddr = declAddr
	}
	return &Node{
		state:            services.State,
		peers:            services.Peers,
		subscribe:        services.Subscribe,
		sync:             sync,
		declAddr:         declAddr,
		bindAddr:         bindAddr,
		scheduler:        services.Scheduler,
		minerInterrupter: interrupter,
		utx:              services.UtxPool,
		ng:               ng,
	}
}

func (a *Node) State() state.State {
	return a.state
}

func (a *Node) PeerManager() peer_manager.PeerManager {
	return a.peers
}

func (a *Node) HandleProtoMessage(mess peer.ProtoMessage) {
	switch t := mess.Message.(type) {
	case *proto.PeersMessage:
		a.handlePeersMessage(mess.ID, t)
	case *proto.GetPeersMessage:
		a.handleGetPeersMessage(mess.ID, t)
	case *proto.ScoreMessage:
		a.handleScoreMessage(mess.ID, t.Score)
	case *proto.BlockMessage:
		a.handleBlockMessage(mess.ID, t)
	case *proto.GetBlockMessage:
		a.handleBlockBySignatureMessage(mess.ID, t.BlockID)
	case *proto.SignaturesMessage:
		a.handleSignaturesMessage(mess.ID, t)
	case *proto.GetSignaturesMessage:
		a.handleGetSignaturesMessage(mess.ID, t)
	case *proto.TransactionMessage:
		a.handleTransactionMessage(mess.ID, t)
	case *proto.MicroBlockInvMessage:
		a.handleMicroblockInvMessage(mess.ID, t)
	case *proto.MicroBlockRequestMessage:
		a.handleMicroBlockRequestMessage(mess.ID, t)
	case *proto.MicroBlockMessage:
		a.handleMicroBlockMessage(mess.ID, t)

	default:
		zap.S().Errorf("unknown proto Message %+v", mess.Message)
	}
}

func (a *Node) handleTransactionMessage(_ peer.Peer, mess *proto.TransactionMessage) {
	t, err := proto.BytesToTransaction(mess.Transaction)
	if err != nil {
		zap.S().Debug(err)
		return
	}
	a.utx.AddWithBytes(t, util.Dup(mess.Transaction))
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

func (a *Node) handleGetPeersMessage(p peer.Peer, m *proto.GetPeersMessage) {
	rs, err := a.peers.KnownPeers()
	if err != nil {
		zap.L().Error("failed got known peers", zap.Error(err))
		return
	}
	_, ok := a.peers.Connected(p)
	if !ok {
		// peer gone offline, skip
		return
	}

	var out []proto.PeerInfo
	for _, r := range rs {
		out = append(out, proto.PeerInfo{
			Addr: net.IP(r.IP[:]),
			Port: uint16(r.Port),
		})
	}

	p.SendMessage(&proto.PeersMessage{Peers: out})
}

func (a *Node) HandleInfoMessage(m peer.InfoMessage) {
	switch t := m.Value.(type) {
	case *peer.Connected:
		a.handleNewConnection(t.Peer)
	case error:
		a.handlePeerError(m.Peer, t)
	}
}

func (a *Node) AskPeers() {
	a.peers.AskPeers()
}

func (a *Node) handlePeerError(p peer.Peer, err error) {
	zap.S().Debug(err)
	a.peers.Disconnect(p)
}

func (a *Node) Close() {
	a.peers.Close()
	a.sync.Close()
	locked := a.state.Mutex().Lock()
	a.state.Close()
	locked.Unlock()
}

func (a *Node) handleNewConnection(peer peer.Peer) {
	_, connected := a.peers.Connected(peer)
	if connected {
		peer.Close()
		return
	}
	if a.peers.IsSuspended(peer) {
		peer.Close()
		return
	}
	a.peers.AddConnected(peer)

	// send score to new connected
	go func() {
		score, err := a.state.CurrentScore()
		if err != nil {
			zap.S().Error(err)
			return
		}
		peer.SendMessage(&proto.ScoreMessage{
			Score: score.Bytes(),
		})
	}()
}

func (a *Node) handleBlockBySignatureMessage(p peer.Peer, sig crypto.Signature) {
	block, err := a.state.Block(sig)
	if err != nil {
		zap.S().Error(err)
		return
	}
	bts, err := block.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	bm := proto.BlockMessage{
		BlockBytes: bts,
	}
	p.SendMessage(&bm)
}

func (a *Node) handleScoreMessage(p peer.Peer, score []byte) {
	b := new(big.Int)
	b.SetBytes(score)
	a.peers.UpdateScore(p, b)

	go func() {
		<-time.After(4 * time.Second)
		a.sync.Sync()
	}()

}

func (a *Node) handleBlockMessage(p peer.Peer, mess *proto.BlockMessage) {
	if !a.subscribe.Receive(p, mess) {
		b := &proto.Block{}
		err := b.UnmarshalBinary(mess.BlockBytes)
		if err != nil {
			zap.S().Debug(err)
			return
		}
		a.ng.HandleBlockMessage(p, b)
	}
}

func (a *Node) handleGetSignaturesMessage(p peer.Peer, mess *proto.GetSignaturesMessage) {
	for _, sig := range mess.Blocks {
		block, err := a.state.Block(sig)
		if err != nil {
			continue
		}
		if block.BlockSignature != sig {
			panic("signature error")
		}
		sendSignatures(block, a.state, p)
		return
	}
}

func (a *Node) handleMicroblockInvMessage(p peer.Peer, mess *proto.MicroBlockInvMessage) {
	a.ng.HandleInvMessage(p, mess)
}

func (a *Node) handleMicroBlockRequestMessage(p peer.Peer, mess *proto.MicroBlockRequestMessage) {
	a.ng.HandleMicroBlockRequestMessage(p, mess)
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

func (a *Node) handleMicroBlockMessage(p peer.Peer, message *proto.MicroBlockMessage) {
	a.ng.HandleMicroBlockMessage(p, message)
}

func (a *Node) handleSignaturesMessage(p peer.Peer, message *proto.SignaturesMessage) {
	a.subscribe.Receive(p, message)
}

func RunNode(ctx context.Context, n *Node, p peer.Parent) {
	go n.sync.Run(ctx)

	go func() {
		for {
			n.SpawnOutgoingConnections(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
		}
	}()

	go func() {
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return
		}

		n.AskPeers()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(4 * time.Minute):
				n.AskPeers()
			}
		}
	}()

	// info messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-p.InfoCh:
				n.HandleInfoMessage(m)
			}
		}
	}()

	go func() {
		if err := n.Serve(ctx); err != nil {
			return
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-p.MessageCh:
			n.HandleProtoMessage(m)
		}
	}
}

type Signatures struct {
	signatures []crypto.Signature
	unique     map[crypto.Signature]struct{}
}

func (a *Signatures) Signatures() []crypto.Signature {
	return a.signatures
}

func NewSignatures(signatures ...crypto.Signature) *Signatures {
	unique := make(map[crypto.Signature]struct{})
	for _, v := range signatures {
		unique[v] = struct{}{}
	}

	return &Signatures{
		signatures: signatures,
		unique:     unique,
	}
}

func (a *Signatures) Exists(sig crypto.Signature) bool {
	_, ok := a.unique[sig]
	return ok
}

func (a *Signatures) Revert() *Signatures {
	out := make([]crypto.Signature, len(a.signatures))
	for k, v := range a.signatures {
		out[len(a.signatures)-1-k] = v
	}
	return NewSignatures(out...)
}

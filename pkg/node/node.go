package node

import (
	"context"
	"math/big"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	peerManager      peer_manager.PeerManager
	state            state.State
	subscribe        *Subscribe
	sync             *StateSync
	declAddr         proto.TCPAddr
	scheduler        types.Scheduler
	minerInterrupter types.MinerInterrupter
	utx              *utxpool.Utx
	ng               *ng.RuntimeImpl
}

func NewNode(stateManager state.State, peerManager peer_manager.PeerManager, declAddr proto.TCPAddr, scheduler types.Scheduler, minerInterrupter types.MinerInterrupter, utx *utxpool.Utx, ng *ng.RuntimeImpl) *Node {
	s := NewSubscribeService()
	return &Node{
		state:            stateManager,
		peerManager:      peerManager,
		subscribe:        s,
		sync:             NewStateSync(stateManager, peerManager, s, scheduler, minerInterrupter),
		declAddr:         declAddr,
		scheduler:        scheduler,
		minerInterrupter: minerInterrupter,
		utx:              utx,
		ng:               ng,
	}
}

func (a *Node) State() state.State {
	return a.state
}

func (a *Node) PeerManager() peer_manager.PeerManager {
	return a.peerManager
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

func (a *Node) handleTransactionMessage(id string, mess *proto.TransactionMessage) {
	t, err := proto.BytesToTransaction(mess.Transaction)
	if err != nil {
		zap.S().Debug(err)
		return
	}
	a.utx.Add(t)
}

func (a *Node) handlePeersMessage(id string, peers *proto.PeersMessage) {
	var prs []proto.TCPAddr
	for _, p := range peers.Peers {
		prs = append(prs, proto.NewTCPAddr(p.Addr, int(p.Port)))
	}

	err := a.peerManager.UpdateKnownPeers(prs)
	if err != nil {
		zap.S().Error(err)
	}
}

func (a *Node) handleGetPeersMessage(id string, m *proto.GetPeersMessage) {
	rs, err := a.peerManager.KnownPeers()
	if err != nil {
		zap.L().Error("failed got known peers", zap.Error(err))
		return
	}
	p, ok := a.peerManager.Connected(id)
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
		a.handlePeerError(m.ID, t)
	}
}

func (a *Node) AskPeers() {
	a.peerManager.AskPeers()
}

func (a *Node) handlePeerError(id string, err error) {
	zap.S().Debug(err)
	a.peerManager.Disconnect(id)
}

func (a *Node) Close() {
	a.peerManager.Close()
	m := a.state.Mutex()
	locked := m.Lock()
	a.state.Close()
	locked.Unlock()
	a.sync.Close()
}

func (a *Node) handleNewConnection(peer peer.Peer) {
	_, connected := a.peerManager.Connected(peer.ID())
	if connected {
		peer.Close()
		return
	}

	if a.peerManager.Banned(peer.ID()) {
		peer.Close()
		return
	}

	a.peerManager.AddConnected(peer)

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

func (a *Node) handleBlockBySignatureMessage(peer string, sig crypto.Signature) {
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

	p, ok := a.peerManager.Connected(peer)
	if ok {
		p.SendMessage(&bm)
	}
}

func (a *Node) handleScoreMessage(peerID string, score []byte) {
	b := new(big.Int)
	b.SetBytes(score)
	a.peerManager.UpdateScore(peerID, b)

	go func() {
		<-time.After(4 * time.Second)
		a.sync.Sync()
	}()

}

func (a *Node) handleBlockMessage(peerID string, mess *proto.BlockMessage) {
	defer util.TimeTrack(time.Now(), "handleBlockMessage")
	if !a.subscribe.Receive(peerID, mess) {
		b := &proto.Block{}
		err := b.UnmarshalBinary(mess.BlockBytes)
		if err != nil {
			zap.S().Debug(err)
			return
		}
		a.ng.HandleBlockMessage(peerID, b)
	}
}

func (a *Node) handleGetSignaturesMessage(peerID string, mess *proto.GetSignaturesMessage) {
	defer util.TimeTrack(time.Now(), "handleGetSignaturesMessage")
	p, ok := a.peerManager.Connected(peerID)
	if !ok {
		return
	}

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

func (a *Node) handleMicroblockInvMessage(peerID string, mess *proto.MicroBlockInvMessage) {
	a.ng.HandleInvMessage(peerID, mess)
}

func (a *Node) handleMicroBlockRequestMessage(peerID string, mess *proto.MicroBlockRequestMessage) {
	a.ng.HandleMicroBlockRequestMessage(peerID, mess)
}

func (a *Node) SpawnOutgoingConnections(ctx context.Context) {
	a.peerManager.SpawnOutgoingConnections(ctx)
}

func (a *Node) SpawnOutgoingConnection(ctx context.Context, addr proto.TCPAddr) error {
	return a.peerManager.Connect(ctx, addr)
}

func (a *Node) Serve(ctx context.Context) error {
	if a.declAddr.Empty() {
		return nil
	}

	l, err := net.Listen("tcp", a.declAddr.String())
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			zap.S().Error(err)
			continue
		}

		go a.peerManager.SpawnIncomingConnection(ctx, conn)
	}
}

func (a *Node) handleMicroBlockMessage(s string, message *proto.MicroBlockMessage) {
	a.ng.HandleMicroBlockMessage(s, message)
}

func (a *Node) handleSignaturesMessage(s string, message *proto.SignaturesMessage) {
	a.subscribe.Receive(s, message)
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
		n.Serve(ctx)
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

func NewSignatures(signatures []crypto.Signature) *Signatures {
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

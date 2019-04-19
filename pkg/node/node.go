package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util"
	"go.uber.org/zap"
	"math/big"
	"net"
	"reflect"
	"time"
)

type Config struct {
	AppName  string
	NodeName string
	Listen   string
	DeclAddr string
}

type Node struct {
	peerManager  PeerManager
	stateManager state.State
	subscribe    *Subscribe
	sync         *StateSync
	declAddr     proto.TCPAddr
}

func NewNode(stateManager state.State, peerManager PeerManager, declAddr proto.TCPAddr) *Node {
	s := NewSubscribeService()
	return &Node{
		stateManager: stateManager,
		peerManager:  peerManager,
		subscribe:    s,
		sync:         NewStateSync(stateManager, peerManager, s),
		declAddr:     declAddr,
	}
}

func (a *Node) HandleProtoMessage(mess peer.ProtoMessage) {

	zap.S().Info("arrived ", reflect.TypeOf(mess.Message))

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
		//a.handleSignaturesMessage()
		a.subscribe.Receive(mess.ID, t)
	case *proto.GetSignaturesMessage:
		a.handleGetSignaturesMessage(mess.ID, t)
	case *proto.TransactionMessage:
	// nothing to do with transactions
	// no utx pool exists
	case *proto.MicroBlockMessage:
	// skip to better times

	default:
		zap.S().Errorf("unknown proto Message %+v", mess.Message)
	}
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
	a.stateManager.Close()
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
		score, err := a.stateManager.CurrentScore()
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
	block, err := a.stateManager.Block(sig)
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

// called every n seconds, handle change runtime state
func (a *Node) SyncState() {
	for {
		err := a.sync.Sync()
		if err != nil {
			zap.S().Error(err)
			// wait only on errors
			time.Sleep(5 * time.Second)
		}
	}
}

func (a *Node) handleScoreMessage(peerID string, score []byte) {
	b := new(big.Int)
	b.SetBytes(score)
	a.peerManager.UpdateScore(peerID, b)
}

func (a *Node) handleBlockMessage(peerID string, mess proto.Message) {
	defer util.TimeTrack(time.Now(), "handleBlockMessage")
	a.subscribe.Receive(peerID, mess)
}

func (a *Node) handleGetSignaturesMessage(peerID string, mess *proto.GetSignaturesMessage) {
	defer util.TimeTrack(time.Now(), "handleGetSignaturesMessage")
	p, ok := a.peerManager.Connected(peerID)
	if !ok {
		return
	}

	for _, sig := range mess.Blocks {

		block, err := a.stateManager.Block(sig)
		if err != nil {
			continue
		}

		if block.BlockSignature != sig {
			panic("signature error")
		}

		sendSignatures(block, a.stateManager, p)
		return
	}
}

func (a *Node) SpawnOutgoingConnections(ctx context.Context) {
	a.peerManager.SpawnOutgoingConnections(ctx)
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

func RunNode(ctx context.Context, n *Node, p peer.Parent) {
	go n.SyncState()

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
		<-time.After(10 * time.Second)
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

package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"math/big"
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
}

func NewNode(stateManager state.State, peerManager PeerManager) *Node {
	s := NewSubscribeService()
	return &Node{
		stateManager: stateManager,
		peerManager:  peerManager,
		subscribe:    s,
		sync:         NewStateSync(stateManager, peerManager, s),
	}
}

func (a *Node) HandleProtoMessage(mess peer.ProtoMessage) {

	zap.S().Info("arrived ", reflect.TypeOf(mess.Message))

	switch t := mess.Message.(type) {
	case *proto.PeersMessage:
		a.handlePeersMessage(mess.ID, t)
	case *proto.GetPeersMessage:
		a.handleGetPeersMessage(mess.ID, t)
	case *proto.GetBlockMessage:
		a.handleBlockBySignatureMessage(mess.ID, t.BlockID)
	case *proto.ScoreMessage:
		a.handleScoreMessage(mess.ID, t.Score)
	case *proto.BlockMessage:
		a.handleBlockMessage(mess.ID, t)
	case *proto.SignaturesMessage:
		//a.handleSignaturesMessage()
		a.subscribe.Receive(mess.ID, t)
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
	a.peerManager.UpdateKnownPeers(peers.Peers)
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
	p.SendMessage(&proto.PeersMessage{Peers: rs})
}

func (a *Node) HandleInfoMessage(m peer.InfoMessage) {
	switch t := m.Value.(type) {
	case *peers.Connected:
		a.handleNewConnection(t.Peer)
	}
}

// TODO implement
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
	// nothing to do
	// TODO check is any work required

	block := proto.Block{}
	err := block.UnmarshalBinary(mess.(*proto.BlockMessage).BlockBytes)
	if err != nil {
		zap.S().Error(err)
	}

	rs, _ := proto.BlockGetSignature(mess.(*proto.BlockMessage).BlockBytes)
	//zap.S().Infof("%+v", block)
	zap.S().Infof("proto.BlockGetSignature %+v", rs)

	a.subscribe.Receive(peerID, mess)

}

//func RunIncomeConnectionsServer(ctx context.Context, n *Node, c Config, s PeerSpawner) {
//	l, err := net.Listen("tcp", c.Listen)
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	for {
//		c, err := l.Accept()
//		if err != nil {
//			zap.S().Error(err)
//			continue
//		}
//
//		go s.SpawnIncoming(ctx, c)
//	}
//}

func RunNode(ctx context.Context, n *Node, p peer.Parent) {

	go n.SyncState()

	// info messages
	go func() {
		select {
		case <-ctx.Done():
			return
		case m := <-p.InfoCh:
			n.HandleInfoMessage(m)
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

type BlockSignatures struct {
	signatures []crypto.Signature
	unique     map[crypto.Signature]struct{}
}

func (a *BlockSignatures) Signatures() []crypto.Signature {
	return a.signatures
}

func NewBlockSignatures(signatures []crypto.Signature) *BlockSignatures {
	unique := make(map[crypto.Signature]struct{})
	for _, v := range signatures {
		unique[v] = struct{}{}
	}

	return &BlockSignatures{
		signatures: signatures,
		unique:     unique,
	}
}

func (a *BlockSignatures) Exists(sig crypto.Signature) bool {
	_, ok := a.unique[sig]
	return ok
}

//type Runtime struct {
//
//}

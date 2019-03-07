package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
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
}

func NewNode(stateManager state.State, peerManager PeerManager) *Node {
	return &Node{
		stateManager: stateManager,
		peerManager:  peerManager,
	}
}

func (a *Node) HandleProtoMessage(mess peer.ProtoMessage) {
	switch t := mess.Message.(type) {
	case *proto.GetBlockMessage:
		a.handleBlockBySignatureMessage(mess.ID, t.BlockID)
	case *proto.ScoreMessage:
		a.handleScoreMessage(mess.ID, t.Score)
	default:
		zap.S().Error("unknown proto Message", mess)
	}
}

func (a *Node) HandleInfoMessage(m peer.InfoMessage) {
	switch t := m.Value.(type) {
	case *peers.Connected:
		a.handleNewConnection(t.Peer)
	}
}

// TODO implement
func (a *Node) Close() {
	panic("implement me")
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
func (a *Node) Tick() {
	for {
		p, score, ok := a.peerManager.PeerWithHighestScore()
		if !ok {
			// no peers, skip
			return
		}

		if score == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		// TODO check if we have highest score

		p.SendMessage(&proto.GetSignaturesMessage{})

		messCh, unsubscribe := a.subscribe.Subscribe(p, &proto.SignaturesMessage{})

		var mess *proto.SignaturesMessage

		select {
		case <-time.After(15 * time.Second):
		// TODO handle timeout
		case received := <-messCh:
			//a.subscribe.Unsubscribe(p, &proto.SignaturesMessage{})
			unsubscribe()
			mess = received.(*proto.SignaturesMessage)
		}

		blockSignatures := BlockSignatures{}

		applyBlock(mess, blockSignatures, p, a)

		//?, ? := a.findMaxCommonBlock(mess.Signatures)

		//for _, i := range mess.Signatures {
		//}

		//if err != nil {
		//	if err == TimeoutErr {
		//		// TODO handle timeout
		//	}
		//}

		//ask.Subscribe(15*time.Second)
		//
		//a.subscribe.Clear(ask)
		//
		//if ask.Timeout() {
		//	// TODO handle timeout
		//}
		//
		//m := ask.Get().(*proto.SignaturesMessage{})

	}
}

func (a *Node) handleScoreMessage(peerID string, score []byte) {
	zap.S().Info("got score messge, bytes ", score)
}

func applyBlock(mess *proto.SignaturesMessage, blockSignatures BlockSignatures, p peer.Peer, a *Node) {
	subscribeCh, unsubscribe := a.subscribe.Subscribe(p, &proto.BlockMessage{})
	defer unsubscribe()
	for _, sig := range mess.Signatures {
		if !blockSignatures.Exists(sig) {
			p.SendMessage(&proto.GetBlockMessage{BlockID: sig})

			// wait for block with expected signature
			timeout := time.After(30 * time.Second)
			for {
				select {
				case <-timeout:
				// TODO HANDLE timeout

				case blockMessage := <-subscribeCh:
					bts := blockMessage.(*proto.BlockMessage).BlockBytes
					blockSignature, err := proto.BlockGetSignature(bts)
					if err != nil {
						zap.S().Error(err)
						continue
					}

					if blockSignature != sig {
						continue
					}

					err = a.stateManager.AddBlock(bts)
					if err != nil {
						// TODO handle error
					}
					break
				}
			}
		}
	}
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

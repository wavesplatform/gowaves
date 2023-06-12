package network

import (
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type InfoMessage interface{}

type Disconnected struct {
	Peer peer.Peer
}

type Connected struct {
	Peer peer.Peer
}

type BestPeerLost struct {
	Peer peer.Peer
}

type BestPeer struct {
	Peer peer.Peer
}

type Network struct {
	InfoCh        chan peer.InfoMessage
	NetworkInfoCh chan InfoMessage

	peers         peer_manager.PeerManager
	storage       state.State
	minPeerMining int
}

func NewNetwork(services services.Services, p peer.Parent, networkMsgCh chan InfoMessage) Network {
	return Network{
		InfoCh:        p.InfoCh,
		NetworkInfoCh: networkMsgCh,
		peers:         services.Peers,
		storage:       services.State,
		minPeerMining: services.MinPeersMining,
	}
}

func sendScore(p peer.Peer, storage state.State) {
	curScore, err := storage.CurrentScore()
	if err != nil {
		zap.S().Errorf("Failed to send current score to peer %q: %v", p.RemoteAddr().String(), err)
		return
	}

	bts := curScore.Bytes()
	p.SendMessage(&proto.ScoreMessage{Score: bts})
}

func (n *Network) Run() {
	for {
		select {
		case m := <-n.InfoCh:
			switch t := m.Value.(type) {
			case *peer.Connected:
				err := n.peers.NewConnection(t.Peer)
				if err != nil {
					zap.S().Debugf("Established connection with %s peer '%s': %s", t.Peer.Direction(), t.Peer.ID(), err)
					continue
				}
				if n.peers.ConnectedCount() == n.minPeerMining {
					n.NetworkInfoCh <- Connected{t.Peer}
				}
				sendScore(t.Peer, n.storage)
				newPeerScore, err := n.peers.Score(t.Peer)
				if err != nil {
					zap.S().Warnf("Failed to get score of new peer '%s': %s", t.Peer.ID(), err)
					continue
				}
				maxScorePeer, err := n.peers.GetPeerWithMaxScore()
				if err != nil {
					zap.S().Debugf("Failed to get peer with max score %s", err)
					continue
				}

				maxScore, err := n.peers.Score(maxScorePeer)
				if err != nil {
					zap.S().Warnf("Failed to get score of peer '%s': %s", t.Peer.ID(), err)
					continue
				}
				if !(maxScorePeer != t.Peer && maxScore == newPeerScore) {
					n.NetworkInfoCh <- BestPeer{Peer: t.Peer}
				}

			case *peer.InternalErr:
				n.peers.Disconnect(m.Peer)
				if n.peers.ConnectedCount() < n.minPeerMining {
					n.NetworkInfoCh <- Disconnected{Peer: m.Peer}
				}
				maxScorePeer, err := n.peers.GetPeerWithMaxScore()
				if err != nil {
					zap.S().Debugf("Failed to get peer with max score %s", err)
					continue
				}
				if maxScorePeer == m.Peer {
					n.NetworkInfoCh <- BestPeerLost{Peer: m.Peer}
				}
			default:
				zap.S().Warnf("[%s] Unknown info message '%T'", m)
			}
		}
	}
}

package network

import (
	"go.uber.org/zap"
	"time"

	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type InfoMessage interface{}

type Disconnected struct {
	Peer peer.Peer
}

type StopMining struct{}

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
	InfoCh        <-chan peer.InfoMessage
	NetworkInfoCh chan InfoMessage

	peers         peer_manager.PeerManager
	storage       state.State
	minPeerMining int
	obsolescence  time.Duration
}

func NewNetwork(services services.Services, p peer.Parent) Network {
	return Network{
		InfoCh:        p.InfoCh,
		NetworkInfoCh: make(chan InfoMessage, 100),
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

func (n *Network) isNewPeerHasMaxScore(p peer.Peer) bool {
	newPeerScore, err := n.peers.Score(p)
	if err != nil {
		zap.S().Warnf("Failed to get score of new peer '%s': %s", p.ID(), err)
		return false
	}
	maxScorePeer, err := n.peers.GetPeerWithMaxScore()
	if err != nil {
		zap.S().Debugf("Failed to get peer with max score %s", err)
		return false
	}

	maxScore, err := n.peers.Score(maxScorePeer)
	if err != nil {
		zap.S().Warnf("Failed to get score of peer '%s': %s", maxScorePeer.ID(), err)
		return false
	}
	return !(maxScorePeer != p && maxScore == newPeerScore)
}

func (n *Network) isTimeToSwitchPeerWithMaxScore() bool {
	now := time.Now()
	obsolescenceTime := now.Add(-n.obsolescence)
	lastBlock := n.storage.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	return !obsolescenceTime.After(lastBlockTime)
}

func (n *Network) Run() {
	for {
		m := <-n.InfoCh
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

			if n.isNewPeerHasMaxScore(t.Peer) && n.isTimeToSwitchPeerWithMaxScore() {
				n.NetworkInfoCh <- BestPeer{Peer: t.Peer}
			}

		case *peer.InternalErr:
			n.peers.Disconnect(m.Peer)
			if n.peers.ConnectedCount() < n.minPeerMining {
				n.NetworkInfoCh <- StopMining{}
			}
			n.NetworkInfoCh <- Disconnected{Peer: m.Peer}
		default:
			zap.S().Warnf("[%s] Unknown info message '%T'", m)
		}
	}
}

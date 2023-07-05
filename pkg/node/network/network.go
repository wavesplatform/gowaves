package network

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type InfoMessage interface{}

type StopSync struct{}

type StopMining struct{}

type StartMining struct{}

type ChangeSyncPeer struct {
	Peer peer.Peer
}

type SyncPeer struct {
	m    sync.Mutex
	peer peer.Peer
}

func (s *SyncPeer) SetPeer(peer peer.Peer) {
	s.m.Lock()
	defer s.m.Unlock()
	s.peer = peer
}

func (s *SyncPeer) GetPeer() peer.Peer {
	s.m.Lock()
	defer s.m.Unlock()
	return s.peer
}

type Network struct {
	InfoCh        <-chan peer.InfoMessage
	NetworkInfoCh chan InfoMessage
	SyncPeer      SyncPeer

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
				n.NetworkInfoCh <- StartMining{}
			}
			sendScore(t.Peer, n.storage)

			if n.isTimeToSwitchPeerWithMaxScore() {
				// Node is getting close to the top of the blockchain, it's time to switch on a node with the highest
				// score every time it updated.
				if n.SyncPeer.GetPeer() != m.Peer && n.isNewPeerHasMaxScore(t.Peer) {
					n.NetworkInfoCh <- ChangeSyncPeer{Peer: t.Peer}
				}
			} else {
				// Node better continue synchronization with one node, switching to new node happens only if the larger
				// group of nodes with the highest score appears.
				//TODO: implement
			}

		case *peer.InternalErr:
			n.peers.Disconnect(m.Peer)
			if n.peers.ConnectedCount() < n.minPeerMining {
				n.NetworkInfoCh <- StopMining{}
			}
			if n.SyncPeer.GetPeer() == m.Peer {
				n.NetworkInfoCh <- StopSync{}
			}
		default:
			zap.S().Warnf("Unknown peer info message '%T'", m)
		}
	}
}

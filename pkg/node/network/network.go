package network

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const defaultChannelSize = 100

type InfoMessage any

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

func (s *SyncPeer) Clear() {
	s.m.Lock()
	defer s.m.Unlock()
	s.peer = nil
}

type Network struct {
	infoCh        <-chan peer.InfoMessage
	networkInfoCh chan<- InfoMessage
	syncPeer      *SyncPeer
	logger        *slog.Logger

	peers         peers.PeerManager
	storage       state.State
	tm            types.Time
	minPeerMining int
	obsolescence  time.Duration
}

func NewNetwork(
	services services.Services,
	p peer.Parent,
	obsolescence time.Duration,
	l *slog.Logger,
) (Network, <-chan InfoMessage) {
	nch := make(chan InfoMessage, defaultChannelSize)
	return Network{
		infoCh:        p.InfoCh,
		networkInfoCh: nch,
		syncPeer:      new(SyncPeer),
		logger:        l,
		peers:         services.Peers,
		storage:       services.State,
		tm:            services.Time,
		minPeerMining: services.MinPeersMining,
		obsolescence:  obsolescence,
	}, nch
}

func (n *Network) SyncPeer() *SyncPeer {
	return n.syncPeer
}

func (n *Network) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.logger.Info("Network terminated")
			return
		case m, ok := <-n.infoCh:
			if !ok {
				n.logger.Warn("Incoming message channel was closed by producer")
				return
			}
			switch t := m.Value.(type) {
			case *peer.Connected:
				n.handleConnected(t)
			case *peer.InternalErr:
				n.handleInternalErr(m)
			default:
				n.logger.Warn("Unknown peer info message", logging.Type(m))
			}
		}
	}
}

func (n *Network) handleConnected(msg *peer.Connected) {
	err := n.peers.NewConnection(msg.Peer)
	if err != nil {
		n.logger.Debug("Failed to establish connection with peer", slog.Any("direction", msg.Peer.Direction()),
			slog.Any("peer", msg.Peer.ID()), logging.Error(err))
		return
	}
	n.logger.Debug("Established connection with peer", "direction", msg.Peer.Direction(),
		"peer", msg.Peer.ID(), "count", n.peers.ConnectedCount())
	if n.peers.ConnectedCount() == n.minPeerMining { // TODO: Consider producing duplicate events here
		n.networkInfoCh <- StartMining{}
	}
	sendScore(msg.Peer, n.storage)

	//TODO: Do we need to check it here after async operation of sending score to the peer. Possibly we don't
	// know peer's score yet, because we haven't received it yet.
	n.switchToNewPeerIfRequired()
}

func (n *Network) handleInternalErr(msg peer.InfoMessage) {
	n.peers.Disconnect(msg.Peer)
	n.logger.Debug("Disconnected from peer", "direction", msg.Peer.Direction(), "peer", msg.Peer.ID(),
		"count", n.peers.ConnectedCount())
	if n.peers.ConnectedCount() < n.minPeerMining {
		// TODO: Consider handling of duplicate events in consumer
		n.networkInfoCh <- StopMining{}
	}
	if msg.Peer.Equal(n.syncPeer.GetPeer()) {
		n.networkInfoCh <- StopSync{}
	}
}

func (n *Network) isTimeToSwitchPeerWithMaxScore() bool {
	now := n.tm.Now()
	obsolescenceTime := now.Add(-n.obsolescence)
	lastBlock := n.storage.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	return !obsolescenceTime.After(lastBlockTime)
}

func (n *Network) switchToNewPeerIfRequired() {
	if n.isTimeToSwitchPeerWithMaxScore() {
		// Node is getting close to the top of the blockchain, it's time to switch on a node with the highest
		// score every time it updated.
		if np, ok := n.peers.CheckPeerWithMaxScore(n.syncPeer.GetPeer()); ok {
			n.networkInfoCh <- ChangeSyncPeer{Peer: np}
		}
	} else {
		// Node better continue synchronization with one node, switching to new node happens only if the larger
		// group of nodes with the highest score appears.
		if np, ok := n.peers.CheckPeerInLargestScoreGroup(n.syncPeer.GetPeer()); ok {
			n.networkInfoCh <- ChangeSyncPeer{Peer: np}
		}
	}
}

func sendScore(p peer.Peer, storage state.State) {
	curScore, err := storage.CurrentScore()
	if err != nil {
		slog.Error("Failed to send current score to peer", slog.String("address", p.RemoteAddr().String()),
			logging.Error(err))
		return
	}

	bts := curScore.Bytes()
	p.SendMessage(&proto.ScoreMessage{Score: bts})
}

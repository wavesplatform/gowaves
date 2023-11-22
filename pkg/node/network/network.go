package network

import (
	"context"
	"math/big"
	"net"
	"reflect"
	"slices"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	ps "github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

const (
	networkChannelsDefaultSize       = 100
	askPeersInterval                 = 5 * time.Minute
	spawnOutgoingConnectionsInterval = time.Minute
)

// Network represent service.
type Network struct {
	sm *stateless.StateMachine

	ctx  context.Context
	wait func() error

	peersCh         <-chan peer.Notification
	networkCh       <-chan peer.ProtoMessage
	commandsCh      <-chan Command
	notificationsCh chan<- Notification

	scheme          proto.Scheme
	quorumThreshold int
	bindAddr        proto.TCPAddr
	declAddr        proto.TCPAddr
	syncPeer        peer.Peer
	leaderMode      bool

	peers peers.PeerManager
	st    state.State

	metricGetPeersMessage prometheus.Counter
	metricPeersMessage    prometheus.Counter
}

func NewNetwork(
	peersCh <-chan peer.Notification,
	networkCh <-chan peer.ProtoMessage,
	peers peers.PeerManager,
	st state.State,
	scheme proto.Scheme,
	quorumThreshold int,
	bindAddr, declAddr proto.TCPAddr,
) (*Network, <-chan Notification) {
	nch := make(chan Notification, networkChannelsDefaultSize)
	n := &Network{
		sm:              stateless.NewStateMachine(stageDisconnected),
		peersCh:         peersCh,
		networkCh:       networkCh,
		notificationsCh: nch,
		peers:           peers,
		st:              st,
		scheme:          scheme,
		quorumThreshold: quorumThreshold,
		bindAddr:        bindAddr,
		declAddr:        declAddr,
	}

	n.registerMetrics()

	n.sm.SetTriggerParameters(eventPeerConnected, reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventPeerDisconnected, reflect.TypeOf((*peer.Peer)(nil)).Elem(),
		reflect.TypeOf((*error)(nil)).Elem())
	n.sm.SetTriggerParameters(eventScore, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf((*proto.Score)(nil)))
	n.sm.SetTriggerParameters(eventGetPeers, reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventPeers, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf([]proto.PeerInfo{}))
	n.sm.SetTriggerParameters(eventBlacklistPeer, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(""))
	n.sm.SetTriggerParameters(eventSuspendPeer, reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(""))
	n.sm.SetTriggerParameters(eventBroadcastTransaction, reflect.TypeOf((*proto.Transaction)(nil)).Elem(),
		reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventBroadcastMicroBlockInv, reflect.TypeOf((*proto.MicroBlockInv)(nil)),
		reflect.TypeOf((*peer.Peer)(nil)).Elem())
	n.sm.SetTriggerParameters(eventBroadcastBlock, reflect.TypeOf((*proto.Block)(nil)),
		reflect.TypeOf((*peer.Peer)(nil)).Elem())

	n.configureDisconnectedState()
	n.configureGroupState()
	n.configureLeaderState()
	n.configureHaltState()

	return n, nch
}

func (n *Network) registerMetrics() {
	n.metricGetPeersMessage = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "messages",
			Name:      "get_peers_total",
			Help:      "Counter of GetPeers message.",
		},
	)
	n.metricPeersMessage = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "messages",
			Name:      "peers_total",
			Help:      "Counter of Peers message.",
		},
	)
	prometheus.MustRegister(n.metricPeersMessage)
	prometheus.MustRegister(n.metricGetPeersMessage)
}

func (n *Network) configureDisconnectedState() {
	n.sm.Configure(stageDisconnected).
		OnEntry(n.onEnterDisconnected).
		InternalTransition(eventScore, n.onScore).
		InternalTransition(eventGetPeers, n.onGetPeers).
		InternalTransition(eventPeers, n.onPeers).
		InternalTransition(eventAskPeers, n.onAskPeers).
		InternalTransition(eventScoreUpdated, n.onScoreUpdated).
		InternalTransition(eventPeerConnected, n.onPeerConnected).
		InternalTransition(eventPeerDisconnected, n.onPeerDisconnected).
		InternalTransition(eventFollowGroup, n.onFollowGroup).
		InternalTransition(eventFollowLeader, n.onFollowLeader).
		Ignore(eventBlacklistPeer).                     // Impossible in Disconnected state.
		Ignore(eventSuspendPeer).                       // Impossible in Disconnected state.
		Ignore(eventBroadcastTransaction).              // Impossible in Disconnected state.
		Ignore(eventQuorumChanged, n.quorumNotReached). // No transition if quorum is not reached.
		Permit(eventQuorumChanged, stageLeader, n.quorumReached, n.followLeader).
		Permit(eventQuorumChanged, stageGroup, n.quorumReached, n.followGroup).
		InternalTransition(eventCheckQuorum, n.updateQuorum, n.quorumReached).
		Ignore(eventCheckQuorum, n.quorumNotReached).
		Ignore(eventFollowingModeChanged). // Do nothing.
		Ignore(eventAnnounceScore).
		Ignore(eventBroadcastMicroBlockInv).
		Ignore(eventBroadcastBlock).
		Permit(eventHalt, stageHalt)
}

func (n *Network) configureGroupState() {
	n.sm.Configure(stageGroup).
		// Entry from Disconnected state, emits eventFollowingModeChanged.
		OnEntryFrom(eventQuorumChanged, n.onEnterGroupFromDisconnected).
		OnEntryFrom(eventFollowingModeChanged, n.selectGroup).
		InternalTransition(eventScore, n.onScore). // Emits eventScoreUpdated.
		InternalTransition(eventGetPeers, n.onGetPeers).
		InternalTransition(eventPeers, n.onPeers).
		InternalTransition(eventAskPeers, n.onAskPeers).
		InternalTransition(eventScoreUpdated, n.selectGroup).            // On re-enter from this state.
		InternalTransition(eventPeerConnected, n.onPeerConnected).       // Emits eventQuorumChanged.
		InternalTransition(eventPeerDisconnected, n.onPeerDisconnected). // Emits eventQuorumChanged.
		InternalTransition(eventFollowGroup, n.onFollowGroup).           // Emits eventFollowingModeChanged.
		InternalTransition(eventFollowLeader, n.onFollowLeader).         // Emits eventFollowingModeChanged.
		InternalTransition(eventBlacklistPeer, n.onBlacklist).
		InternalTransition(eventSuspendPeer, n.onSuspend).
		InternalTransition(eventBroadcastTransaction, n.onBroadcast).
		Ignore(eventQuorumChanged, n.quorumReached).
		Permit(eventQuorumChanged, stageDisconnected, n.quorumNotReached).
		InternalTransition(eventCheckQuorum, n.updateQuorum, n.quorumReached).
		Ignore(eventCheckQuorum, n.quorumNotReached).
		Ignore(eventFollowingModeChanged, n.followGroup).
		Permit(eventFollowingModeChanged, stageLeader, n.followLeader).
		InternalTransition(eventAnnounceScore, n.onAnnounceScore).
		InternalTransition(eventBroadcastMicroBlockInv, n.onBroadcastMicroBlockInv).
		InternalTransition(eventBroadcastBlock, n.onBroadcastBlock).
		Permit(eventHalt, stageHalt)
}

func (n *Network) configureLeaderState() {
	n.sm.Configure(stageLeader).
		OnEntryFrom(eventQuorumChanged, n.onEnterLeaderFromDisconnected).
		OnEntryFrom(eventFollowingModeChanged, n.selectLeader).
		InternalTransition(eventScore, n.onScore).
		InternalTransition(eventGetPeers, n.onGetPeers).
		InternalTransition(eventPeers, n.onPeers).
		InternalTransition(eventAskPeers, n.onAskPeers).
		InternalTransition(eventScoreUpdated, n.selectLeader).
		InternalTransition(eventPeerConnected, n.onPeerConnected).
		InternalTransition(eventPeerDisconnected, n.onPeerDisconnected).
		InternalTransition(eventFollowGroup, n.onFollowGroup).
		InternalTransition(eventFollowLeader, n.onFollowLeader).
		InternalTransition(eventBlacklistPeer, n.onBlacklist).
		InternalTransition(eventSuspendPeer, n.onSuspend).
		InternalTransition(eventBroadcastTransaction, n.onBroadcast).
		Ignore(eventQuorumChanged, n.quorumReached).
		Permit(eventQuorumChanged, stageDisconnected, n.quorumNotReached).
		InternalTransition(eventCheckQuorum, n.updateQuorum, n.quorumReached).
		Ignore(eventCheckQuorum, n.quorumNotReached).
		Permit(eventFollowingModeChanged, stageGroup, n.followGroup).
		Ignore(eventFollowingModeChanged, n.followLeader).
		InternalTransition(eventAnnounceScore, n.onAnnounceScore).
		InternalTransition(eventBroadcastMicroBlockInv, n.onBroadcastMicroBlockInv).
		InternalTransition(eventBroadcastBlock, n.onBroadcastBlock).
		Permit(eventHalt, stageHalt)
}

func (n *Network) configureHaltState() {
	n.sm.Configure(stageHalt).
		OnEntry(n.onEnterHalt).
		Ignore(eventScore).
		Ignore(eventGetPeers).
		Ignore(eventPeers).
		Ignore(eventAskPeers).
		Ignore(eventScoreUpdated).
		Ignore(eventPeerConnected).
		Ignore(eventPeerDisconnected).
		Ignore(eventFollowGroup).
		Ignore(eventFollowLeader).
		Ignore(eventBlacklistPeer).
		Ignore(eventSuspendPeer).
		Ignore(eventBroadcastTransaction).
		Ignore(eventQuorumChanged).
		Ignore(eventCheckQuorum).
		Ignore(eventFollowingModeChanged).
		Ignore(eventAnnounceScore).
		Ignore(eventBroadcastMicroBlockInv).
		Ignore(eventBroadcastBlock).
		Ignore(eventHalt)
}

func (n *Network) Run(ctx context.Context) {
	g, gc := errgroup.WithContext(ctx)
	n.ctx = gc
	n.wait = g.Wait

	g.Go(n.runPeersExchange)
	g.Go(n.runOutgoingConnections)
	g.Go(n.runIncomingConnections)
	g.Go(n.handleEvents)
}

func (n *Network) Shutdown() {
	if err := n.wait(); err != nil {
		zap.S().Named(logging.NetworkNamespace).
			Warnf("[%s] Failed to shutdown network service: %v", n.sm.MustState(), err)
	}
	zap.S().Named(logging.NetworkNamespace).Infof("[%s] Network shutdown successfully", n.sm.MustState())
}

func (n *Network) SetCommandChannel(commandCh <-chan Command) {
	if commandCh == nil {
		panic("commandCh must not be nil")
	}
	n.commandsCh = commandCh
}

func (n *Network) runPeersExchange() error {
	ticker := time.NewTicker(askPeersInterval)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Peers exchange stopped", n.sm.MustState())
			return nil
		case <-ticker.C:
			if err := n.sm.Fire(eventAskPeers); err != nil {
				zap.S().Named(logging.NetworkNamespace).
					Warnf("[%s] Failed to ask for peers: %v", n.sm.MustState(), err)
			}
		}
	}
}

func (n *Network) runOutgoingConnections() error {
	ticker := time.NewTicker(spawnOutgoingConnectionsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-n.ctx.Done():
			zap.S().Named(logging.NetworkNamespace).
				Debugf("[%s] Outgoing connections creation stopped", n.sm.MustState())
			return nil
		case <-ticker.C:
			n.peers.SpawnOutgoingConnections(n.ctx)
		}
	}
}

func (n *Network) runIncomingConnections() error {
	// if empty declared address, listen on port doesn't make sense
	if n.declAddr.Empty() {
		zap.S().Named(logging.NetworkNamespace).Warn("Declared address is empty")
		return nil
	}

	if n.bindAddr.Empty() {
		zap.S().Named(logging.NetworkNamespace).Warn("Bind address is empty")
		return nil
	}

	zap.S().Named(logging.NetworkNamespace).Infof("Start listening on %s", n.bindAddr.String())
	var lc net.ListenConfig
	l, err := lc.Listen(n.ctx, "tcp", n.bindAddr.String())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	defer func() {
		if clErr := l.Close(); clErr != nil {
			zap.S().Named(logging.NetworkNamespace).
				Errorf("Failed to close %T on addr %q: %v", l, l.Addr().String(), clErr)
		}
	}()

	// TODO: implement good graceful shutdown
	for {
		conn, acErr := l.Accept()
		if acErr != nil {
			if errors.Is(acErr, context.Canceled) {
				return nil
			}
			zap.S().Named(logging.NetworkNamespace).Errorf("Failed to accept new peer: %v", err)
			continue
		}

		go func() {
			if spErr := n.peers.SpawnIncomingConnection(n.ctx, conn); spErr != nil {
				zap.S().Named(logging.NetworkNamespace).Debugf("Incoming connection failure from '%s': %v",
					conn.RemoteAddr().String(), err)
				return
			}
		}()
	}
}

func (n *Network) handleEvents() error {
	for {
		select {
		case <-n.ctx.Done():
			if err := n.sm.Fire(eventHalt); err != nil {
				zap.S().Named(logging.NetworkNamespace).
					Warnf("[%s] Failed to handle halt event: %v", n.sm.MustState(), err)
			}
			zap.S().Named(logging.NetworkNamespace).
				Debugf("[%s] Network termination started", n.sm.MustState())
			return nil
		case m, ok := <-n.peersCh:
			if err := n.handlePeerNotifications(m, ok); err != nil {
				return err
			}
		case m, ok := <-n.networkCh:
			if err := n.handleNetworkMessages(m, ok); err != nil {
				return err
			}
		case c, ok := <-n.commandsCh:
			if err := n.handleCommands(c, ok); err != nil {
				return err
			}
		}
	}
}

func (n *Network) handlePeerNotifications(m peer.Notification, ok bool) error {
	if !ok {
		zap.S().Named(logging.NetworkNamespace).
			Warnf("[%s] Peers notifications channel was closed by producer", n.sm.MustState())
		return errors.New("peers notifications channel was closed")
	}
	switch v := m.(type) {
	case peer.ConnectedNotification:
		if v.Peer == nil {
			zap.S().Named(logging.NetworkNamespace).
				Debugf("[%s] Connected notification with empty peer", n.sm.MustState())
			return nil
		}
		zap.S().Named(logging.NetworkNamespace).
			Debugf("[%s] Notification about connection with peer %s (%s)",
				n.sm.MustState(), v.Peer.RemoteAddr(), v.Peer.ID())
		if err := n.sm.Fire(eventPeerConnected, v.Peer); err != nil {
			zap.S().Named(logging.NetworkNamespace).Warnf("[%s] Failed to handle new peer: %v",
				n.sm.MustState(), err)
		}
	case peer.DisconnectedNotification:
		if err := n.sm.Fire(eventPeerDisconnected, v.Peer, v.Err); err != nil {
			zap.S().Named(logging.NetworkNamespace).Warnf("[%s] Failed to handle peer error: %v",
				n.sm.MustState(), err)
		}
	default:
		zap.S().Named(logging.NetworkNamespace).Errorf("[%s] Unknown peer info message '%T'",
			n.sm.MustState(), m)
		return errors.Errorf("unexpected peers info message '%T'", m)
	}
	return nil
}

func (n *Network) handleNetworkMessages(m peer.ProtoMessage, ok bool) error {
	if !ok {
		zap.S().Named(logging.NetworkNamespace).
			Warnf("[%s] Network channel was closed by producer", n.sm.MustState())
		return errors.New("network channel was closed")
	}
	switch msg := m.Message.(type) {
	case *proto.ScoreMessage:
		if err := n.sm.Fire(eventScore, m.ID, new(big.Int).SetBytes(msg.Score)); err != nil {
			zap.S().Named(logging.NetworkNamespace).Warnf("[%s] Failed to handle Score message: %v",
				n.sm.MustState(), err)
		}
	case *proto.GetPeersMessage:
		if err := n.sm.Fire(eventGetPeers, m.ID); err != nil {
			zap.S().Named(logging.NetworkDataNamespace).
				Warnf("[%s] Failed to handle GetPeers message: %v", n.sm.MustState(), err)
		}

	case *proto.PeersMessage:
		if err := n.sm.Fire(eventPeers, m.ID, msg.Peers); err != nil {
			zap.S().Named(logging.NetworkDataNamespace).
				Warnf("[%s] Failed to handle Peers message: %v", n.sm.MustState(), err)
		}
	default:
		zap.S().Named(logging.NetworkNamespace).
			Errorf("[%s] Unexpected network message '%T' from %s peer '%s'",
				n.sm.MustState(), msg, m.ID.Direction(), m.ID.ID().String())
		return errors.Errorf("unexpected network message '%T'", m)
	}
	return nil
}

func (n *Network) handleCommands(c Command, ok bool) error {
	if !ok {
		zap.S().Named(logging.NetworkNamespace).
			Warnf("[%s] Network service command channel was closed", n.sm.MustState())
		return errors.New("network commands channel was closed")
	}
	switch cmd := c.(type) {
	case FollowGroupCommand:
		if err := n.sm.Fire(eventFollowGroup); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle FollowGroup command: %v", n.sm.MustState(), err)
		}
	case FollowLeaderCommand:
		if err := n.sm.Fire(eventFollowLeader); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle FollowLeader command: %v", n.sm.MustState(), err)
		}
	case BlacklistPeerCommand:
		if err := n.sm.Fire(eventBlacklistPeer, cmd.Peer, cmd.Message); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle BlacklistPeer command: %v", n.sm.MustState(), err)
		}
	case SuspendPeerCommand:
		if err := n.sm.Fire(eventSuspendPeer, cmd.Peer, cmd.Message); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle SuspendPeer command: %v", n.sm.MustState(), err)
		}
	case BroadcastTransactionCommand:
		if err := n.sm.Fire(eventBroadcastTransaction, cmd.Transaction, cmd.Origin); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle BroadcastTransaction command: %v", n.sm.MustState(), err)
		}
	case AnnounceScoreCommand:
		if err := n.sm.Fire(eventAnnounceScore); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle AnnounceScore command: %v", n.sm.MustState(), err)
		}
	case BroadcastMicroBlockInvCommand:
		if err := n.sm.Fire(eventBroadcastMicroBlockInv, cmd.MicroBlockInv, cmd.Origin); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle BroadcastMicroBlockInv command: %v", n.sm.MustState(), err)
		}
	case BroadcastBlockCommand:
		if err := n.sm.Fire(eventBroadcastBlock, cmd.Block, cmd.Origin); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle BroadcastBlock command: %v", n.sm.MustState(), err)
		}
	case RequestQuorumUpdate:
		if err := n.sm.Fire(eventCheckQuorum); err != nil {
			zap.S().Named(logging.NetworkNamespace).
				Warnf("[%s] Failed to handle RequestQuorumUpdate command: %v", n.sm.MustState(), err)
		}
	default:
		zap.S().Named(logging.NetworkNamespace).Errorf("[%s] Unexpected network command type %T",
			n.sm.MustState(), c)
		return errors.Errorf("unexpected network command '%T'", c)
	}
	return nil
}

// onEnterDisconnected notifies about the loss of quorum. This function is not called on creation of the state.
func (n *Network) onEnterDisconnected(_ context.Context, _ ...any) error {
	n.notificationsCh <- QuorumLostNotification{}
	return nil
}

func (n *Network) sendScore(p peer.Peer) {
	s, err := n.st.CurrentScore()
	if err != nil {
		zap.S().Errorf("[%s] Failed to send local score to peer %q: %v",
			n.sm.MustState(), p.RemoteAddr().String(), err)
		return
	}
	p.SendMessage(&proto.ScoreMessage{Score: s.Bytes()})
}

// onScore updates peer's score in the score manager and fires `eventScoreUpdated`.
func (n *Network) onScore(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first, expected 'peer.Peer'", args[0])
	}
	s, ok := args[1].(*proto.Score)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '*proto.Score'", args[1])
	}
	if err := n.peers.UpdateScore(p, s); err != nil {
		return err
	}
	return n.sm.Fire(eventScoreUpdated)
}

// onScoreUpdated fires eventQuorumChanged to initiate quorum check after receiving score from recently connected peer.
func (n *Network) onScoreUpdated(_ context.Context, _ ...any) error {
	return n.sm.Fire(eventQuorumChanged)
}

// onGetPeers replies to the requester peer with all known peers.
func (n *Network) onGetPeers(_ context.Context, args ...any) error {
	n.metricGetPeersMessage.Inc()
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	rs := n.peers.AllKnownPeers()
	out := make([]proto.PeerInfo, 0, len(rs))
	for _, r := range rs {
		ipPort := proto.IpPort(r)
		out = append(out, proto.PeerInfo{
			Addr: ipPort.Addr(),
			Port: uint16(ipPort.Port()),
		})
	}
	p.SendMessage(&proto.PeersMessage{Peers: out})
	return nil
}

// onPeers updates known peers in the peer manager by adding an unknown peers from received message.
func (n *Network) onPeers(_ context.Context, args ...any) error {
	n.metricPeersMessage.Inc()
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	msg, ok := args[1].([]proto.PeerInfo)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected '[]proto.PeerInfo'", args[1])
	}
	if len(msg) == 0 {
		return nil
	}
	alreadyKnown := n.peers.AllKnownPeers()
	r := make([]ps.KnownPeer, 0, len(msg))
	for _, mp := range msg {
		kp := ps.KnownPeer(proto.NewTCPAddr(mp.Addr, int(mp.Port)).ToIpPort())
		if slices.Contains(alreadyKnown, kp) {
			continue
		}
		r = append(r, kp)
	}
	if len(r) > 0 {
		zap.S().Named(logging.NetworkNamespace).
			Debugf("[%s] %d unknown peers received from '%s'", n.sm.MustState(), len(r), p.ID().String())
	}
	return n.peers.UpdateKnownPeers(r)
}

// onPeerConnected registers new connection in the peer manager, sends current score to the newly connected peer.
func (n *Network) onPeerConnected(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	if err := n.peers.NewConnection(p); err != nil {
		zap.S().Named(logging.NetworkNamespace).Warnf("[%s] Failed to register new %s peer '%s': %v",
			n.sm.MustState(), p.Direction(), p.ID(), err)
		return nil // Do not interrupt state machine execution with an error.
	}
	n.sendScore(p) // Always send our score to newly connected peer.
	return nil
}

// onPeerDisconnected registers the peer as disconnected in the peer manager and fires `eventQuorumChanged`
// event to initiate quorum check.
func (n *Network) onPeerDisconnected(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	e, ok := args[1].(error)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'error'", args[1])
	}

	n.peers.Disconnect(p)
	zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Lost connection with %s peer '%s': %v",
		n.sm.MustState(), p.Direction(), p.ID(), e)

	return n.sm.Fire(eventQuorumChanged)
}

// onFollowGroup updates following mode flag in the Network's internal state and fires `eventFollowingModeChanged`
// event to initiate transition if required.
func (n *Network) onFollowGroup(_ context.Context, _ ...any) error {
	n.leaderMode = false
	return n.sm.Fire(eventFollowingModeChanged)
}

// onFollowLeader updates following mode flag in the Network's internal state and fires `eventFollowingModeChanged`
// event to initiate transition if required.
func (n *Network) onFollowLeader(_ context.Context, _ ...any) error {
	n.leaderMode = true
	return n.sm.Fire(eventFollowingModeChanged)
}

// onEnterGroupFromDisconnected selects the peer from the largest group and sends
// the QuorumMetNotification with this peer.
func (n *Network) onEnterGroupFromDisconnected(_ context.Context, _ ...any) error {
	if np, _, ok := n.peers.CheckPeerInLargestScoreGroup(n.syncPeer); ok {
		pid := "n/a"
		if n.syncPeer != nil {
			pid = n.syncPeer.ID().String()
		}
		n.syncPeer = np
		zap.S().Named(logging.NetworkNamespace).Debugf("Changing best peer from '%s' to '%s'",
			pid, np.ID().String())
		n.notificationsCh <- QuorumMetNotification{Peer: np}
	}
	return nil
}

// onEnterLeaderFromDisconnected selects the peer with the largest score and sends
// the QuorumMetNotification with this peer.
func (n *Network) onEnterLeaderFromDisconnected(_ context.Context, _ ...any) error {
	if np, _, ok := n.peers.CheckPeerWithMaxScore(n.syncPeer); ok {
		pid := "n/a"
		if n.syncPeer != nil {
			pid = n.syncPeer.ID().String()
		}
		n.syncPeer = np
		zap.S().Named(logging.NetworkNamespace).Debugf("Changing peer with max score from '%s' to '%s'",
			pid, np.ID().String())
		n.notificationsCh <- QuorumMetNotification{Peer: np}
	}
	return nil
}

// onAskPeers fans out a request for known peers to all connected nodes.
// Initiated by the Network itself every `askPeersInterval` (5 minutes by default).
func (n *Network) onAskPeers(_ context.Context, _ ...any) error {
	zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Requesting peers", n.sm.MustState())
	n.peers.AskPeers()
	return nil
}

// onBlacklist adds peer to the blacklist of the peer manager.
func (n *Network) onBlacklist(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	m, ok := args[1].(string)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'string'", args[1])
	}
	n.peers.AddToBlackList(p, time.Now(), m)
	return nil
}

// onSuspend suspends peer in the peer manager.
func (n *Network) onSuspend(_ context.Context, args ...any) error {
	p, ok := args[0].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'peer.Peer'", args[0])
	}
	m, ok := args[1].(string)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'string'", args[1])
	}
	n.peers.Suspend(p, time.Now(), m)
	return nil
}

func (n *Network) onBroadcast(_ context.Context, args ...any) error {
	tx, ok := args[0].(proto.Transaction)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected 'proto.Transaction'", args[0])
	}
	id, err := tx.GetID(n.scheme)
	if err != nil {
		return errors.Wrapf(err, "failed to get transaction ID in state '%s'", n.sm.MustState())
	}
	txID := base58.Encode(id)
	op, ok := args[1].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'peer.Peer'", args[1])
	}
	bts, err := tx.MarshalToProtobuf(n.scheme)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal transactoin '%s' in state '%s'", n.sm.MustState(), txID)
	}
	msg := &proto.PBTransactionMessage{Transaction: bts}
	cnt := 0
	n.peers.EachConnected(func(p peer.Peer, _ *proto.Score) {
		if p != op {
			p.SendMessage(msg)
			cnt++
		}
	})
	zap.S().Named(logging.NetworkNamespace).
		Debugf("[%s] Transaction message '%s'' sent to %d peers", n.sm.MustState(), txID, cnt)
	return nil
}

func (n *Network) selectGroup(_ context.Context, _ ...any) error {
	if np, s, ok := n.peers.CheckPeerInLargestScoreGroup(n.syncPeer); ok {
		n.syncPeer = np
		n.notificationsCh <- SyncPeerChangedNotification{Peer: np, Score: s}
	}
	return nil
}

func (n *Network) selectLeader(_ context.Context, _ ...any) error {
	if np, s, ok := n.peers.CheckPeerWithMaxScore(n.syncPeer); ok {
		n.syncPeer = np
		n.notificationsCh <- SyncPeerChangedNotification{Peer: np, Score: s}
	}
	return nil
}

func (n *Network) onAnnounceScore(_ context.Context, _ ...any) error {
	score, err := n.st.CurrentScore()
	if err != nil {
		zap.S().Named(logging.NetworkNamespace).
			Errorf("[%s] Failed to get current score: %v", n.sm.MustState(), err)
		return nil
	}
	var (
		msg = &proto.ScoreMessage{Score: score.Bytes()}
		cnt int
	)
	n.peers.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(msg)
		cnt++
	})
	zap.S().Named(logging.NetworkNamespace).Debugf("[%s] Score '%s' announced to %d peers",
		n.sm.MustState(), score.String(), cnt)
	return nil
}

func (n *Network) onBroadcastMicroBlockInv(_ context.Context, args ...any) error {
	inv, ok := args[0].(*proto.MicroBlockInv)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected '*proto.MicroBlockInv'", args[0])
	}
	op, ok := args[1].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'peer.Peer'", args[1])
	}

	bts, err := inv.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal binary MicroBlockInv message")
	}
	msg := &proto.MicroBlockInvMessage{Body: bts}
	cnt := 0
	n.peers.EachConnected(func(p peer.Peer, _ *proto.Score) {
		if p != op {
			p.SendMessage(msg)
			cnt++
		}
	})
	zap.S().Named(logging.NetworkNamespace).
		Debugf("[%s] MicroBlockInv message (%s <- %s) sent to %d peers",
			n.sm.MustState(), inv.Reference.String(), inv.TotalBlockID.String(), cnt)
	return nil
}

func (n *Network) onBroadcastBlock(_ context.Context, args ...any) error {
	b, ok := args[0].(*proto.Block)
	if !ok {
		return errors.Errorf("invalid type '%T' of first argument, expected '*proto.Block'", args[0])
	}
	op, ok := args[1].(peer.Peer)
	if !ok {
		return errors.Errorf("invalid type '%T' of second argument, expected 'peer.Peer'", args[1])
	}

	bts, err := b.Marshal(n.scheme)
	if err != nil {
		return errors.Wrap(err, "failed to marshal Block message")
	}
	msg := &proto.BlockMessage{BlockBytes: bts}
	var (
		cnt int
	)
	n.peers.EachConnected(func(p peer.Peer, _ *proto.Score) {
		if p != op {
			p.SendMessage(msg)
			cnt++
		}
	})
	zap.S().Named(logging.NetworkNamespace).
		Debugf("[%s] Block message '%s' sent to %d peers", n.sm.MustState(), b.BlockID().String(), cnt)
	return nil
}

func (n *Network) onEnterHalt(_ context.Context, _ ...any) error {
	n.peers.Close()
	close(n.notificationsCh)
	return nil
}

func (n *Network) quorumReached(_ context.Context, _ ...any) bool {
	return n.peers.ConnectedCount() >= n.quorumThreshold
}

func (n *Network) quorumNotReached(ctx context.Context, args ...any) bool {
	return !n.quorumReached(ctx, args...)
}

func (n *Network) followLeader(_ context.Context, _ ...any) bool {
	return n.leaderMode
}

func (n *Network) followGroup(_ context.Context, _ ...any) bool {
	return !n.leaderMode
}

func (n *Network) updateQuorum(_ context.Context, _ ...any) error {
	if n.syncPeer != nil {
		n.notificationsCh <- QuorumMetNotification{Peer: n.syncPeer}
	}
	return nil
}

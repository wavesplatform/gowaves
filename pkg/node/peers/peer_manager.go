package peers

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	suspendDuration              = 5 * time.Minute
	clearRestrictedPeersInterval = 1 * time.Minute
)

type peerInfo struct {
	score *big.Int
	peer  peer.Peer
}

func newPeerInfo(peer peer.Peer) peerInfo {
	return peerInfo{
		score: big.NewInt(0),
		peer:  peer,
	}
}

type PeerManager interface {
	NewConnection(peer.Peer) error
	ConnectedCount() int
	EachConnected(func(peer.Peer, *proto.Score))
	Suspend(peer peer.Peer, suspendTime time.Time, reason string)
	Suspended() []storage.SuspendedPeer
	AddToBlackList(peer peer.Peer, blockTime time.Time, reason string)
	BlackList() []storage.BlackListedPeer
	ClearBlackList() error
	UpdateScore(p peer.Peer, score *proto.Score) error
	KnownPeers() []storage.KnownPeer
	UpdateKnownPeers([]storage.KnownPeer) error
	Close() error
	SpawnOutgoingConnections(context.Context)
	SpawnIncomingConnection(ctx context.Context, conn net.Conn) error
	Spawned() []proto.IpPort
	Connect(context.Context, proto.TCPAddr) error
	Score(p peer.Peer) (*proto.Score, error)

	// AskPeers sends GetPeersMessage message to all connected nodes.
	AskPeers()

	CheckPeerWithMaxScore(p peer.Peer) (peer.Peer, bool)
	CheckPeerInLargestScoreGroup(p peer.Peer) (peer.Peer, bool)

	Disconnect(peer.Peer)
}

type PeerManagerImpl struct {
	spawner                   PeerSpawner
	active                    activePeers
	mu                        sync.RWMutex
	peerStorage               PeerStorage
	spawned                   map[proto.IpPort]struct{}
	enableOutboundConnections bool
	blackListDuration         time.Duration
	limitConnections          int
	newConnectionsLimit       int
	version                   proto.Version
	networkName               string
	logger                    *slog.Logger
}

func NewPeerManager(spawner PeerSpawner, storage PeerStorage, limitConnections int, version proto.Version,
	networkName string, enableOutboundConnections bool, newConnectionsLimit int,
	blackListDuration time.Duration, logger *slog.Logger) *PeerManagerImpl {
	return &PeerManagerImpl{
		spawner:                   spawner,
		active:                    newActivePeers(),
		peerStorage:               storage,
		spawned:                   make(map[proto.IpPort]struct{}),
		enableOutboundConnections: enableOutboundConnections,
		blackListDuration:         blackListDuration,
		limitConnections:          limitConnections,
		newConnectionsLimit:       newConnectionsLimit,
		version:                   version,
		networkName:               networkName,
		logger:                    logger,
	}
}

func (a *PeerManagerImpl) NewConnection(p peer.Peer) (err error) {
	_, connected := a.connected(p)
	if connected {
		_ = p.Close()
		return errors.Errorf("already connected peer '%s'", p.ID())
	}

	now := time.Now()
	if p.Direction() == peer.Outgoing && a.suspended(p, now) {
		_ = p.Close()
		return errors.Errorf("peer '%s' is suspended", p.ID())
	}
	if p.Direction() == peer.Incoming && a.blackListed(p, now) {
		_ = p.Close()
		return errors.Errorf("peer '%s' is in black list", p.ID())
	}

	if p.Handshake().Version.CmpMinor(a.version) >= 2 {
		err := errors.Errorf(
			"versions are too different, current %s, connected %s (peer '%s')",
			a.version.String(),
			p.Handshake().Version.String(),
			p.ID(),
		)
		a.restrict(p, now, err.Error())
		_ = p.Close()
		return proto.NewInfoMsg(err)
	}
	if p.Handshake().AppName != a.networkName {
		err := errors.Errorf("peer '%s' has the invalid network name '%s', required '%s'",
			p.ID(), p.Handshake().AppName, a.networkName)
		a.restrict(p, now, err.Error())
		_ = p.Close()
		return proto.NewInfoMsg(err)
	}
	in, out := a.countDirections()
	switch p.Direction() {
	case peer.Incoming:
		if in >= a.limitConnections {
			_ = p.Close()
			return proto.NewInfoMsg(errors.Errorf("exceed incoming connections limit, incoming peer '%s'", p.ID()))
		}
	case peer.Outgoing:
		if !p.Handshake().DeclaredAddr.Empty() {
			known := storage.KnownPeer(proto.TCPAddr(p.Handshake().DeclaredAddr).ToIpPort())
			// TODO(nickeskov): maybe log error?
			_ = a.peerStorage.AddOrUpdateKnown([]storage.KnownPeer{known}, now)
		}
		if out >= a.limitConnections {
			_ = p.Close()
			return proto.NewInfoMsg(errors.Errorf("exceed outgoing connections limit, outgoing peer '%s'", p.ID()))
		}
	default:
		_ = p.Close()
		return errors.Errorf("unknown connection direction for peer '%s'", p.ID())
	}
	a.addConnected(p)
	return nil
}

func (a *PeerManagerImpl) ConnectedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.unsafeConnectedCount()
}

func (a *PeerManagerImpl) EachConnected(f func(peer peer.Peer, score *big.Int)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.active.forEach(
		func(_ peer.ID, info peerInfo) {
			f(info.peer, info.score)
		},
	)
}

func (a *PeerManagerImpl) Suspend(p peer.Peer, suspendTime time.Time, reason string) {
	a.Disconnect(p)
	a.mu.Lock()
	defer a.mu.Unlock()
	suspended := storage.NewSuspendedPeer(
		storage.IpFromIpPort(p.RemoteAddr().ToIpPort()),
		suspendTime.UnixMilli(),
		suspendDuration,
		reason,
	)
	if err := a.peerStorage.AddSuspended([]storage.SuspendedPeer{suspended}); err != nil {
		slog.Error("Failed to suspend peer", slog.Any("peer", p.ID()), slog.String("reason", reason),
			logging.Error(err))
	} else {
		a.logger.Debug("Suspending peer", "peer", p.ID(), "reason", reason)
	}
}

func (a *PeerManagerImpl) Suspended() []storage.SuspendedPeer {
	return a.peerStorage.Suspended(time.Now())
}

func (a *PeerManagerImpl) AddToBlackList(p peer.Peer, blockTime time.Time, reason string) {
	if a.blackListDuration <= 0 {
		return
	}

	a.Disconnect(p)
	a.mu.Lock()
	defer a.mu.Unlock()
	blackListed := storage.NewBlackListedPeer(
		storage.IpFromIpPort(p.RemoteAddr().ToIpPort()),
		blockTime.UnixMilli(),
		a.blackListDuration,
		reason,
	)
	if err := a.peerStorage.AddToBlackList([]storage.BlackListedPeer{blackListed}); err != nil {
		slog.Error("Failed to add peer to black list", slog.Any("peer", p.ID()),
			slog.String("reason", reason), logging.Error(err))
	} else {
		a.logger.Debug("Peer added to black list", "peer", p.ID(), "reason", reason)
	}
}

func (a *PeerManagerImpl) BlackList() []storage.BlackListedPeer {
	return a.peerStorage.BlackList(time.Now())
}

func (a *PeerManagerImpl) ClearBlackList() error {
	return a.peerStorage.DropBlackList()
}

func (a *PeerManagerImpl) UpdateScore(p peer.Peer, score *big.Int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.active.updateScore(p.ID(), score); err != nil {
		return errors.Wrap(err, "failed to update score")
	}
	return nil
}

func (a *PeerManagerImpl) KnownPeers() []storage.KnownPeer {
	return a.peerStorage.Known(a.newConnectionsLimit)
}

func (a *PeerManagerImpl) UpdateKnownPeers(known []storage.KnownPeer) error {
	if len(known) == 0 {
		return nil
	}
	if err := a.peerStorage.AddOrUpdateKnown(known, time.Now()); err != nil {
		return errors.Wrap(err, "failed to update known peers")
	}
	return nil
}

func (a *PeerManagerImpl) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.active.forEach(
		func(_ peer.ID, info peerInfo) {
			_ = info.peer.Close()
		},
	)
	return nil
}

func (a *PeerManagerImpl) SpawnOutgoingConnections(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.unsafeConnectedCount() > a.limitConnections*2 {
		return
	}
	var outCnt int
	a.active.forEach(
		func(_ peer.ID, info peerInfo) {
			if info.peer.Direction() == peer.Outgoing {
				outCnt += 1
			}
		},
	)

	a.logger.Debug("Spawning outgoing connections", "count", outCnt, "limit", a.limitConnections)

	if outCnt > a.limitConnections {
		return
	}

	if !a.enableOutboundConnections {
		return
	}

	known := a.KnownPeers()

	active := map[proto.IpPort]struct{}{}
	a.active.forEach(func(_ peer.ID, info peerInfo) {
		if info.peer.Direction() == peer.Outgoing {
			active[info.peer.RemoteAddr().ToIpPort()] = struct{}{}
		} else {
			if !info.peer.Handshake().DeclaredAddr.Empty() {
				active[info.peer.Handshake().DeclaredAddr.ToIpPort()] = struct{}{}
			}
		}
	})
	a.logger.Debug("Peer manager stats", "known", len(known), "active", len(active))
	for _, knowPeer := range known {
		ipPort := knowPeer.IpPort()
		if _, ok := active[ipPort]; ok {
			continue
		}
		if _, ok := a.spawned[ipPort]; ok {
			continue
		}
		if a.peerStorage.IsSuspendedIP(knowPeer.IP(), time.Now()) {
			continue
		}

		a.spawned[ipPort] = struct{}{}

		go func(ipPort proto.IpPort) {
			addr := proto.NewTCPAddr(ipPort.Addr(), ipPort.Port())
			defer a.removeSpawned(addr)
			if err := a.spawner.SpawnOutgoing(ctx, addr); err != nil {
				a.logger.Debug("Failed to establish outbound connection",
					slog.String("address", ipPort.String()), logging.Error(err))
			}
			if err := a.UpdateKnownPeers([]storage.KnownPeer{storage.KnownPeer(ipPort)}); err != nil {
				slog.Error("Failed to update peer info in peer storage",
					slog.String("address", ipPort.String()), logging.Error(err))
			}

		}(ipPort)
	}
}

func (a *PeerManagerImpl) SpawnIncomingConnection(ctx context.Context, conn net.Conn) error {
	return a.spawner.SpawnIncoming(ctx, conn)
}

func (a *PeerManagerImpl) Spawned() []proto.IpPort {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]proto.IpPort, 0, len(a.spawned))
	for k := range a.spawned {
		out = append(out, k)
	}
	return out
}

func (a *PeerManagerImpl) Connect(ctx context.Context, addr proto.TCPAddr) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	active := map[proto.IpPort]struct{}{}
	a.active.forEach(func(_ peer.ID, info peerInfo) {
		if info.peer.Direction() == peer.Outgoing {
			active[info.peer.RemoteAddr().ToIpPort()] = struct{}{}
		} else {
			if !info.peer.Handshake().DeclaredAddr.Empty() {
				active[info.peer.Handshake().DeclaredAddr.ToIpPort()] = struct{}{}
			}
		}
	})

	if _, ok := active[addr.ToIpPort()]; ok {
		return nil
	}

	if _, ok := a.spawned[addr.ToIpPort()]; ok {
		return nil
	}

	a.spawned[addr.ToIpPort()] = struct{}{}

	go func(addr proto.TCPAddr) {
		defer a.removeSpawned(addr)
		err := a.spawner.SpawnOutgoing(ctx, addr)
		if err != nil {
			slog.Error("Failed to spawn outgoing peer", slog.String("address", addr.String()),
				logging.Error(err))
		}
	}(addr)

	return nil
}

func (a *PeerManagerImpl) Score(p peer.Peer) (*proto.Score, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	info, ok := a.active.get(p.ID())
	if !ok {
		return nil, errors.New("peer not found")
	}
	return info.score, nil
}

func (a *PeerManagerImpl) AskPeers() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.active.forEach(func(_ peer.ID, info peerInfo) {
		info.peer.SendMessage(&proto.GetPeersMessage{})
	})
}

func (a *PeerManagerImpl) Disconnect(p peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	pid := p.ID()
	a.active.remove(pid)
	if err := p.Close(); err != nil {
		a.logger.Debug("Failed to disconnect the peer", slog.Any("peer", pid), logging.Error(err))
	} else {
		a.logger.Debug("Peer disconnected", "peer", pid)
	}
}

func (a *PeerManagerImpl) Run(ctx context.Context) {
	ticker := time.NewTicker(clearRestrictedPeersInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.clearRestrictedPeers(time.Now())
		}
	}
}

func (a *PeerManagerImpl) AddAddress(addr proto.TCPAddr) error {
	known := storage.KnownPeer(addr.ToIpPort())
	if err := a.peerStorage.AddOrUpdateKnown([]storage.KnownPeer{known}, time.Now()); err != nil {
		return errors.Wrapf(err, "failed to add addr %q into known peers storage", addr.String())
	}
	return nil
}

func (a *PeerManagerImpl) CheckPeerWithMaxScore(p peer.Peer) (peer.Peer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var pid peer.ID
	pIDStr := "n/a"
	if p != nil {
		pid = p.ID()
		pIDStr = p.ID().String()
	}
	cpi, ok := a.active.get(pid)
	if !ok {
		return nil, false
	}
	npi, ok := a.active.getPeerWithMaxScore()
	if !ok { // No need to change peer
		a.logger.Debug("No need to change peer with max score", "peer", pIDStr)
		return p, false
	}

	if cpi.score.Cmp(npi.score) < 0 { // npi has a bigger score - switch to it
		a.logger.Debug("Changing peer with max score", "from", pIDStr, "to", npi.peer.ID().String())
		return npi.peer, true
	}
	a.logger.Debug("No need to change peer with max score", "peer", pIDStr)
	return p, false // Otherwise stick to currently used peer
}

func (a *PeerManagerImpl) CheckPeerInLargestScoreGroup(p peer.Peer) (peer.Peer, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	pid := "n/a"
	if p != nil {
		pid = p.ID().String()
	}

	np, ok := a.active.getPeerFromLargestPeerGroup(p)
	if !ok { // No need to change peer
		a.logger.Debug("No need to change peer", "peer", pid)
		return p, false
	}
	a.logger.Debug("Changing best peer", "from", pid, "to", np.peer.ID().String())
	return np.peer, true
}

func (a *PeerManagerImpl) connected(p peer.Peer) (peer.Peer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	info, ok := a.active.get(p.ID())
	return info.peer, ok
}

// non thread safe
func (a *PeerManagerImpl) unsafeConnectedCount() int {
	return a.active.size()
}

func (a *PeerManagerImpl) clearRestrictedPeers(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.peerStorage.RefreshSuspended(now); err != nil {
		slog.Error("Failed to clear suspended peers", logging.Error(err))
	}
	if err := a.peerStorage.RefreshBlackList(now); err != nil {
		slog.Error("Failed to clear black listed peers", logging.Error(err))
	}
}

func (a *PeerManagerImpl) addConnected(peer peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.spawned, peer.RemoteAddr().ToIpPort())
	a.active.add(peer)
}

func (a *PeerManagerImpl) suspended(p peer.Peer, now time.Time) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ip := storage.IpFromIpPort(p.RemoteAddr().ToIpPort())
	return a.peerStorage.IsSuspendedIP(ip, now)
}

func (a *PeerManagerImpl) blackListed(p peer.Peer, now time.Time) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ip := storage.IpFromIpPort(p.RemoteAddr().ToIpPort())
	return a.peerStorage.IsBlackListedIP(ip, now)
}

func (a *PeerManagerImpl) restrict(p peer.Peer, now time.Time, reason string) {
	switch d := p.Direction(); d {
	case peer.Incoming:
		a.AddToBlackList(p, now, reason)
	case peer.Outgoing:
		a.Suspend(p, now, reason)
	default:
		panic(fmt.Sprintf("BUG, CREATE REPORT: can't restrict peer because of unexpected peer direction (%d)", d))
	}
}

// countDirections counts connected peers by its directions and returns number of inbound and outbound connections.
func (a *PeerManagerImpl) countDirections() (int, int) {
	in, out := 0, 0
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.active.forEach(func(_ peer.ID, info peerInfo) {
		if info.peer.Direction() == peer.Outgoing {
			out += 1
		} else {
			in += 1
		}
	})
	return in, out
}

func (a *PeerManagerImpl) removeSpawned(addr proto.TCPAddr) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.spawned, addr.ToIpPort())
}

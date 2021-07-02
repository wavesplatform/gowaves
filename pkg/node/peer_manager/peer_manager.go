package peer_manager

import (
	"context"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const suspendDuration = 5 * time.Minute

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
	Connected(peer.Peer) (peer.Peer, bool)
	NewConnection(peer.Peer) error
	ConnectedCount() int
	InOutCount() (in int, out int)
	EachConnected(func(peer.Peer, *proto.Score))
	IsSuspended(peer.Peer) bool
	Suspend(peer peer.Peer, suspendTime time.Time, reason string)
	Suspended() []storage.SuspendedPeer
	AddConnected(peer.Peer)
	UpdateScore(p peer.Peer, score *proto.Score) error
	UpdateKnownPeers([]storage.KnownPeer) error
	KnownPeers() []storage.KnownPeer
	Close()
	SpawnOutgoingConnections(context.Context)
	SpawnIncomingConnection(ctx context.Context, conn net.Conn) error
	Spawned() []proto.IpPort
	Connect(context.Context, proto.TCPAddr) error
	Score(p peer.Peer) (*proto.Score, error)

	// AskPeers sends GetPeersMessage message to all connected nodes.
	AskPeers()

	Disconnect(peer.Peer)
}

type PeerManagerImpl struct {
	spawner          PeerSpawner
	active           map[peer.Peer]peerInfo
	mu               sync.RWMutex
	peerStorage      PeerStorage
	spawned          map[proto.IpPort]struct{}
	connectPeers     bool // spawn outgoing
	limitConnections int
	version          proto.Version
	networkName      string
}

func NewPeerManager(spawner PeerSpawner, storage PeerStorage,
	limitConnections int, version proto.Version, networkName string) *PeerManagerImpl {

	return &PeerManagerImpl{
		spawner:          spawner,
		active:           make(map[peer.Peer]peerInfo),
		peerStorage:      storage,
		spawned:          make(map[proto.IpPort]struct{}),
		connectPeers:     true,
		limitConnections: limitConnections,
		version:          version,
		networkName:      networkName,
	}
}

func (a *PeerManagerImpl) Score(p peer.Peer) (*proto.Score, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	info, ok := a.active[p]
	if !ok {
		return nil, errors.New("peer not found")
	}
	return info.score, nil
}

func (a *PeerManagerImpl) SetConnectPeers(connect bool) {
	a.mu.Lock()
	a.connectPeers = connect
	zap.S().Debug("set connect peers to ", a.connectPeers)
	a.mu.Unlock()
}

func (a *PeerManagerImpl) Connected(p peer.Peer) (peer.Peer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p1, ok := a.active[p]
	return p1.peer, ok
}

func (a *PeerManagerImpl) ConnectedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.unsafeConnectedCount()
}

// non thread safe
func (a *PeerManagerImpl) unsafeConnectedCount() int {
	return len(a.active)
}

func (a *PeerManagerImpl) NewConnection(p peer.Peer) error {
	_, connected := a.Connected(p)
	if connected {
		_ = p.Close()
		return errors.New("already connected")
	}
	if a.IsSuspended(p) {
		_ = p.Close()
		return errors.Errorf("peer '%s' is suspended", p.ID())
	}
	if p.Handshake().Version.CmpMinor(a.version) >= 2 {
		err := errors.Errorf(
			"versions are too different, current %s, connected %s",
			a.version.String(),
			p.Handshake().Version.String(),
		)
		a.Suspend(p, time.Now(), err.Error())
		_ = p.Close()
		return proto.NewInfoMsg(err)
	}
	if p.Handshake().AppName != a.networkName {
		err := errors.Errorf("peer '%s' has the invalid network name '%s', required '%s'",
			p.ID(), p.Handshake().AppName, a.networkName)
		a.Suspend(p, time.Now(), err.Error())
		_ = p.Close()
		return proto.NewInfoMsg(err)
	}
	in, out := a.InOutCount()
	switch p.Direction() {
	case peer.Incoming:
		if in >= a.limitConnections {
			_ = p.Close()
			return proto.NewInfoMsg(errors.New("exceed incoming connections limit"))
		}
	case peer.Outgoing:
		if !p.Handshake().DeclaredAddr.Empty() {
			known := storage.KnownPeer(proto.TCPAddr(p.Handshake().DeclaredAddr).ToIpPort())
			// TODO(nickeskov): maybe log error?
			_ = a.peerStorage.AddKnown([]storage.KnownPeer{known})
		}
		if out >= a.limitConnections {
			_ = p.Close()
			return proto.NewInfoMsg(errors.New("exceed outgoing connections limit"))
		}
	default:
		_ = p.Close()
		return errors.New("unknown connection direction")
	}
	a.AddConnected(p)
	return nil
}

func (a *PeerManagerImpl) ClearSuspended(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.peerStorage.RefreshSuspended(now); err != nil {
		zap.S().Errorf("failed to clear suspended peers: %v", err)
	}
}

func (a *PeerManagerImpl) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
			a.ClearSuspended(time.Now())
		}
	}
}

func (a *PeerManagerImpl) AddConnected(peer peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.spawned, peer.RemoteAddr().ToIpPort())
	a.active[peer] = newPeerInfo(peer)
}

func (a *PeerManagerImpl) UpdateScore(p peer.Peer, score *big.Int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if row, ok := a.active[p]; ok {
		row.score = score
		a.active[p] = row
		return nil
	}
	return errors.Errorf("peer '%s' is not active", p.ID())
}

func (a *PeerManagerImpl) IsSuspended(p peer.Peer) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ip := storage.IpFromIpPort(p.RemoteAddr().ToIpPort())
	return a.peerStorage.IsSuspendedIP(ip, time.Now())
}

// InOutCount counts connected peers,
// in - incoming connections
// out - outgoing connections
func (a *PeerManagerImpl) InOutCount() (in int, out int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, v := range a.active {
		if v.peer.Direction() == peer.Outgoing {
			out += 1
		} else {
			in += 1
		}
	}
	return in, out
}

func (a *PeerManagerImpl) Suspend(p peer.Peer, suspendTime time.Time, reason string) {
	a.Disconnect(p)
	a.mu.Lock()
	defer a.mu.Unlock()
	suspended := storage.SuspendedPeer{
		IP:                     storage.IpFromIpPort(p.RemoteAddr().ToIpPort()),
		SuspendTimestampMillis: unixMillis(suspendTime),
		SuspendDuration:        suspendDuration,
		Reason:                 reason,
	}
	if err := a.peerStorage.AddSuspended([]storage.SuspendedPeer{suspended}); err != nil {
		zap.S().Errorf("[%s] Failed to suspend peer, reason %q: %v", p.ID(), reason, err)
	} else {
		zap.S().Debugf("[%s] Suspend peer, reason: %s ", p.ID(), reason)
	}
}

func (a *PeerManagerImpl) Suspended() []storage.SuspendedPeer {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.peerStorage.Suspended(time.Now())
}

func (a *PeerManagerImpl) AddAddress(ctx context.Context, addr proto.TCPAddr) error {
	known := storage.KnownPeer(addr.ToIpPort())
	if err := a.peerStorage.AddKnown([]storage.KnownPeer{known}); err != nil {
		return errors.Wrapf(err, "failed to add addr %q into known peers storage", addr.String())
	}
	go func() {
		if err := a.spawner.SpawnOutgoing(ctx, addr); err != nil {
			// TODO(nickeskov): maybe don't remove from known peers in this case?
			if removeErr := a.peerStorage.DeleteKnown([]storage.KnownPeer{known}); removeErr != nil {
				zap.S().Errorf("Failed to remove peer %q from known peers storage", known.String())
			}
			zap.S().Debug(err)
		}
	}()
	return nil
}

func (a *PeerManagerImpl) UpdateKnownPeers(known []storage.KnownPeer) error {
	if len(known) == 0 {
		return nil
	}

	if err := a.peerStorage.AddKnown(known); err != nil {
		return errors.Wrap(err, "failed to update known peers")
	}
	return nil
}

func (a *PeerManagerImpl) KnownPeers() []storage.KnownPeer {
	return a.peerStorage.Known()
}

func (a *PeerManagerImpl) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, v := range a.active {
		_ = v.peer.Close()
	}
}

func (a *PeerManagerImpl) SpawnOutgoingConnections(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.unsafeConnectedCount() > a.limitConnections*2 {
		return
	}
	var outCnt int
	for _, v := range a.active {
		if v.peer.Direction() == peer.Outgoing {
			outCnt += 1
		}
	}

	if outCnt > a.limitConnections {
		return
	}

	if !a.connectPeers {
		return
	}

	known := a.KnownPeers()

	active := map[proto.IpPort]struct{}{}
	for _, p := range a.active {
		if p.peer.Direction() == peer.Outgoing {
			active[p.peer.RemoteAddr().ToIpPort()] = struct{}{}
		} else {
			if !p.peer.Handshake().DeclaredAddr.Empty() {
				active[p.peer.Handshake().DeclaredAddr.ToIpPort()] = struct{}{}
			}
		}
	}

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
			defer a.RemoveSpawned(addr)
			// TODO(nickeskov): maybe log error?
			_ = a.spawner.SpawnOutgoing(ctx, addr)
		}(ipPort)
	}
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

func (a *PeerManagerImpl) SpawnIncomingConnection(ctx context.Context, conn net.Conn) error {
	return a.spawner.SpawnIncoming(ctx, conn)
}

func (a *PeerManagerImpl) RemoveSpawned(addr proto.TCPAddr) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.spawned, addr.ToIpPort())
}

func (a *PeerManagerImpl) AskPeers() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, p := range a.active {
		p.peer.SendMessage(&proto.GetPeersMessage{})
	}
}

func (a *PeerManagerImpl) Disconnect(p peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.active, p)
	_ = p.Close()
}

func (a *PeerManagerImpl) EachConnected(f func(peer peer.Peer, score *big.Int)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, row := range a.active {
		f(row.peer, row.score)
	}
}

func (a *PeerManagerImpl) Connect(ctx context.Context, addr proto.TCPAddr) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	active := map[proto.IpPort]struct{}{}
	for _, p := range a.active {
		if p.peer.Direction() == peer.Outgoing {
			active[p.peer.RemoteAddr().ToIpPort()] = struct{}{}
		} else {
			if !p.peer.Handshake().DeclaredAddr.Empty() {
				active[p.peer.Handshake().DeclaredAddr.ToIpPort()] = struct{}{}
			}
		}
	}

	if _, ok := active[addr.ToIpPort()]; ok {
		return nil
	}

	if _, ok := a.spawned[addr.ToIpPort()]; ok {
		return nil
	}

	a.spawned[addr.ToIpPort()] = struct{}{}

	go func(addr proto.TCPAddr) {
		defer a.RemoveSpawned(addr)
		err := a.spawner.SpawnOutgoing(ctx, addr)
		if err != nil {
			zap.S().Error(err)
		}
	}(addr)

	return nil
}

func unixMillis(now time.Time) int64 {
	return now.UnixNano() / 1_000_000
}

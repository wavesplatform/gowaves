package peer_manager

import (
	"context"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
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

type byScore []peerInfo

func (a byScore) Len() int           { return len(a) }
func (a byScore) Less(i, j int) bool { return a[i].score.Cmp(a[j].score) < 0 }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type PeerManager interface {
	Connected(peer.Peer) (peer.Peer, bool)
	NewConnection(peer.Peer) error
	ConnectedCount() int
	InOutCount() (in int, out int)
	EachConnected(func(peer.Peer, *proto.Score))
	IsSuspended(peer.Peer) bool
	Suspend(peer.Peer, string)
	Suspended() []string
	AddConnected(peer.Peer)
	PeerWithHighestScore() (peer.Peer, *big.Int, bool)
	UpdateScore(p peer.Peer, score *proto.Score) error
	UpdateKnownPeers([]proto.TCPAddr) error
	KnownPeers() ([]proto.TCPAddr, error)
	Close()
	SpawnOutgoingConnections(context.Context)
	SpawnIncomingConnection(ctx context.Context, conn net.Conn) error
	Spawned() []proto.IpPort
	Connect(context.Context, proto.TCPAddr) error
	Score(p peer.Peer) (*proto.Score, error)

	// for all connected node send GetPeersMessage
	AskPeers()

	Disconnect(peer.Peer)
}

type Ip = [net.IPv6len]byte

type suspended map[Ip]time.Time

func (a suspended) Blocked(ipPort proto.IpPort, now time.Time) bool {
	ip := Ip{}
	copy(ip[:], ipPort[:net.IPv6len])
	v, ok := a[ip]
	if !ok {
		return false
	}
	if v.Add(5 * time.Minute).After(now) { //suspended
		return true
	} else {
		return false
	}
}

func (a suspended) AllBlocked() []string {
	out := make([]string, 0, len(a))
	for ip := range a {
		out = append(out, net.IP(ip[:]).String())
	}
	return out
}

func (a suspended) clear(now time.Time) {
	for ip, v := range a {
		if v.Add(5 * time.Minute).Before(now) {
			delete(a, ip)
		}
	}
}

func (a suspended) Block(ip proto.IpPort, d time.Duration) {
	a[ipPortToIp(ip)] = time.Now().Add(d)
}

func ipPortToIp(ipPort proto.IpPort) [net.IPv6len]byte {
	ip := Ip{}
	copy(ip[:], ipPort[:net.IPv6len])
	return ip
}

func (a suspended) Len() int {
	return len(a)
}

type PeerManagerImpl struct {
	spawner          PeerSpawner
	active           map[peer.Peer]peerInfo
	mu               sync.RWMutex
	state            PeerStorage
	spawned          map[proto.IpPort]struct{}
	suspended        suspended
	connectPeers     bool // spawn outgoing
	limitConnections int
	version          proto.Version
}

func NewPeerManager(spawner PeerSpawner, storage PeerStorage, limitConnections int, version proto.Version) *PeerManagerImpl {
	return &PeerManagerImpl{
		spawner:          spawner,
		active:           make(map[peer.Peer]peerInfo),
		state:            storage,
		spawned:          make(map[proto.IpPort]struct{}),
		suspended:        suspended{},
		connectPeers:     true,
		limitConnections: limitConnections,
		version:          version,
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
	return a.connectedCount()
}

// non thread safe
func (a *PeerManagerImpl) connectedCount() int {
	return len(a.active)
}

func (a *PeerManagerImpl) NewConnection(p peer.Peer) error {
	_, connected := a.Connected(p)
	if connected {
		p.Close()
		return errors.New("already connected")
	}
	if a.IsSuspended(p) {
		p.Close()
		return errors.New("peer is suspended")
	}
	if p.Handshake().Version.CmpMinor(a.version) >= 2 {
		err := errors.Errorf(
			"versions are too different, current %s, connected %s",
			a.version.String(),
			p.Handshake().Version.String(),
		)
		a.Suspend(p, err.Error())
		p.Close()
		return err
	}

	in, out := a.InOutCount()
	switch p.Direction() {
	case peer.Incoming:
		if in >= a.limitConnections {
			_ = p.Close()
			return errors.New("exceed incoming connections limit")
		}
	case peer.Outgoing:
		if !p.Handshake().DeclaredAddr.Empty() {
			_ = a.state.AddKnown(proto.TCPAddr(p.Handshake().DeclaredAddr))
		}
		if out >= a.limitConnections {
			_ = p.Close()
			return errors.New("exceed outgoing connections limit")
		}
	default:
		_ = p.Close()
		return errors.New("unknown connection direction")
	}
	a.AddConnected(p)
	return nil
}

func (a *PeerManagerImpl) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
			a.mu.Lock()
			a.suspended.clear(time.Now())
			a.mu.Unlock()
		}
	}
}

func (a *PeerManagerImpl) AddConnected(peer peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.spawned, peer.RemoteAddr().ToIpPort())
	a.active[peer] = newPeerInfo(peer)
}

func (a *PeerManagerImpl) PeerWithHighestScore() (peer.Peer, *big.Int, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.active) == 0 {
		return nil, nil, false
	}

	var peers []peerInfo
	for _, p := range a.active {
		peers = append(peers, p)
	}

	sort.Sort(byScore(peers))

	highest := peers[len(peers)-1]
	return highest.peer, highest.score, true
}

func (a *PeerManagerImpl) UpdateScore(p peer.Peer, score *big.Int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if row, ok := a.active[p]; ok {
		row.score = score
		a.active[p] = row
	}
	return nil
}

func (a *PeerManagerImpl) IsSuspended(p peer.Peer) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.suspended.Blocked(p.RemoteAddr().ToIpPort(), time.Now())
}

// Count connected peers,
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

func (a *PeerManagerImpl) Suspend(p peer.Peer, reason string) {
	a.Disconnect(p)
	a.mu.Lock()
	a.suspended.Block(p.RemoteAddr().ToIpPort(), 5*time.Minute)
	a.mu.Unlock()
	zap.S().Debugf("[%s] Suspend peer, reason: %s ", p.ID(), reason)
}

func (a *PeerManagerImpl) Suspended() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.suspended.AllBlocked()
}

func (a *PeerManagerImpl) AddAddress(ctx context.Context, addr string) {
	_ = a.state.Add([]proto.TCPAddr{proto.NewTCPAddrFromString(addr)})
	go func() {
		if err := a.spawner.SpawnOutgoing(ctx, proto.NewTCPAddrFromString(addr)); err != nil {
			zap.S().Info(err)
		}
	}()
}

func (a *PeerManagerImpl) UpdateKnownPeers(known []proto.TCPAddr) error {
	if len(known) == 0 {
		return nil
	}
	return a.state.Add(known)
}

func (a *PeerManagerImpl) KnownPeers() ([]proto.TCPAddr, error) {
	return a.state.Known()
}

func (a *PeerManagerImpl) Close() {
	a.mu.Lock()
	for _, v := range a.active {
		v.peer.Close()
	}
	a.mu.Unlock()
}

func (a *PeerManagerImpl) SpawnOutgoingConnections(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.connectedCount() > a.limitConnections*2 {
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

	known, err := a.KnownPeers()
	if err != nil {
		zap.S().Error(err)
		return
	}

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

	for _, addr := range known {
		addrIpPort := addr.ToIpPort()
		if _, ok := active[addrIpPort]; ok {
			continue
		}
		if _, ok := a.spawned[addrIpPort]; ok {
			continue
		}
		if a.suspended.Blocked(addrIpPort, time.Now()) {
			continue
		}

		a.spawned[addr.ToIpPort()] = struct{}{}

		go func(addr proto.TCPAddr) {
			defer a.RemoveSpawned(addr)
			_ = a.spawner.SpawnOutgoing(ctx, addr)
		}(addr)
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

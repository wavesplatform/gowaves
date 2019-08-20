package peer_manager

import (
	"context"
	"math/big"
	"net"
	"sort"
	"sync"

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
	Connected(unique string) (peer.Peer, bool)
	EachConnected(func(peer.Peer, *proto.Score))
	Banned(unique string) bool
	AddConnected(p peer.Peer)
	PeerWithHighestScore() (peer.Peer, *big.Int, bool)
	UpdateScore(id string, score *proto.Score)
	UpdateKnownPeers([]proto.TCPAddr) error
	KnownPeers() ([]proto.TCPAddr, error)
	Close()
	SpawnOutgoingConnections(context.Context)
	SpawnIncomingConnection(ctx context.Context, conn net.Conn) error
	Connect(context.Context, proto.TCPAddr) error

	// for all connected node send GetPeersMessage
	AskPeers()

	Disconnect(id string)
}

type PeerManagerImpl struct {
	spawner    PeerSpawner
	active     map[string]peerInfo //peer.Peer
	knownPeers map[string]proto.Version
	mu         sync.RWMutex
	state      PeerStorage
	spawned    map[proto.IpPort]struct{}
}

func NewPeerManager(spawner PeerSpawner, storage PeerStorage) *PeerManagerImpl {
	return &PeerManagerImpl{
		spawner:    spawner,
		active:     make(map[string]peerInfo),
		knownPeers: make(map[string]proto.Version),
		state:      storage,
		spawned:    make(map[proto.IpPort]struct{}),
	}
}

func (a *PeerManagerImpl) Connected(unique string) (peer.Peer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p, ok := a.active[unique]
	return p.peer, ok
}

// TODO check remove spawned
func (a *PeerManagerImpl) AddConnected(peer peer.Peer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active[peer.ID()] = newPeerInfo(peer)
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

func (a *PeerManagerImpl) UpdateScore(id string, score *big.Int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	zap.S().Debugf("update score for %s, set %s", id, score.String())

	if row, ok := a.active[id]; ok {
		row.score = score
		a.active[id] = row
	} else {
		zap.S().Warnf("no peer with id %s found in active peers", id, a.active)
	}
}

// TODO implement banned logic
func (a *PeerManagerImpl) Banned(id string) bool {
	return false
}

func (a *PeerManagerImpl) AddAddress(ctx context.Context, addr string) {
	go func() {
		if err := a.spawner.SpawnOutgoing(ctx, proto.NewTCPAddrFromString(addr)); err != nil {
			zap.S().Error(err)
		}
	}()
}

func (a *PeerManagerImpl) UpdateKnownPeers(known []proto.TCPAddr) error {
	if len(known) == 0 {
		return nil
	}

	return a.state.SavePeers(known)
}

func (a *PeerManagerImpl) KnownPeers() ([]proto.TCPAddr, error) {
	rs, err := a.state.Peers()
	if err != nil {
		return nil, err
	}

	if len(rs) == 0 {
		return nil, nil
	}

	out := make([]proto.TCPAddr, len(rs))
	copy(out, rs)
	return out, nil
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
		if _, ok := active[addr.ToIpPort()]; ok {
			continue
		}

		if _, ok := a.spawned[addr.ToIpPort()]; ok {
			continue
		}

		a.spawned[addr.ToIpPort()] = struct{}{}

		go func(addr proto.TCPAddr) {
			defer a.RemoveSpawned(addr)
			err := a.spawner.SpawnOutgoing(ctx, addr)
			if err != nil {
				zap.S().Error(err)
			}
		}(addr)
	}
}

func (a *PeerManagerImpl) SpawnIncomingConnection(ctx context.Context, conn net.Conn) error {
	if err := a.spawner.SpawnIncoming(ctx, conn); err != nil {
		return err
	}
	return nil
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

func (a *PeerManagerImpl) Disconnect(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.active[id]
	if ok {
		p.peer.Close()
		delete(a.active, id)
	}
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

package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"math/big"
	"sort"
	"sync"
)

var defaultVersion = proto.Version{0, 15, 0}

type peerInfo struct {
	score *big.Int
	peer  peer.Peer
}

type byScore []peerInfo

func (a byScore) Len() int           { return len(a) }
func (a byScore) Less(i, j int) bool { return a[i].score.Cmp(a[j].score) < 0 }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type PeerManager interface {
	Connected(unique string) (peer.Peer, bool)
	Banned(unique string) bool
	AddConnected(p peer.Peer)
	PeerWithHighestScore() (peer.Peer, *big.Int, bool)
	UpdateScore(id string, score *big.Int)
	UpdateKnownPeers([]proto.PeerInfo) error
	KnownPeers() ([]proto.PeerInfo, error)
	Close()
}

type PeerManagerImpl struct {
	spawner    PeerSpawner
	active     map[string]peerInfo //peer.Peer
	knownPeers map[string]proto.Version
	mu         sync.RWMutex
	state      state.State
}

func NewPeerManager(spawner PeerSpawner, state state.State) *PeerManagerImpl {
	return &PeerManagerImpl{
		spawner:    spawner,
		active:     make(map[string]peerInfo),
		knownPeers: make(map[string]proto.Version),
		state:      state,
	}
}

func (a *PeerManagerImpl) Connected(unique string) (peer.Peer, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p, ok := a.active[unique]
	return p.peer, ok
}

func (a *PeerManagerImpl) AddConnected(peer peer.Peer) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.active[peer.ID()] = peerInfo{peer: peer}
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
	a.mu.RLock()
	defer a.mu.RUnlock()

	zap.S().Debugf("update score for %s, set %s", id, score.String())

	if row, ok := a.active[id]; ok {
		row.score = score
		a.active[id] = row
	} else {
		zap.S().Warnf("no peer with id %s found in active peers", id)
	}
}

// TODO implement banned logic
func (a *PeerManagerImpl) Banned(id string) bool {
	return false
}

func (a *PeerManagerImpl) AddAddress(ctx context.Context, addr string) {
	go a.spawner.SpawnOutgoing(ctx, addr, defaultVersion)
}

func (a *PeerManagerImpl) UpdateKnownPeers(known []proto.PeerInfo) error {
	if len(known) == 0 {
		return nil
	}

	peers := make([]state.KnownPeer, len(known))
	for idx, p := range known {
		peers[idx] = state.KnownPeer{
			IP:   p.Addr,
			Port: p.Port,
		}
	}

	return a.state.SavePeers(peers)
}

func (a *PeerManagerImpl) KnownPeers() ([]proto.PeerInfo, error) {
	rs, err := a.state.Peers()
	if err != nil {
		return nil, err
	}

	if len(rs) == 0 {
		return nil, nil
	}

	out := make([]proto.PeerInfo, len(rs))
	for idx, p := range rs {
		out[idx] = proto.PeerInfo{
			Addr: p.IP,
			Port: p.Port,
		}
	}

	return out, nil
}

func (a *PeerManagerImpl) Close() {
	a.mu.Lock()
	for _, v := range a.active {
		v.peer.Close()
	}
	a.mu.Unlock()
}

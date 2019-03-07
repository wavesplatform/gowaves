package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"sort"
	"sync"
)

var defaultVersion = proto.Version{0, 15, 0}

type peerInfo struct {
	score uint64
	peer  peer.Peer
}

type byScore []peerInfo

func (a byScore) Len() int           { return len(a) }
func (a byScore) Less(i, j int) bool { return a[i].score < a[j].score }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type PeerManager interface {
	Connected(unique string) (peer.Peer, bool)
	Banned(unique string) bool
	AddConnected(p peer.Peer)
	PeerWithHighestScore() (peer.Peer, uint64, bool)
}

type PeerManagerImpl struct {
	spawner    PeerSpawner
	active     map[string]peerInfo //peer.Peer
	knownPeers map[string]proto.Version
	mu         sync.RWMutex
}

func NewPeerManager(spawner PeerSpawner) *PeerManagerImpl {
	return &PeerManagerImpl{
		spawner:    spawner,
		active:     make(map[string]peerInfo),
		knownPeers: make(map[string]proto.Version),
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

func (a *PeerManagerImpl) PeerWithHighestScore() (peer.Peer, uint64, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.active) == 0 {
		return nil, 0, false
	}

	var peers []peerInfo
	for _, p := range a.active {
		peers = append(peers, p)
	}

	sort.Sort(byScore(peers))

	highest := peers[len(peers)-1]
	return highest.peer, highest.score, true
}

func (a *PeerManagerImpl) UpdateScore(id string, score uint64) {
	a.mu.RLock()
	defer a.mu.RUnlock()

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

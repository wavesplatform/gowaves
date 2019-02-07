package utils

import (
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerInfo struct {
	Peer       peer.Peer
	CreatedAt  time.Time
	Status     int
	Version    proto.Version
	DeclAddr   proto.PeerInfo
	RemoteAddr string
	LocalAddr  string
	LastError  struct {
		At    time.Time
		Error error
	}
}

// Active peers
type Addr2Peers struct {
	addr2peer map[string]*PeerInfo
	lock      sync.RWMutex
}

func NewAddr2Peers() *Addr2Peers {
	return &Addr2Peers{
		addr2peer: make(map[string]*PeerInfo),
	}
}

func (a *Addr2Peers) Exists(id string) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	_, ok := a.addr2peer[id]
	return ok
}

func (a *Addr2Peers) Add(address string, info *PeerInfo) {
	a.lock.Lock()
	a.addr2peer[address] = info
	a.lock.Unlock()
}

func (a *Addr2Peers) Addresses() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var out []string
	for addr := range a.addr2peer {
		out = append(out, addr)
	}
	return out
}

func (a *Addr2Peers) Each(f func(id string, p *PeerInfo)) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	for id, p := range a.addr2peer {
		f(id, p)
	}
}

func (a *Addr2Peers) Get(id string) *PeerInfo {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.addr2peer[id]
}

func (a *Addr2Peers) Delete(id string) {
	a.lock.RLock()
	delete(a.addr2peer, id)
	a.lock.RUnlock()
}

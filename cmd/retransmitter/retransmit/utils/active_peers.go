package utils

import (
	"sync"

	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

// Active peers
type Addr2Peers struct {
	addr2peer map[string]Peer
	lock      sync.RWMutex
}

func NewAddr2Peers() *Addr2Peers {
	return &Addr2Peers{
		addr2peer: make(map[string]Peer),
	}
}

// check address already exists
func (a *Addr2Peers) Exists(address string) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	_, ok := a.addr2peer[address]
	return ok
}

// add address to known list
func (a *Addr2Peers) Add(address string, peer Peer) {
	a.lock.Lock()
	a.addr2peer[address] = peer
	a.lock.Unlock()
}

// get all known addresses
func (a *Addr2Peers) Addresses() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var out []string
	for addr := range a.addr2peer {
		out = append(out, addr)
	}
	return out
}

// execute function with each address
func (a *Addr2Peers) Each(f func(id string, p Peer)) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	for id, p := range a.addr2peer {
		f(id, p)
	}
}

// returns *PeerInfo by address, nil if not found
func (a *Addr2Peers) Get(id string) Peer {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.addr2peer[id]
}

// delete address
func (a *Addr2Peers) Delete(address string) {
	a.lock.Lock()
	delete(a.addr2peer, address)
	a.lock.Unlock()
}

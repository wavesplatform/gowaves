package utils

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Active peers
type Addr2Peers struct {
	addr2peer map[peer.Peer]proto.IpPort
	p2peer    map[proto.IpPort]peer.Peer
	lock      sync.RWMutex
}

func NewAddr2Peers() *Addr2Peers {
	return &Addr2Peers{
		addr2peer: make(map[peer.Peer]proto.IpPort),
		p2peer:    make(map[proto.IpPort]peer.Peer),
	}
}

// check address already exists
func (a *Addr2Peers) Exists(address string) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	_, ok := a.p2peer[proto.NewTCPAddrFromString(address).ToIpPort()]
	return ok
}

// add address to known list
func (a *Addr2Peers) Add(address string, peer peer.Peer) {
	addr := proto.NewTCPAddrFromString(address)
	a.lock.Lock()
	a.addr2peer[peer] = addr.ToIpPort()
	a.p2peer[addr.ToIpPort()] = peer
	a.lock.Unlock()
}

// get all known addresses
func (a *Addr2Peers) Addresses() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var out []string
	for addr := range a.p2peer {
		out = append(out, addr.String())
	}
	return out
}

// execute function with each address
func (a *Addr2Peers) Each(f func(p peer.Peer)) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	for id := range a.addr2peer {
		f(id)
	}
}

// returns *PeerInfo by address, nil if not found
func (a *Addr2Peers) Get(p peer.Peer) peer.Peer {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if _, ok := a.addr2peer[p]; ok {
		return p
	}
	return nil
}

// delete address
func (a *Addr2Peers) Delete(p peer.Peer) {
	a.lock.Lock()
	delete(a.addr2peer, p)
	delete(a.p2peer, p.RemoteAddr().ToIpPort())
	a.lock.Unlock()
}

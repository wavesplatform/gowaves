package retransmit

import (
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"sync"
)

type Addr2Peers struct {
	addr2peer map[peer.UniqID]*PeerInfo
	lock      sync.RWMutex
}

func NewAddr2Peers() *Addr2Peers {
	return &Addr2Peers{
		addr2peer: make(map[peer.UniqID]*PeerInfo),
	}
}

func (a *Addr2Peers) Exists(id peer.UniqID) bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	_, ok := a.addr2peer[id]
	return ok
}

func (a *Addr2Peers) Add(id peer.UniqID, info *PeerInfo) {
	a.lock.Lock()
	a.addr2peer[id] = info
	a.lock.Unlock()
}

func (a *Addr2Peers) Addresses() []proto.PeerInfo {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var out []proto.PeerInfo
	for addr := range a.addr2peer {
		rs, err := proto.NewPeerInfoFromString(string(addr))
		if err != nil {
			fmt.Println(err)
			continue
		}
		out = append(out, rs)
	}
	return out
}

func (a *Addr2Peers) Each(f func(id peer.UniqID, p *PeerInfo)) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	for id, p := range a.addr2peer {
		f(id, p)
	}
}

func (a *Addr2Peers) Get(id peer.UniqID) *PeerInfo {
	a.lock.RLock()
	defer a.lock.Unlock()
	return a.addr2peer[id]
}

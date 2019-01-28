package retransmit

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"io/ioutil"
	"os"
	"sync"
)

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

func (a *Addr2Peers) Add(id string, info *PeerInfo) {
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

type KnownPeers struct {
	knownPeers map[string]proto.Version
	lock       sync.Mutex
	f          afero.File
}

func NewKnownPeers() *KnownPeers {
	return &KnownPeers{
		knownPeers: make(map[string]proto.Version),
	}
}

func (a *KnownPeers) Add(peer string, version proto.Version) {
	a.lock.Lock()
	a.knownPeers[peer] = version
	a.lock.Unlock()
}

func (a *KnownPeers) GetAll() []string {
	a.lock.Lock()
	defer a.lock.Unlock()
	var out []string
	for k := range a.knownPeers {
		out = append(out, k)
	}
	return out
}

func (a *KnownPeers) save() error {
	if a.f == nil {
		return errors.New("no file")
	}

	_, err := a.f.Seek(0, 0)
	if err != nil {
		return err
	}

	var out []JsonKnowPeerRow
	for k, v := range a.knownPeers {
		out = append(out, JsonKnowPeerRow{
			Addr:    k,
			Version: v,
		})
	}

	bts, err := json.Marshal(&out)
	if err != nil {
		return err
	}

	_, err = a.f.Write(bts)
	if err != nil {
		return err
	}

	return nil
}

func (a *KnownPeers) Stop() {
	if a.f == nil {
		return
	}

	err := a.save()
	if err != nil {
		fmt.Println(err)
	}
	_ = a.f.Close()
}

type JsonKnowPeerRow struct {
	Addr    string
	Version proto.Version
}

func NewKnownPeersFileBased(fs afero.Fs, pathToFile string) (*KnownPeers, error) {
	f, err := fs.OpenFile(pathToFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	bts, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var rows []JsonKnowPeerRow
	_ = json.Unmarshal(bts, &rows)

	p := NewKnownPeers()
	for _, row := range rows {
		p.knownPeers[row.Addr] = row.Version
	}

	p.f = f
	return p, nil
}

type SpawnedPeers struct {
	addrs map[string]struct{}
	mu    sync.Mutex
}

func NewSpawnedPeers() *SpawnedPeers {
	return &SpawnedPeers{
		addrs: make(map[string]struct{}),
	}
}

func (a *SpawnedPeers) Add(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.addrs[addr] = struct{}{}
}

func (a *SpawnedPeers) Exists(addr string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.addrs[addr]
	return ok
}

func (a *SpawnedPeers) Delete(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.addrs, addr)
}

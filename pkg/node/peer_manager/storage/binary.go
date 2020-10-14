package storage

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Peers []proto.TCPAddr

func (a Peers) Len() int           { return len(a) }
func (a Peers) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Peers) Less(i, j int) bool { return a[i].ToUint64() < a[j].ToUint64() }

type BinaryStorage struct {
	lock       sync.Mutex
	statePath  string
	allCache   Peers
	knownCache Peers
}

func (a *BinaryStorage) all() string {
	known := path.Join(a.statePath, "blocks_storage", "peers_all.dat")
	return known
}

func (a *BinaryStorage) known() string {
	known := path.Join(a.statePath, "blocks_storage", "peers_known.dat")
	return known
}

func (a *BinaryStorage) All() ([]proto.TCPAddr, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	err := a.loadCache(&a.allCache, a.all())
	if err != nil {
		return nil, err
	}
	out := make([]proto.TCPAddr, len(a.allCache))
	copy(out, a.allCache)
	return out, nil
}

func (a *BinaryStorage) loadCache(cache *Peers, file string) error {
	if len(*cache) > 0 {
		return nil
	}
	bts, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var peers Peers
	for len(bts) >= 8 {
		val := binary.BigEndian.Uint64(bts[:8])
		peers = append(peers, proto.NewTcpAddrFromUint64(val))
		bts = bts[8:]
	}
	*cache = peers
	return nil
}

func (a *BinaryStorage) Known() ([]proto.TCPAddr, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	err := a.loadCache(&a.knownCache, a.known())
	if err != nil {
		return nil, err
	}
	out := make([]proto.TCPAddr, len(a.knownCache))
	copy(out, a.knownCache)
	return out, nil
}

func (a *BinaryStorage) AddKnown(new proto.TCPAddr) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	err := a.loadCache(&a.knownCache, a.known())
	if err != nil {
		return err
	}
	a.knownCache = append(a.knownCache, new)
	sort.Sort(a.knownCache)
	err = a.save(a.known(), a.knownCache)
	if err != nil {
		return err
	}
	a.knownCache = nil
	return nil
}

func (a *BinaryStorage) Add(addrs []proto.TCPAddr) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	err := a.loadCache(&a.allCache, a.all())
	if err != nil {
		return err
	}
	a.allCache = append(a.allCache, addrs...)
	sort.Sort(a.allCache)
	err = a.save(a.all(), a.allCache)
	if err != nil {
		return err
	}
	a.allCache = nil
	return nil
}

func (a *BinaryStorage) save(file string, peers Peers) error {
	buf := bytes.Buffer{}
	prev := uint64(0)
	for _, peer := range peers {
		cur := peer.ToUint64()
		if cur == prev {
			continue
		}
		prev = cur
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, cur)
		buf.Write(b)
	}
	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

func NewBinaryStorage(statePath string) *BinaryStorage {
	return &BinaryStorage{
		statePath: statePath,
		lock:      sync.Mutex{},
	}
}

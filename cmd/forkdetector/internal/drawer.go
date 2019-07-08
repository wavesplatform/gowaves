package internal

import (
	"github.com/seiflotfy/cuckoofilter"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sync"
)

const (
	filterCapacity  = 10 * 1024 * 1024
	blocksBatchSize = 100
)

type safeCuckooFilter struct {
	once   sync.Once
	mu     sync.RWMutex
	filter *cuckoo.Filter
}

func (f *safeCuckooFilter) insert(d []byte) bool {
	f.once.Do(func() {
		f.filter = cuckoo.NewFilter(filterCapacity)
	})
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.filter.InsertUnique(d)
}

func (f *safeCuckooFilter) lookup(d []byte) bool {
	f.once.Do(func() {
		f.filter = cuckoo.NewFilter(filterCapacity)
	})
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.filter.Lookup(d)
}

type drawerStats struct {
	total int
	short int
	long  int
}

type drawer struct {
	storage  *storage
	registry *Registry
	filter   *safeCuckooFilter
	graph    *graph
	longest  []uint32
	top      uint32
}

func NewDrawer(storage *storage, registry *Registry) (*drawer, error) {
	g := newGraph()
	f := new(safeCuckooFilter)
	it, err := storage.newBlockLinkIterator()
	if err != nil {
		return nil, err
	}
	count := 0
	for it.next() {
		from, to, sig := it.value()
		g.edge(from, to)
		f.insert(sig[:])
		count++
	}
	zap.S().Debugf("[DRA] State restored, %d blocks loaded", count)
	return &drawer{
		storage:  storage,
		registry: registry,
		filter:   f,
		graph:    g,
	}, nil
}

func (d *drawer) front(peer net.IP) ([]crypto.Signature, error) {
	zap.S().Debugf("[DRA] Requesting front blocks for %s", peer.String())
	return d.storage.frontBlocks(peer, blocksBatchSize)
}

func (d *drawer) movePeer(peer net.IP, signature crypto.Signature) error {
	zap.S().Debugf("[DRA] Moving peer '%s' to '%s'", peer.String(), signature.String())
	return d.storage.movePeerLeash(peer, signature)
}

func (d *drawer) hasBlock(signature crypto.Signature) (bool, error) {
	if !d.filter.lookup(signature[:]) { // A block with such signature is definitely unseen
		return false, nil
	}
	_, ok, err := d.storage.block(signature)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (d *drawer) appendBlock(block *proto.Block) error {
	zap.S().Debugf("[DRA] Appending block '%s'", block.BlockSignature.String())
	from, to, err := d.storage.appendBlock(block)
	if err != nil {
		return err
	}
	d.graph.edge(from, to)
	d.filter.insert(block.BlockSignature[:])
	return nil
}

func (d *drawer) forks() ([]Fork, error) {
	lastBlocks, err := d.storage.peersLastBlocks()
	if err != nil {
		return nil, err
	}

	paths, top := d.allPeersPathsWithTop(lastBlocks)

	forks := make(map[uint32][]uint32)
	longest := paths[top]
	forks[1] = []uint32{top}
	for v, p := range paths {
		if v == top {
			continue
		}
		common := d.graph.pathsIntersection(longest, p)
		if common == v {
			forks[1] = append(forks[1], v)
			// On the same fork
		} else {
			// On different fork, combine by common block
			forks[common] = append(forks[common], v)
		}
	}

	result := make([]Fork, 0)
	//for k, v := range forks {
	//
	//	block := d.storage.getSignature()
	//	fork := Fork{
	//		HeadBlock: block,
	//		Longest:   k == 1,
	//		CommonBlock:k,
	//
	//	}
	//	result = append(result)
	//}
	return result, nil
}

func (d *drawer) allPeersPathsWithTop(lastBlocks map[uint32][]net.IP) (map[uint32][]uint32, uint32) {
	paths := make(map[uint32][]uint32)
	var top uint32
	longest := 0
	for k := range lastBlocks {
		paths[k] = d.graph.path(k)
		if l := len(paths[k]); l > longest {
			longest = l
			top = k
		}
	}
	return paths, top
}

func (d *drawer) fork(peer net.IP) (Fork, error) {
	//n, err := d.storage.peerLastBlock(peer)
	//if err != nil {
	//	return Fork{}, err
	//}
	//
	//path := d.graph.path(n)
	//
	return Fork{}, nil
}

func (d *drawer) stats() *drawerStats {
	return &drawerStats{}
}

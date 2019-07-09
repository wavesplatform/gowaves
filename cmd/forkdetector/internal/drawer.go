package internal

import (
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"
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
	blocks int
	short  int
	long   int
}

type drawer struct {
	storage  *storage
	registry *Registry
	filter   *safeCuckooFilter
	mu       sync.RWMutex
	graph    *graph
	st       *drawerStats
}

func NewDrawer(storage *storage, registry *Registry) (*drawer, error) {
	zap.S().Info("[DRA] Restoring state...")
	start := time.Now()
	g := newGraph()
	f := new(safeCuckooFilter)
	it, err := storage.newBlockLinkIterator()
	if err != nil {
		return nil, err
	}
	defer it.close()
	count := 0
	for it.next() {
		from, to, sig := it.value()
		g.edge(from, to)
		f.insert(sig[:])
		count++
	}
	zap.S().Infof("[DRA] State restored, %d blocks loaded in %s", count, time.Since(start))
	return &drawer{
		storage:  storage,
		registry: registry,
		filter:   f,
		graph:    g,
		st:       &drawerStats{blocks: count},
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
	d.mu.Lock()
	d.graph.edge(from, to)
	d.st.blocks++
	d.mu.Unlock()
	d.filter.insert(block.BlockSignature[:])
	return nil
}

func (d *drawer) forks() ([]Fork, error) {
	lastBlocks, err := d.storage.peersLastBlocks()
	if err != nil {
		return nil, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	blocks := make([]uint32, len(lastBlocks))
	i := 0
	for key := range lastBlocks {
		blocks[i] = key
		i++
	}

	forks := d.graph.forks(blocks)
	result := make([]Fork, len(forks))
	for i, f := range forks {
		head, err := d.storage.link(f.top)
		if err != nil {
			return nil, err
		}
		common, err := d.storage.link(f.common)
		if err != nil {
			return nil, err
		}
		peers := make([]PeerForkInfo, 0)
		for n, l := range f.lags {
			ips, ok := lastBlocks[n]
			zap.S().Debugf("[DRA] FORK: %s; BLOCK#: %d; LAG: %d; IPS: %v", head.signature.String(), n, l, ips)
			if !ok {
				return nil, errors.Errorf("failure to collect peers for block #d", n)
			}
			for _, ip := range ips {
				peer, err := d.storage.peer(ip)
				if err != nil {
					return nil, errors.Wrap(err, "failed to collect peers")
				}
				pi := PeerForkInfo{
					Peer:    ip,
					Lag:     l,
					Name:    peer.Name,
					Version: peer.Version,
				}
				peers = append(peers, pi)
			}
		}
		fork := Fork{
			Longest:          i == 0,
			Height:           int(head.height),
			HeadBlock:        head.signature,
			LastCommonHeight: int(common.height),
			LastCommonBlock:  common.signature,
			Length:           f.length,
			Peers:            peers,
		}
		result[i] = fork
	}
	return result, nil
}

func (d *drawer) allPeersPathsWithTop(lastBlocks map[uint32][]net.IP) (map[uint32][]uint32, uint32) {
	paths := make(map[uint32][]uint32)
	var top uint32
	longest := 0
	for k := range lastBlocks {
		d.mu.RLock()
		paths[k] = d.graph.path(k)
		d.mu.RUnlock()
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.st
}

package internal

import (
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
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
	storage *storage
	filter  *safeCuckooFilter
	mu      sync.RWMutex
	graph   *graph
	st      *drawerStats
}

func NewDrawer(storage *storage) (*drawer, error) {
	zap.S().Info("Restoring state...")
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
		from, to, id := it.value()
		g.edge(from, to)
		f.insert(id.Bytes())
		count++
	}
	zap.S().Infof("State restored, %d blocks loaded in %s", count, time.Since(start))
	return &drawer{
		storage: storage,
		filter:  f,
		graph:   g,
		st:      &drawerStats{blocks: count},
	}, nil
}

func (d *drawer) front(peer net.IP) ([]proto.BlockID, error) {
	zap.S().Debugf("[DRA] Requesting front blocks for %s", peer.String())
	return d.storage.frontBlocks(peer, blocksBatchSize)
}

func (d *drawer) movePeer(peer net.IP, id proto.BlockID) error {
	zap.S().Debugf("[DRA] Moving peer '%s' to '%s'", peer.String(), id.String())
	return d.storage.movePeerLeash(peer, id)
}

func (d *drawer) hasBlock(id proto.BlockID) (bool, error) {
	if !d.filter.lookup(id.Bytes()) { // A block with such id is definitely unseen
		return false, nil
	}
	_, ok, err := d.storage.block(id)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (d *drawer) appendBlock(block *proto.Block) error {
	zap.S().Debugf("[DRA] Appending block '%s'", block.BlockID().String())
	from, to, err := d.storage.appendBlock(block)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.graph.edge(from, to)
	d.st.blocks++
	d.mu.Unlock()
	d.filter.insert(block.BlockID().Bytes())
	return nil
}

func (d *drawer) combineForks(forks []fork, peersByBlocks map[uint32][]net.IP) ([]Fork, error) {
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
			ips, ok := peersByBlocks[n]
			if !ok {
				return nil, errors.Errorf("failure to collect peers for block %d", n)
			}
			for _, ip := range ips {
				peer, err := d.storage.peer(ip)
				if err != nil {
					if err == leveldb.ErrNotFound {
						zap.S().Warnf("[DRA] Peer '%s' not found", ip.String())
						continue
					}
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
			HeadBlock:        head.id,
			LastCommonHeight: int(common.height),
			LastCommonBlock:  common.id,
			Length:           f.length,
			Peers:            peers,
		}
		result[i] = fork
	}
	return result, nil
}

func (d *drawer) extractBlockNumbers(peersByBlocks map[uint32][]net.IP) []uint32 {
	blocks := make([]uint32, len(peersByBlocks))
	i := 0
	for key := range peersByBlocks {
		blocks[i] = key
		i++
	}
	return blocks
}

func (d *drawer) forks(addresses []net.IP) ([]Fork, error) {
	lastBlocks, err := d.storage.peersLastBlocks(d.buildFilter(addresses))
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	forks := d.graph.forks(d.extractBlockNumbers(lastBlocks))
	return d.combineForks(forks, lastBlocks)
}

func (d *drawer) fork(ip net.IP, addresses []net.IP) ([]Fork, error) {
	lastBlocks, err := d.storage.peersLastBlocks(d.buildFilter(addresses))
	if err != nil {
		return nil, err
	}
	for k, v := range lastBlocks {
		if !d.containsIP(v, ip) {
			delete(lastBlocks, k)
		}
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	forks := d.graph.forks(d.extractBlockNumbers(lastBlocks))
	return d.combineForks(forks, lastBlocks)
}

func (d *drawer) containsIP(addresses []net.IP, ip net.IP) bool {
	for _, a := range addresses {
		if ip.To16().Equal(a.To16()) {
			return true
		}
	}
	return false
}

func (d *drawer) buildFilter(addresses []net.IP) func(ip net.IP) bool {
	addrMap := make(map[uint64]struct{})
	for _, a := range addresses {
		addrMap[hash(a.To16())] = struct{}{}
	}
	return func(ip net.IP) bool {
		_, ok := addrMap[hash(ip.To16())]
		return ok
	}
}

func (d *drawer) stats() *drawerStats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.st
}

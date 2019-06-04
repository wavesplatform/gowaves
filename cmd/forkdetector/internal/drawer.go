package internal

import (
	"github.com/seiflotfy/cuckoofilter"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"sync"
)

const (
	filterCapacity  = 10 * 1000000
	blocksBatchSize = 100
)

type drawerStats struct {
	total int
	short int
	long  int
}

type drawer struct {
	storage  *storage
	registry *Registry
	mu       sync.Mutex
	filter   *cuckoo.Filter
	graph    *graph
}

func NewDrawer(storage *storage, registry *Registry) *drawer {
	//TODO: restore state
	return &drawer{
		storage:  storage,
		registry: registry,
		filter:   cuckoo.NewFilter(filterCapacity),
		graph:    newGraph(),
	}
}

func (d *drawer) front(peer net.IP) ([]crypto.Signature, error) {
	return d.storage.frontBlocks(peer, blocksBatchSize)
}

func (d *drawer) movePeer(peer net.IP, signature crypto.Signature) error {
	return nil
}

func (d *drawer) number(signature crypto.Signature) (uint32, bool) {
	return 0, false
}

func (d *drawer) appendBlock(block proto.Block) error {
	return nil
}

func (d *drawer) forks() ([]Fork, error) {
	return nil, nil
}

func (d *drawer) fork(peer net.IP) (Fork, error) {
	return Fork{}, nil
}

func (d *drawer) stats() *drawerStats {
	return &drawerStats{}
}

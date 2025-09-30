package endorsementpool

import (
	"container/heap"
	"errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"sync"
)

const endorsementPoolLimit = 128

type key struct {
	blockID       [32]byte
	endorserIndex int32
}

func makeKey(blockID []byte, idx int32) key {
	var k key
	copy(k.blockID[:], blockID)
	k.endorserIndex = idx
	return k
}

type heapItemEndorsement struct {
	eb    *proto.EndorseBlock
	seq   uint64 // insertion sequence for FIFO priority
	index int    // position in heap
}

type endorsementHeap []*heapItemEndorsement

func (h endorsementHeap) Len() int { return len(h) }

// Oldest = smallest seq → floats to top
func (h endorsementHeap) Less(i, j int) bool {
	return h[i].seq < h[j].seq
}

func (h endorsementHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *endorsementHeap) Push(x any) {
	it := x.(*heapItemEndorsement)
	it.index = len(*h)
	*h = append(*h, it)
}

func (h *endorsementHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	it := old[n-1]
	it.index = -1
	old[n-1] = nil
	*h = old[:n-1]
	return it
}

type EndorseVerifier interface {
	Verify(eb *proto.EndorseBlock) bool
}

type EndorsementPool struct {
	mu         sync.Mutex
	seq        uint64
	countLimit int
	byKey      map[key]*heapItemEndorsement
	h          endorsementHeap
}

func NewEndorsementPool() *EndorsementPool {
	return &EndorsementPool{
		countLimit: endorsementPoolLimit,
		byKey:      make(map[key]*heapItemEndorsement),
	}
}

func (p *EndorsementPool) Add(e *proto.EndorseBlock) error {
	if e == nil || len(e.EndorsedBlockId) == 0 {
		return errors.New("invalid endorsed block id")
	}
	k := makeKey(e.EndorsedBlockId, e.EndorserIndex)

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.byKey[k]; exists {
		return errors.New("duplicate endorsement")
	}

	// Evict oldest if full
	if p.countLimit > 0 && len(p.byKey) >= p.countLimit {
		return errors.New("the endorsement pool is full")
	}

	p.seq++
	it := &heapItemEndorsement{eb: e, seq: p.seq}
	heap.Push(&p.h, it)
	p.byKey[k] = it
	return nil
}

func (p *EndorsementPool) GetAll() []*proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]*proto.EndorseBlock, 0, len(p.h))
	for _, it := range p.h {
		if it != nil {
			out = append(out, it.eb)
		}
	}
	return out
}

func (p *EndorsementPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.byKey)
}

func (p *EndorsementPool) CleanAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byKey = make(map[key]*heapItemEndorsement)
	p.h = endorsementHeap{} // reset heap too
}
func (p *EndorsementPool) Pop() *proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.h) == 0 {
		return nil
	}
	it := heap.Pop(&p.h).(*heapItemEndorsement)
	k := makeKey(it.eb.EndorsedBlockId, it.eb.EndorserIndex)
	delete(p.byKey, k)
	return it.eb
}

func (p *EndorsementPool) FindByBlockID(blockID proto.BlockID) ([]*proto.EndorseBlock, error) {
	var bid [32]byte
	copy(bid[:], blockID.Bytes())

	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]*proto.EndorseBlock, 0)
	for k, it := range p.byKey {
		if k.blockID == bid {
			out = append(out, it.eb)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no endorsements found for block ID")
	}
	return out, nil
}

package endorsementpool

import (
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
	verifier   EndorseVerifier
}

func NewEndorsementPool() *EndorsementPool {
	return &EndorsementPool{
		countLimit: endorsementPoolLimit,
		byKey:      make(map[key]*heapItemEndorsement),
		verifier:   v,
	}
}

func (p *EndorsementPool) Add(e *proto.EndorseBlock) (bool, error) {
	if e == nil || len(e.EndorsedBlockId) == 0 {
		return false, errors.New("invalid endorsed block id")
	}
	if p.verifier != nil && !p.verifier.Verify(e) {
		return false, errors.New("failed to verify the endorsement")
	}
	k := makeKey(e.EndorsedBlockId, e.EndorserIndex)

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.byKey[k]; exists {
		return false, errors.New("the endorsement is a duplicate") // duplicate
	}
	if len(p.byKey) >= p.countLimit {
		return false, errors.New("the endorsement pool is full")
	}

	p.seq++
	p.byKey[k] = &heapItemEndorsement{eb: e, seq: p.seq}
	return true, nil
}

func (p *EndorsementPool) GetAll() []*proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*proto.EndorseBlock, 0, len(p.byKey))
	for _, it := range p.byKey {
		out = append(out, it.eb)
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
}

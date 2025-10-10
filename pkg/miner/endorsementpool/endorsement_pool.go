package endorsementpool

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type key struct {
	blockID       proto.BlockID
	endorserIndex int32
}

func makeKey(blockID proto.BlockID, idx int32) key {
	return key{blockID: blockID, endorserIndex: idx}
}

type heapItemEndorsement struct {
	eb         *proto.EndorseBlock
	endorserPK bls.PublicKey
	seq        uint64 // insertion sequence for FIFO priority
	index      int    // position in heap
}

type endorsementHeap []*heapItemEndorsement

func (h endorsementHeap) Len() int { return len(h) }

// Less — smaller seq = older endorsement = higher priority (floats to top).
func (h endorsementHeap) Less(i, j int) bool {
	return h[i].seq < h[j].seq
}

func (h endorsementHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *endorsementHeap) Push(x any) {
	it, ok := x.(*heapItemEndorsement)
	if !ok {
		panic(fmt.Sprintf("endorsementHeap.Push: unexpected type %T", x))
	}
	it.index = len(*h)
	*h = append(*h, it)
}

func (h *endorsementHeap) Pop() any {
	if h == nil || len(*h) == 0 {
		return nil
	}
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	(*h)[n-1] = nil // avoid memory leaks
	*h = old[:n-1]
	return item
}

type EndorseVerifier interface {
	Verify(eb *proto.EndorseBlock) bool
}

type EndorsementPool struct {
	mu                      sync.Mutex
	seq                     uint64
	countLimit              int
	byKey                   map[key]*heapItemEndorsement
	h                       endorsementHeap
	endorsersPublicKeyCache GeneratorsPublicKeysCache
}

func NewEndorsementPool(cache GeneratorsPublicKeysCache) *EndorsementPool {
	return &EndorsementPool{
		countLimit:              settings.EndorsementPoolLimit,
		byKey:                   make(map[key]*heapItemEndorsement),
		endorsersPublicKeyCache: cache,
	}
}

// Add inserts an endorsement into the pool with proper synchronization and consistency.
func (p *EndorsementPool) Add(e *proto.EndorseBlock) error {
	if e == nil {
		return errors.New("invalid endorsed block id")
	}
	k := makeKey(e.EndorsedBlockID, e.EndorserIndex)

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.byKey[k]; exists {
		return fmt.Errorf("duplicate endorsement: endorser index %d, block ID %s",
			e.EndorserIndex, e.EndorsedBlockID.String())
	}
	if p.countLimit > 0 && len(p.byKey) >= p.countLimit {
		return errors.New("the endorsement pool is full")
	}

	p.seq++
	endorserPublicKey := p.endorsersPublicKeyCache.PublicKeyByEndorserIndex(e.EndorserIndex)
	item := &heapItemEndorsement{
		eb:         e,
		seq:        p.seq,
		endorserPK: endorserPublicKey,
	}
	heap.Push(&p.h, item)
	p.byKey[k] = item
	return nil
}

// GetAll returns a copy of all endorsements currently in the pool.
func (p *EndorsementPool) GetAll() []proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]proto.EndorseBlock, 0, len(p.h))
	for _, endorsementItem := range p.h {
		if endorsementItem != nil {
			out = append(out, *endorsementItem.eb)
		}
	}
	return out
}

func (p *EndorsementPool) Finalize() proto.FinalizationVoting {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO: implement actual finalization aggregation.
	return proto.FinalizationVoting{}
}

func (p *EndorsementPool) GetEndorsers() []proto.WavesAddress {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.endorsersPublicKeyCache.AllEndorsers()
}

func (p *EndorsementPool) GetGenerators() []proto.WavesAddress {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.endorsersPublicKeyCache.AllGenerators()
}

func (p *EndorsementPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.byKey)
}

// CleanAll safely resets the pool.
func (p *EndorsementPool) CleanAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byKey = make(map[key]*heapItemEndorsement)
	p.h = endorsementHeap{}
	p.endorsersPublicKeyCache.CleanAllEndorsers()
}

// Pop removes the oldest endorsement (smallest seq) from the pool.
func (p *EndorsementPool) Pop() *proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.h) == 0 {
		return nil
	}
	v := heap.Pop(&p.h)
	it, ok := v.(*heapItemEndorsement)
	if !ok || it == nil {
		return nil
	}
	k := makeKey(it.eb.EndorsedBlockID, it.eb.EndorserIndex)
	delete(p.byKey, k)
	return it.eb
}

// Verify validates all endorsements in the pool by aggregating BLS signatures.
func (p *EndorsementPool) Verify() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := p.h.Len()
	if n == 0 {
		return false, errors.New("failed to verify endorsements, the endorsement pool is empty")
	}

	sigs := make([]bls.Signature, 0, n)
	pks := make([]bls.PublicKey, 0, n)

	for _, heapItem := range p.h {
		var sig bls.Signature
		if err := sig.UnmarshalJSON(heapItem.eb.Signature); err != nil {
			return false, fmt.Errorf("invalid signature at endorser index %d: %w",
				heapItem.eb.EndorserIndex, err)
		}
		sigs = append(sigs, sig)
		pks = append(pks, heapItem.endorserPK)
	}

	msg, err := p.h[0].eb.EndorsementMessage() // all endorsements use the same message
	if err != nil {
		return false, err
	}

	aggregatedSignature, err := bls.AggregateSignatures(sigs)
	if err != nil {
		return false, err
	}

	return bls.VerifyAggregate(pks, msg, aggregatedSignature), nil
}

type GeneratorsPublicKeysCache interface {
	PublicKeyByEndorserIndex(endorserIndex int32) bls.PublicKey
	AllGenerators() []proto.WavesAddress
	AllEndorsers() []proto.WavesAddress
	CleanAllEndorsers()
}

type GeneratorsPublicKeysCacheImpl struct{}

func (c *GeneratorsPublicKeysCacheImpl) PublicKeyByEndorserIndex(_ int32) bls.PublicKey {
	panic("not implemented")
}

func (c *GeneratorsPublicKeysCacheImpl) AllEndorsers() []proto.WavesAddress {
	panic("not implemented")
}

func (c *GeneratorsPublicKeysCacheImpl) CleanAllEndorsers() {
	panic("not implemented")
}

func (c *GeneratorsPublicKeysCacheImpl) AllGenerators() []proto.WavesAddress {
	panic("not implemented")
}

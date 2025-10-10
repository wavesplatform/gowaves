package endorsementpool

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const endorsementPoolLimit = 128

type key struct {
	blockID       proto.BlockID
	endorserIndex int32
}

func makeKey(blockID proto.BlockID, idx int32) key {
	var k key
	k.blockID = blockID
	k.endorserIndex = idx
	return k
}

type heapItemEndorsement struct {
	eb         *proto.EndorseBlock
	endorserPK bls.PublicKey
	seq        uint64 // insertion sequence for FIFO priority
	index      int    // position in heap
}

type endorsementHeap []*heapItemEndorsement

func (h endorsementHeap) Len() int { return len(h) }

// Less Oldest = smallest seq - floats to top.
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
	if n == 0 {
		return nil
	}
	item := old[n-1]
	item.index = -1
	old[n-1] = nil
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
	byKey                   map[key]heapItemEndorsement
	h                       endorsementHeap
	endorsersPublicKeyCache GeneratorsPublicKeysCache
}

func NewEndorsementPool(cache GeneratorsPublicKeysCache) *EndorsementPool {
	return &EndorsementPool{
		countLimit:              endorsementPoolLimit,
		byKey:                   make(map[key]heapItemEndorsement),
		endorsersPublicKeyCache: cache,
	}
}

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
	it := heapItemEndorsement{eb: e, seq: p.seq, endorserPK: endorserPublicKey}
	heap.Push(&p.h, it)
	p.byKey[k] = it
	return nil
}

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

	// TODO finalize.
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

func (p *EndorsementPool) CleanAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byKey = make(map[key]heapItemEndorsement)
	p.h = endorsementHeap{} // reset heap too
	p.endorsersPublicKeyCache.CleanAllEndorsers()
}

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

	msg, err := p.h[0].eb.EndorsementMessage() // all messages are assumed the same
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

type GeneratorsPublicKeysCacheImpl struct {
}

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

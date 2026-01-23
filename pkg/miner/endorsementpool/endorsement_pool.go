package endorsementpool

import (
	"bytes"
	"container/heap"
	"errors"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const EndorsementIDCacheSizeDefault = 1000

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
	balance    uint64
	seq        uint64
}

type endorsementMinHeap []*heapItemEndorsement

func (h endorsementMinHeap) Len() int { return len(h) }

func (h endorsementMinHeap) Less(i, j int) bool {
	if h[i].balance == h[j].balance {
		// Late (Higher seq), lower priority.
		return h[i].seq > h[j].seq
	}
	// Lower balance, lower priority.
	return h[i].balance < h[j].balance
}

func (h endorsementMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *endorsementMinHeap) Push(x any) {
	item, ok := x.(*heapItemEndorsement)
	if !ok {
		return // Impossible, but silences errcheck.
	}
	*h = append(*h, item)
}

func (h *endorsementMinHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type EndorsementPool struct {
	mu              sync.Mutex
	seq             uint64
	byKey           map[key]*heapItemEndorsement
	h               endorsementMinHeap
	conflicts       []proto.EndorseBlock
	maxEndorsements int
}

func NewEndorsementPool(maxGenerators int) (*EndorsementPool, error) {
	if maxGenerators <= 0 {
		return nil, errors.New("the max number of endorsements must be more than 0")
	}
	return &EndorsementPool{
		byKey:           make(map[key]*heapItemEndorsement),
		maxEndorsements: maxGenerators,
	}, nil
}

// Add inserts an endorsement into the heap with priority based on balance desc, seq asc.
func (p *EndorsementPool) Add(e *proto.EndorseBlock, pk bls.PublicKey,
	lastFinalizedHeight proto.Height, lastFinalizedBlockID proto.BlockID, balance uint64) error {
	if e == nil {
		return errors.New("invalid endorsement")
	}

	k := makeKey(e.EndorsedBlockID, e.EndorserIndex)

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.byKey[k]; exists {
		p.conflicts = append(p.conflicts, *e)
		return nil
	}
	if proto.Height(e.FinalizedBlockHeight) <= lastFinalizedHeight &&
		e.FinalizedBlockID != lastFinalizedBlockID {
		p.conflicts = append(p.conflicts, *e)
		return nil
	}

	p.seq++
	item := &heapItemEndorsement{
		eb:         e,
		endorserPK: pk,
		balance:    balance,
		seq:        p.seq,
	}

	// If heap is not filled yet.
	if len(p.h) < p.maxEndorsements {
		heap.Push(&p.h, item)
		p.byKey[k] = item
		return nil
	}

	// If heap is full — check min (root).
	minItem := p.h[0]
	// If priority is lower or equal the min, throw the new one away.
	if balance < minItem.balance || (balance == minItem.balance && item.seq > minItem.seq) {
		return nil
	}

	// Otherwise remove min and insert the new one.
	r := heap.Pop(&p.h)
	removed, _ := r.(*heapItemEndorsement)
	delete(p.byKey, makeKey(removed.eb.EndorsedBlockID, removed.eb.EndorserIndex))

	heap.Push(&p.h, item)
	p.byKey[k] = item
	return nil
}

func (p *EndorsementPool) GetAll() []proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]proto.EndorseBlock, len(p.h))
	for i, it := range p.h {
		out[i] = *it.eb
	}
	return out
}

func (p *EndorsementPool) FormFinalization(lastFinalizedHeight proto.Height) (proto.FinalizationVoting, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	signatures := make([]bls.Signature, 0, len(p.h))
	endorsersIndexes := make([]int32, 0, len(p.h))
	var aggregatedSignature bls.Signature

	for _, it := range p.h {
		signatures = append(signatures, it.eb.Signature)
		endorsersIndexes = append(endorsersIndexes, it.eb.EndorserIndex)
	}
	if len(signatures) != 0 {
		aggregatedSignatureBytes, err := bls.AggregateSignatures(signatures)
		if err != nil {
			return proto.FinalizationVoting{}, err
		}
		var errCnvrt error
		aggregatedSignature, errCnvrt = bls.NewSignatureFromBytes(aggregatedSignatureBytes)
		if errCnvrt != nil {
			return proto.FinalizationVoting{}, errCnvrt
		}
	}

	return proto.FinalizationVoting{
		AggregatedEndorsementSignature: aggregatedSignature,
		FinalizedBlockHeight:           lastFinalizedHeight,
		EndorserIndexes:                endorsersIndexes,
		ConflictEndorsements:           p.conflicts,
	}, nil
}

func (p *EndorsementPool) GetEndorsers() []bls.PublicKey {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]bls.PublicKey, len(p.h))
	for i, it := range p.h {
		out[i] = it.endorserPK
	}
	return out
}

func (p *EndorsementPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.h)
}

func (p *EndorsementPool) CleanAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.byKey = make(map[key]*heapItemEndorsement)
	p.h = nil
	p.conflicts = nil
}

func (p *EndorsementPool) Verify() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.h)
	if n == 0 {
		return false, errors.New("failed to verify endorsements: pool is empty")
	}

	sigs := make([]bls.Signature, 0, n)
	pks := make([]bls.PublicKey, 0, n)
	msg, err := p.h[0].eb.EndorsementMessage()
	if err != nil {
		return false, err
	}
	for _, it := range p.h {
		sigs = append(sigs, it.eb.Signature)
		pks = append(pks, it.endorserPK)
		nextMsg, msgErr := it.eb.EndorsementMessage()
		if msgErr != nil {
			return false, msgErr
		}
		if !bytes.Equal(nextMsg, msg) {
			return false, errors.New("failed to verify endorsements: inconsistent endorsement messages")
		}
	}
	agg, err := bls.AggregateSignatures(sigs)
	if err != nil {
		return false, err
	}
	return bls.VerifyAggregate(pks, msg, agg), nil
}

func (p *EndorsementPool) ConflictEndorsements() []proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]proto.EndorseBlock, len(p.conflicts))
	copy(out, p.conflicts)
	return out
}

type EndorsementIDsCache struct {
	ids   map[crypto.Digest]struct{}
	order []crypto.Digest
	limit int
}

func NewEndorsementIDsCache(cacheLimit int) *EndorsementIDsCache {
	return &EndorsementIDsCache{
		ids:   make(map[crypto.Digest]struct{}),
		limit: cacheLimit,
	}
}

func (cache *EndorsementIDsCache) SeenEndorsement(id crypto.Digest) bool {
	_, ok := cache.ids[id]
	return ok
}

func (cache *EndorsementIDsCache) RememberEndorsement(id crypto.Digest) {
	if cache.ids == nil {
		cache.ids = make(map[crypto.Digest]struct{})
	}
	if cache.limit <= 0 {
		return
	}
	if _, exists := cache.ids[id]; exists {
		return
	}
	if len(cache.ids) >= cache.limit && len(cache.order) > 0 {
		// Evict oldest.
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.ids, oldest)
	}
	cache.ids[id] = struct{}{}
	cache.order = append(cache.order, id)
}

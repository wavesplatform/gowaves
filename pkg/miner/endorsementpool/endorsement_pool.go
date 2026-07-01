package endorsementpool

import (
	"container/heap"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const EndorsementIDCacheSizeDefault = 1000

type heapItem struct {
	eb         *proto.BlockEndorsement
	endorserPK bls.PublicKey
	balance    uint64
	seq        uint64
}

type endorsementMinHeap []*heapItem

func (h endorsementMinHeap) Len() int { return len(h) }

func (h endorsementMinHeap) Less(i, j int) bool {
	if h[i].balance == h[j].balance {
		return h[i].seq > h[j].seq // higher seq = lower priority (earlier arrival wins)
	}
	return h[i].balance < h[j].balance // lower balance = lower priority
}

func (h endorsementMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *endorsementMinHeap) Push(x any) {
	item, ok := x.(*heapItem)
	if !ok {
		return // impossible, but satisfies errcheck
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

// EndorsementPool holds valid endorsements (bounded priority queue by endorser balance) and
// conflicting endorsements (append-only, one per generator) for a single key-block round.
//
// The pool is reset via Reset() on each new key-block to start a fresh round.
//
// Change tracking is split into two independent signals:
//   - snapshot: the full current set of good endorsements changed (add, evict, or conflict removal)
//   - conflicts: new conflicting endorsements arrived since the last FormFinalization call
//
// HasUpdate() returns true when either signal is set, indicating a new FinalizationVoting should
// be produced.
type EndorsementPool struct {
	mu          sync.Mutex
	seq         uint64
	byIndex     map[uint32]*heapItem
	h           endorsementMinHeap
	conflicts   []proto.BlockEndorsement
	conflictSet map[uint32]struct{} // Indexes of generators that produced conflicting endorsements.
	maxSize     int

	// Change tracking — two-phase commit:
	//   FormFinalization records a "pending" snapshot of the current state and returns a voting.
	//   CommitFinalization advances the "committed" watermarks to the pending snapshot.
	//   HasUpdate compares current state against the committed watermarks.
	//
	// This ensures that if the microblock carrying the voting is never applied (e.g. ErrStateChanged),
	// the watermarks remain at the last committed position and HasUpdate keeps returning true until
	// the data is successfully published.
	snapshotVersion    uint64 // Incremented on every snapshot change.
	lastCommitVersion  uint64 // snapshotVersion at last CommitFinalization call.
	committedWatermark int    // len(conflicts) at last CommitFinalization call.
	pendingVersion     uint64 // snapshotVersion saved by the last FormFinalization call.
	pendingWatermark   int    // len(conflicts) saved by the last FormFinalization call.
}

func NewEndorsementPool(maxSize int) (*EndorsementPool, error) {
	if maxSize <= 0 {
		return nil, errors.New("max pool size must be positive")
	}
	return &EndorsementPool{
		byIndex:     make(map[uint32]*heapItem),
		conflictSet: make(map[uint32]struct{}),
		maxSize:     maxSize,
	}, nil
}

// Reset re-initializes the pool for a new key-block round, discarding all state.
func (p *EndorsementPool) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seq = 0
	p.byIndex = make(map[uint32]*heapItem)
	p.h = nil
	p.conflicts = nil
	p.conflictSet = make(map[uint32]struct{})
	p.snapshotVersion = 0
	p.lastCommitVersion = 0
	p.committedWatermark = 0
	p.pendingVersion = 0
	p.pendingWatermark = 0
	slog.Debug("Endorsement pool reset for new round")
}

// Add inserts a valid endorsement into the pool.
// Returns true if the valid endorsement pool changed.
// Silently rejects (returns false, nil) when:
//   - the generator already has a conflicting endorsement recorded
//   - the generator already has a valid endorsement (duplicate)
//   - the pool is full and the new balance is not strictly greater than the current minimum
func (p *EndorsementPool) Add(e *proto.BlockEndorsement, pk bls.PublicKey, balance uint64) (bool, error) {
	if e == nil {
		return false, errors.New("nil endorsement")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := e.EndorserIndex
	if _, conflicted := p.conflictSet[idx]; conflicted {
		slog.Debug("Endorsement rejected: generator already has a conflicting endorsement", "index", idx)
		return false, nil
	}
	if _, exists := p.byIndex[idx]; exists {
		slog.Debug("Endorsement rejected: duplicate for generator", "index", idx)
		return false, nil
	}

	p.seq++
	item := &heapItem{eb: e, endorserPK: pk, balance: balance, seq: p.seq}

	if len(p.h) < p.maxSize {
		heap.Push(&p.h, item)
		p.byIndex[idx] = item
		p.snapshotVersion++
		return true, nil
	}

	minItem := p.h[0]
	if balance <= minItem.balance {
		// Equal or less balance: earlier arrival (lower seq) stays in pool.
		return false, nil
	}

	removed, ok := heap.Pop(&p.h).(*heapItem)
	if !ok {
		return false, errors.New("internal error: heap contained unexpected type")
	}
	delete(p.byIndex, removed.eb.EndorserIndex)
	heap.Push(&p.h, item)
	p.byIndex[idx] = item
	p.snapshotVersion++
	slog.Debug("Evicted lower-balance endorsement from full pool",
		"evictedIndex", removed.eb.EndorserIndex, "evictedBalance", removed.balance,
		"newIndex", idx, "newBalance", balance)
	return true, nil
}

// AddConflict records a conflicting endorsement for a generator.
// Returns true if this is a new conflict (first time for this generator index).
// Returns false for any subsequent conflicting endorsement from the same generator (silently dropped).
// If the generator had a valid endorsement in the pool, it is removed and the snapshot is updated.
func (p *EndorsementPool) AddConflict(e *proto.BlockEndorsement) bool {
	if e == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := e.EndorserIndex
	if _, already := p.conflictSet[idx]; already {
		slog.Debug("Conflicting endorsement ignored: duplicate for generator", "index", idx)
		return false
	}

	p.conflictSet[idx] = struct{}{}
	p.conflicts = append(p.conflicts, *e)

	if item, inPool := p.byIndex[idx]; inPool {
		p.removeFromHeap(item)
		delete(p.byIndex, idx)
		p.snapshotVersion++
		slog.Debug("Removed conflicted generator from valid endorsements pool", "index", idx)
	}

	return true
}

func (p *EndorsementPool) removeFromHeap(target *heapItem) {
	for i, it := range p.h {
		if it == target {
			heap.Remove(&p.h, i)
			return
		}
	}
}

// HasUpdate reports whether there are pending changes since the last FormFinalization call.
// HasUpdate reports pending changes since the last CommitFinalization call.
// Returns true when:
//   - the valid endorsements pool changed since the last commit
//   - new conflicting endorsements were added since the last commit
func (p *EndorsementPool) HasUpdate() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshotVersion != p.lastCommitVersion || p.committedWatermark < len(p.conflicts)
}

// FormFinalization produces a FinalizationVoting from the current pool state and saves a pending
// watermark snapshot, but does NOT advance the committed watermarks.
//
// The caller must invoke CommitFinalization after the microblock carrying this voting is
// successfully applied. If the microblock is never applied, HasUpdate continues returning true
// and the next FormFinalization call re-produces the same data.
//
// The snapshot (good endorsements) is always the full current set.
// The conflicts section contains only entries added since the previous CommitFinalization call.
func (p *EndorsementPool) FormFinalization() (proto.FinalizationVoting, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.h) == 0 && len(p.conflicts) == 0 {
		return proto.FinalizationVoting{}, errors.New("pool is empty: no endorsements to form finalization")
	}

	var (
		aggregatedSig   *bls.Signature
		endorserIndexes []uint32
		finalizedHeight proto.Height
	)

	if len(p.h) > 0 {
		sigs := make([]bls.Signature, len(p.h))
		endorserIndexes = make([]uint32, len(p.h))
		for i, it := range p.h {
			sigs[i] = it.eb.Signature
			endorserIndexes[i] = it.eb.EndorserIndex
		}
		agg, err := bls.AggregateSignatures(sigs)
		if err != nil {
			return proto.FinalizationVoting{}, fmt.Errorf("failed to aggregate endorsement signatures: %w", err)
		}
		aggregatedSig = &agg
		finalizedHeight = proto.Height(p.h[0].eb.FinalizedBlockHeight)
	}

	newConflicts := slices.Clone(p.conflicts[p.committedWatermark:])

	// Save pending state; committed watermarks are not advanced until CommitFinalization is called.
	p.pendingVersion = p.snapshotVersion
	p.pendingWatermark = len(p.conflicts)

	return proto.FinalizationVoting{
		AggregatedEndorsementSignature: aggregatedSig,
		FinalizedBlockHeight:           finalizedHeight,
		EndorserIndexes:                endorserIndexes,
		ConflictEndorsements:           newConflicts,
	}, nil
}

// CommitFinalization advances the committed watermarks to the pending state saved by the last
// FormFinalization call. Must be called after the microblock carrying the voting is successfully
// applied to the blockchain. Calling it without a prior FormFinalization is a no-op.
func (p *EndorsementPool) CommitFinalization() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastCommitVersion = p.pendingVersion
	p.committedWatermark = p.pendingWatermark
}

// Len returns the number of good endorsements currently in the pool.
func (p *EndorsementPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.h)
}

// GetAll returns all good endorsements currently in the pool. For testing and inspection.
func (p *EndorsementPool) GetAll() []proto.BlockEndorsement {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]proto.BlockEndorsement, len(p.h))
	for i, it := range p.h {
		out[i] = *it.eb
	}
	return out
}

// ConflictEndorsements returns all recorded conflicting endorsements. For testing and inspection.
func (p *EndorsementPool) ConflictEndorsements() []proto.BlockEndorsement {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]proto.BlockEndorsement, len(p.conflicts))
	copy(out, p.conflicts)
	return out
}

// EndorsementIDsCache is an LRU-style seen-endorsement tracker to suppress re-processing.
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
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.ids, oldest)
	}
	cache.ids[id] = struct{}{}
	cache.order = append(cache.order, id)
}

func (cache *EndorsementIDsCache) Clear() {
	cache.ids = make(map[crypto.Digest]struct{})
	cache.order = nil
}

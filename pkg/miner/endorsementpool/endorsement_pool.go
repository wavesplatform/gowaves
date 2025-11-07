package endorsementpool

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	balance    uint64
	seq        uint64
}
type EndorsementPool struct {
	mu        sync.Mutex
	seq       uint64
	byKey     map[key]*heapItemEndorsement
	items     []*heapItemEndorsement // always sorted
	conflicts []proto.EndorseBlock
}

func NewEndorsementPool() *EndorsementPool {
	return &EndorsementPool{
		byKey: make(map[key]*heapItemEndorsement),
	}
}

// Add inserts an endorsement keeping the pool sorted by balance desc, seq asc.
func (p *EndorsementPool) Add(e *proto.EndorseBlock, pk bls.PublicKey, balance uint64) error {
	if e == nil {
		return errors.New("invalid endorsement")
	}
	k := makeKey(e.EndorsedBlockID, e.EndorserIndex)

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.byKey[k]; exists {
		p.conflicts = append(p.conflicts, *e)
		return fmt.Errorf("duplicate endorsement: endorser %d, block %s",
			e.EndorserIndex, e.EndorsedBlockID.String())
	}

	p.seq++
	item := &heapItemEndorsement{
		eb:         e,
		endorserPK: pk,
		balance:    balance,
		seq:        p.seq,
	}
	p.insertSorted(item)
	p.byKey[k] = item
	return nil
}

func (p *EndorsementPool) insertSorted(item *heapItemEndorsement) {
	i := sort.Search(len(p.items), func(i int) bool {
		if p.items[i].balance != item.balance {
			// descending balance: insert before smaller balances
			return p.items[i].balance < item.balance
		}
		// ascending seq: insert before newer (larger seq)
		return p.items[i].seq > item.seq
	})
	p.items = append(p.items, nil)
	copy(p.items[i+1:], p.items[i:])
	p.items[i] = item
}

// GetAll returns a copy of all endorsements currently in the pool.
func (p *EndorsementPool) GetAll() []proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]proto.EndorseBlock, 0, len(p.items))
	for _, it := range p.items {
		out = append(out, *it.eb)
	}
	return out
}

func (p *EndorsementPool) Finalize() (proto.FinalizationVoting, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var (
		signatures       = make([]bls.Signature, 0, len(p.items))
		endorsersIndexes = make([]int32, 0, len(p.items))
	)

	for _, it := range p.items {
		endorseBlock := it.eb
		sig, err := bls.NewSignatureFromBytes(endorseBlock.Signature)
		if err != nil {
			// TODO punish generator for bad signature
			return proto.FinalizationVoting{}, err
		}
		signatures = append(signatures, sig)
		endorsersIndexes = append(endorsersIndexes, endorseBlock.EndorserIndex)
	}
	aggregatedSignature, err := bls.AggregateSignatures(signatures)
	if err != nil {
		return proto.FinalizationVoting{}, err
	}
	return proto.FinalizationVoting{
		AggregatedEndorsementSignature: aggregatedSignature,
		EndorserIndexes:                endorsersIndexes,
		ConflictEndorsements:           p.conflicts,
	}, nil
}

func (p *EndorsementPool) GetEndorsers() []bls.PublicKey {
	p.mu.Lock()
	defer p.mu.Unlock()

	endorsers := make([]bls.PublicKey, 0, len(p.items))
	for _, it := range p.items {
		if it != nil {
			endorsers = append(endorsers, it.endorserPK)
		}
	}
	return endorsers
}

func (p *EndorsementPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

// CleanAll safely resets the pool.
func (p *EndorsementPool) CleanAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byKey = make(map[key]*heapItemEndorsement)
	p.items = nil
	p.conflicts = nil
}

// Verify validates all endorsements in the pool by aggregating BLS signatures.
// Verify validates all endorsements in the pool by aggregating BLS signatures.
func (p *EndorsementPool) Verify() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.items)
	if n == 0 {
		return false, errors.New("failed to verify endorsements: the endorsement pool is empty")
	}

	sigs := make([]bls.Signature, 0, n)
	pks := make([]bls.PublicKey, 0, n)

	for _, it := range p.items {
		var sig bls.Signature
		if err := sig.UnmarshalJSON(it.eb.Signature); err != nil {
			return false, fmt.Errorf("invalid signature at endorser index %d: %w",
				it.eb.EndorserIndex, err)
		}
		sigs = append(sigs, sig)
		pks = append(pks, it.endorserPK)
	}

	msg, err := p.items[0].eb.EndorsementMessage() // all endorsements use the same message
	if err != nil {
		return false, err
	}

	agg, err := bls.AggregateSignatures(sigs)
	if err != nil {
		return false, err
	}
	return bls.VerifyAggregate(pks, msg, agg), nil
}

// ConflictEndorsements returns all endorsements that were detected as duplicates.
func (p *EndorsementPool) ConflictEndorsements() []proto.EndorseBlock {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]proto.EndorseBlock, len(p.conflicts))
	copy(out, p.conflicts)
	return out
}

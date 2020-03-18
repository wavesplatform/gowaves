package ordered_blocks

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OrderedBlocks struct {
	sigSequence    []crypto.Signature
	uniqSignatures map[crypto.Signature]*proto.Block
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		sigSequence:    nil,
		uniqSignatures: make(map[crypto.Signature]*proto.Block),
	}
}

func (a *OrderedBlocks) Contains(sig crypto.Signature) bool {
	_, ok := a.uniqSignatures[sig]
	return ok
}

func (a *OrderedBlocks) SetBlock(b *proto.Block) {
	a.uniqSignatures[b.BlockSignature] = b
}

func (a *OrderedBlocks) pop() (crypto.Signature, *proto.Block, bool) {
	if len(a.sigSequence) == 0 {
		return crypto.Signature{}, nil, false
	}
	firstSig := a.sigSequence[0]
	bts := a.uniqSignatures[firstSig]
	if bts != nil {
		delete(a.uniqSignatures, firstSig)
		a.sigSequence = a.sigSequence[1:]
		return firstSig, bts, true
	}
	return crypto.Signature{}, nil, false
}

func (a *OrderedBlocks) PopAll() []*proto.Block {
	var out []*proto.Block
	for {
		_, b, ok := a.pop()
		if !ok {
			return out
		}
		out = append(out, b)
	}
}

// true - added, false - not added
func (a *OrderedBlocks) Add(sig crypto.Signature) bool {
	// already contains
	if _, ok := a.uniqSignatures[sig]; ok {
		return false
	}
	a.sigSequence = append(a.sigSequence, sig)
	a.uniqSignatures[sig] = nil
	return true
}

func (a *OrderedBlocks) WaitingCount() int {
	return len(a.sigSequence)
}

// blocks count available for pop
func (a *OrderedBlocks) AvailableCount() int {
	for i, sig := range a.sigSequence {
		if a.uniqSignatures[sig] == nil {
			return i
		}
	}
	return len(a.sigSequence)
}

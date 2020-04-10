package ordered_blocks

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OrderedBlocks struct {
	sigSequence  []proto.BlockID
	uniqBlockIDs map[proto.BlockID]*proto.Block
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		sigSequence:  nil,
		uniqBlockIDs: make(map[proto.BlockID]*proto.Block),
	}
}

func (a *OrderedBlocks) Contains(sig proto.BlockID) bool {
	_, ok := a.uniqBlockIDs[sig]
	return ok
}

func (a *OrderedBlocks) SetBlock(b *proto.Block) {
	a.uniqBlockIDs[b.BlockID()] = b
}

func (a *OrderedBlocks) pop() (proto.BlockID, *proto.Block, bool) {
	if len(a.sigSequence) == 0 {
		return proto.BlockID{}, nil, false
	}
	firstSig := a.sigSequence[0]
	bts := a.uniqBlockIDs[firstSig]
	if bts != nil {
		delete(a.uniqBlockIDs, firstSig)
		a.sigSequence = a.sigSequence[1:]
		return firstSig, bts, true
	}
	return proto.BlockID{}, nil, false
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
func (a *OrderedBlocks) Add(sig proto.BlockID) bool {
	// already contains
	if _, ok := a.uniqBlockIDs[sig]; ok {
		return false
	}
	a.sigSequence = append(a.sigSequence, sig)
	a.uniqBlockIDs[sig] = nil
	return true
}

func (a *OrderedBlocks) WaitingCount() int {
	return len(a.sigSequence)
}

// blocks count available for pop
func (a *OrderedBlocks) AvailableCount() int {
	for i, sig := range a.sigSequence {
		if a.uniqBlockIDs[sig] == nil {
			return i
		}
	}
	return len(a.sigSequence)
}

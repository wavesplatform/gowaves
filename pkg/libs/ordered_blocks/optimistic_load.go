package ordered_blocks

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OrderedBlocks struct {
	requested []proto.BlockID
	blocks    map[proto.BlockID]*proto.Block
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		requested: nil,
		blocks:    make(map[proto.BlockID]*proto.Block),
	}
}

func (a *OrderedBlocks) Contains(sig proto.BlockID) bool {
	_, ok := a.blocks[sig]
	return ok
}

func (a *OrderedBlocks) SetBlock(b *proto.Block) {
	a.blocks[b.BlockID()] = b
}

func (a *OrderedBlocks) pop() (proto.BlockID, *proto.Block, bool) {
	if len(a.requested) == 0 {
		return proto.BlockID{}, nil, false
	}
	firstSig := a.requested[0]
	bts := a.blocks[firstSig]
	if bts != nil {
		delete(a.blocks, firstSig)
		a.requested = a.requested[1:]
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
	if _, ok := a.blocks[sig]; ok {
		return false
	}
	a.requested = append(a.requested, sig)
	a.blocks[sig] = nil
	return true
}

func (a *OrderedBlocks) RequestedCount() int {
	return len(a.requested)
}

// blocks count available for pop
func (a *OrderedBlocks) ReceivedCount() int {
	for i, sig := range a.requested {
		if a.blocks[sig] == nil {
			return i
		}
	}
	return len(a.requested)
}

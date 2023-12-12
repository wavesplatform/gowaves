package ordered_blocks

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OrderedBlocks struct {
	requestedBlocks []proto.BlockID
	blocks          map[proto.BlockID]*proto.Block
	snapshots       map[proto.BlockID]*proto.BlockSnapshot
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		requestedBlocks: nil,
		blocks:          make(map[proto.BlockID]*proto.Block),
		snapshots:       make(map[proto.BlockID]*proto.BlockSnapshot),
	}
}

func (a *OrderedBlocks) Contains(sig proto.BlockID) bool {
	_, ok := a.blocks[sig]
	return ok
}

func (a *OrderedBlocks) SetBlock(b *proto.Block) {
	a.blocks[b.BlockID()] = b
}

func (a *OrderedBlocks) SetSnapshot(blockID proto.BlockID, snapshot *proto.BlockSnapshot) {
	a.snapshots[blockID] = snapshot
}

func (a *OrderedBlocks) pop() (proto.BlockID, *proto.Block, *proto.BlockSnapshot, bool) {
	if len(a.requestedBlocks) == 0 {
		return proto.BlockID{}, nil, nil, false
	}
	firstSig := a.requestedBlocks[0]
	bts := a.blocks[firstSig]
	bsn := a.snapshots[firstSig]
	if bts != nil && bsn != nil {
		delete(a.blocks, firstSig)
		delete(a.snapshots, firstSig)
		a.requestedBlocks = a.requestedBlocks[1:]
		return firstSig, bts, bsn, true
	}
	return proto.BlockID{}, nil, nil, false
}

func (a *OrderedBlocks) PopAll() ([]*proto.Block, []*proto.BlockSnapshot) {
	var outBlocks []*proto.Block
	var outSnapshots []*proto.BlockSnapshot
	for {
		_, b, s, ok := a.pop()
		if !ok {
			return outBlocks, outSnapshots
		}
		outBlocks = append(outBlocks, b)
		outSnapshots = append(outSnapshots, s)
	}
}

// true - added, false - not added
func (a *OrderedBlocks) Add(sig proto.BlockID) bool {
	// already contains
	if _, ok := a.blocks[sig]; ok {
		return false
	}
	a.requestedBlocks = append(a.requestedBlocks, sig)
	a.blocks[sig] = nil
	a.snapshots[sig] = nil
	return true
}

func (a *OrderedBlocks) RequestedCount() int {
	return len(a.requestedBlocks)
}

// blocks count available for pop
func (a *OrderedBlocks) ReceivedCount(isLightNode bool) int {
	for i, sig := range a.requestedBlocks {
		if isLightNode && a.blocks[sig] == nil || a.snapshots[sig] == nil {
			return i
		} else if !isLightNode && a.blocks[sig] == nil {
			return i
		}
	}
	return len(a.requestedBlocks)
}

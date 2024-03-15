package ordered_blocks

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OrderedBlocks struct {
	requested []proto.BlockID
	blocks    map[proto.BlockID]*proto.Block
	snapshots map[proto.BlockID]*proto.BlockSnapshot
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		requested: nil,
		blocks:    make(map[proto.BlockID]*proto.Block),
		snapshots: make(map[proto.BlockID]*proto.BlockSnapshot),
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

func (a *OrderedBlocks) pop(isLightNode bool) (proto.BlockID, *proto.Block, *proto.BlockSnapshot, bool) {
	if len(a.requested) == 0 {
		return proto.BlockID{}, nil, nil, false
	}
	firstSig := a.requested[0]
	bts := a.blocks[firstSig]
	bsn := a.snapshots[firstSig]
	if bts != nil {
		delete(a.blocks, firstSig)
		if isLightNode && bsn != nil {
			delete(a.snapshots, firstSig)
			a.requested = a.requested[1:]
			return firstSig, bts, bsn, true
		}
		a.requested = a.requested[1:]
		return firstSig, bts, nil, true
	}
	return proto.BlockID{}, nil, nil, false
}

func (a *OrderedBlocks) PopAll(isLightNode bool) ([]*proto.Block, []*proto.BlockSnapshot) {
	var outBlocks []*proto.Block
	var outSnapshots []*proto.BlockSnapshot
	for {
		_, b, s, ok := a.pop(isLightNode)
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
	a.requested = append(a.requested, sig)
	a.blocks[sig] = nil
	a.snapshots[sig] = nil
	return true
}

func (a *OrderedBlocks) RequestedCount() int {
	return len(a.requested)
}

// blocks count available for pop
func (a *OrderedBlocks) ReceivedCount(isLightNode bool) int {
	for i, sig := range a.requested {
		blockIsNil := a.blocks[sig] == nil
		if isLightNode && (blockIsNil || a.snapshots[sig] == nil) {
			return i
		} else if !isLightNode && blockIsNil {
			return i
		}
	}
	return len(a.requested)
}

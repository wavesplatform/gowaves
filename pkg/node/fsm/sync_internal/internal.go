package sync_internal

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Blocks []*proto.Block
type Snapshots []*proto.BlockSnapshot
type Eof = bool
type BlockApplied bool

var NoSignaturesExpectedErr = proto.NewInfoMsg(errors.New("no signatures expected"))
var UnexpectedBlockErr = proto.NewInfoMsg(errors.New("unexpected block"))

type PeerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
	AskBlock(id proto.BlockID)
	AskBlockSnapshot(id proto.BlockID)
}

type Internal struct {
	respondedSignatures  *signatures.BlockIDs
	orderedBlocks        *ordered_blocks.OrderedBlocks
	waitingForSignatures bool
	isLightNode          bool
}

func InternalFromLastSignatures(
	p extension.PeerExtension,
	signatures *signatures.ReverseOrdering,
	isLightNode bool,
) Internal {
	p.AskBlocksIDs(signatures.BlockIDS())
	return NewInternal(ordered_blocks.NewOrderedBlocks(), signatures, true, isLightNode)
}

func NewInternal(
	orderedBlocks *ordered_blocks.OrderedBlocks,
	respondedSignatures *signatures.ReverseOrdering,
	waitingForSignatures bool,
	isLightNode bool,
) Internal {
	return Internal{
		respondedSignatures:  respondedSignatures,
		orderedBlocks:        orderedBlocks,
		waitingForSignatures: waitingForSignatures,
		isLightNode:          isLightNode,
	}
}

func (a Internal) BlockIDs(p PeerExtension, ids []proto.BlockID) (Internal, error) {
	if !a.waitingForSignatures {
		return a, NoSignaturesExpectedErr
	}
	var newIDs []proto.BlockID
	for _, id := range ids {
		if a.respondedSignatures.Exists(id) {
			continue
		}
		newIDs = append(newIDs, id)
		if a.orderedBlocks.Add(id) {
			p.AskBlock(id)
			if a.isLightNode {
				p.AskBlockSnapshot(id)
			}
		}
	}
	respondedSignatures := signatures.NewSignatures(newIDs...).Revert()
	return NewInternal(a.orderedBlocks, respondedSignatures, false, a.isLightNode), nil
}

func (a Internal) WaitingForSignatures() bool {
	return a.waitingForSignatures
}

func (a Internal) Block(block *proto.Block) (Internal, error) {
	if !a.orderedBlocks.Contains(block.BlockID()) {
		return a, UnexpectedBlockErr
	}
	a.orderedBlocks.SetBlock(block)
	return a, nil
}

func (a Internal) SetSnapshot(blockID proto.BlockID, snapshot *proto.BlockSnapshot) (Internal, error) {
	if !a.orderedBlocks.Contains(blockID) {
		return a, UnexpectedBlockErr
	}
	a.orderedBlocks.SetSnapshot(blockID, snapshot)
	return a, nil
}

type peerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
}

func (a Internal) Blocks() (Internal, Blocks, Snapshots, Eof) {
	if a.waitingForSignatures {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.isLightNode), nil, nil, false
	}
	if a.orderedBlocks.RequestedCount() > a.orderedBlocks.ReceivedCount(a.isLightNode) {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.isLightNode), nil, nil, false
	}
	if a.orderedBlocks.RequestedCount() < 100 {
		bs, ss := a.orderedBlocks.PopAll(a.isLightNode)
		return NewInternal(a.orderedBlocks, a.respondedSignatures, false, a.isLightNode), bs, ss, true
	}
	bs, ss := a.orderedBlocks.PopAll(a.isLightNode)
	return NewInternal(a.orderedBlocks, a.respondedSignatures, true, a.isLightNode), bs, ss, false
}

func (a Internal) AskBlocksIDs(p peerExtension) {
	p.AskBlocksIDs(a.respondedSignatures.BlockIDS())
}

func (a Internal) AvailableCount() int {
	return a.orderedBlocks.ReceivedCount(a.isLightNode)
}

func (a Internal) RequestedCount() int {
	return a.orderedBlocks.RequestedCount()
}

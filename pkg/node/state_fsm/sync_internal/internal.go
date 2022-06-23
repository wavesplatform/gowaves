package sync_internal

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Blocks []*proto.Block
type Eof = bool
type ChangePeerNeeded = bool
type BlockApplied bool

var NoSignaturesExpectedErr = proto.NewInfoMsg(errors.New("no signatures expected"))
var UnexpectedBlockErr = proto.NewInfoMsg(errors.New("unexpected block"))

type PeerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
	AskBlock(id proto.BlockID)
}

type Internal struct {
	respondedSignatures  *signatures.BlockIDs
	orderedBlocks        *ordered_blocks.OrderedBlocks
	waitingForSignatures bool
}

func InternalFromLastSignatures(p extension.PeerExtension, signatures *signatures.ReverseOrdering) Internal {
	p.AskBlocksIDs(signatures.BlockIDS())
	return NewInternal(ordered_blocks.NewOrderedBlocks(), signatures, true)
}

func NewInternal(orderedBlocks *ordered_blocks.OrderedBlocks, respondedSignatures *signatures.ReverseOrdering, waitingForSignatures bool) Internal {
	return Internal{
		respondedSignatures:  respondedSignatures,
		orderedBlocks:        orderedBlocks,
		waitingForSignatures: waitingForSignatures,
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
		}
	}
	respondedSignatures := signatures.NewSignatures(newIDs...).Revert()
	return NewInternal(a.orderedBlocks, respondedSignatures, false), nil
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

type peerExtension interface {
	AskBlocksIDs(id []proto.BlockID)
}

func (a Internal) Blocks(p peerExtension, needToChangePeerSyncWithFunc func() bool) (Internal, Blocks, Eof, ChangePeerNeeded) {
	if a.waitingForSignatures {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures), nil, false, false
	}
	if a.orderedBlocks.RequestedCount() > a.orderedBlocks.ReceivedCount() {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures), nil, false, false
	}
	if a.orderedBlocks.RequestedCount() < 100 {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, false), a.orderedBlocks.PopAll(), true, false
	}
	if needToChangePeerSyncWithFunc != nil && needToChangePeerSyncWithFunc() {
		return a, nil, false, true
	}

	p.AskBlocksIDs(a.respondedSignatures.BlockIDS())
	return NewInternal(a.orderedBlocks, a.respondedSignatures, true), a.orderedBlocks.PopAll(), false, false
}

func (a Internal) AvailableCount() int {
	return a.orderedBlocks.ReceivedCount()
}

func (a Internal) RequestedCount() int {
	return a.orderedBlocks.RequestedCount()
}

package sync_internal

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Blocks []*proto.Block
type Eof = bool
type BlockApplied bool

const NoSignaturesExpected = false
const WaitingForSignatures = true

var NoSignaturesExpectedErr = errors.New("no signatures expected")
var UnexpectedBlockErr = errors.New("unexpected block")

type Internal struct {
	respondedSignatures  *signatures.BlockIDs
	orderedBlocks        *ordered_blocks.OrderedBlocks
	waitingForSignatures bool
	nearEnd              bool
}

func InternalFromLastSignatures(p PeerWrapper, sigs *signatures.ReverseOrdering) Internal {
	p.AskBlocksIDs(sigs.BlockIDS())
	return NewInternal(
		ordered_blocks.NewOrderedBlocks(),
		sigs,
		WaitingForSignatures,
		false)
}

func NewInternal(orderedBlocks *ordered_blocks.OrderedBlocks, respondedSignatures *signatures.ReverseOrdering, waitingForSignatures bool, nearEnd bool) Internal {
	return Internal{
		respondedSignatures:  respondedSignatures,
		orderedBlocks:        orderedBlocks,
		waitingForSignatures: waitingForSignatures,
		nearEnd:              nearEnd,
	}
}

func (a Internal) BlockIDs(p PeerWrapper, sigs []proto.BlockID) (Internal, error) {
	if !a.waitingForSignatures {
		return a, NoSignaturesExpectedErr
	}
	var newSigs []proto.BlockID
	for _, blockID := range sigs {
		if a.respondedSignatures.Exists(blockID) {
			continue
		}
		newSigs = append(newSigs, blockID)
		if a.orderedBlocks.Add(blockID) {
			p.AskBlock(blockID)
		}
	}
	respondedSignatures := signatures.NewSignatures(newSigs...).Revert()
	return NewInternal(a.orderedBlocks, respondedSignatures, NoSignaturesExpected, respondedSignatures.Len() < 100), nil
}

func (a Internal) NearEnd() bool {
	return a.nearEnd
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

func (a Internal) Blocks(p PeerWrapper) (Internal, Blocks, Eof) {
	if a.nearEnd {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, NoSignaturesExpected, a.nearEnd),
			a.orderedBlocks.PopAll(),
			a.orderedBlocks.WaitingCount() == 0
	}
	var blocks []*proto.Block
	if a.orderedBlocks.AvailableCount() >= 50 {
		blocks = a.orderedBlocks.PopAll()
	}
	if a.waitingForSignatures {
		return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.nearEnd), blocks, false
	}
	if a.orderedBlocks.WaitingCount() < 100 {
		p.AskBlocksIDs(a.respondedSignatures.BlockIDS())
		return NewInternal(a.orderedBlocks, a.respondedSignatures, WaitingForSignatures, a.nearEnd), blocks, false
	}
	return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.nearEnd), blocks, false
}

func (a Internal) AvailableCount() int {
	return a.orderedBlocks.AvailableCount()
}

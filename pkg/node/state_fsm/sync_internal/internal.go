package sync_internal

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Blocks []*proto.Block
type Eof = bool
type BlockApplied bool

const NoSignaturesExpected = false
const WaitingForSignatures = true

var NoSignaturesExpectedErr = errors.New("no signatures expected")
var UnexpectedBlockErr = errors.New("unexpected block")

type Internal struct {
	respondedSignatures  *signatures.Signatures
	orderedBlocks        *ordered_blocks.OrderedBlocks
	waitingForSignatures bool
	nearEnd              bool
}

func InternalFromLastSignatures(p types.MessageSender, sigs *signatures.ReverseOrdering) Internal {
	p.SendMessage(&proto.GetSignaturesMessage{Signatures: sigs.Signatures()})
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

func (a Internal) Signatures(p types.MessageSender, sigs []crypto.Signature) (Internal, error) {
	if !a.waitingForSignatures {
		return a, NoSignaturesExpectedErr
	}
	var newSigs []crypto.Signature
	for _, sig := range sigs {
		if a.respondedSignatures.Exists(sig) {
			continue
		}
		newSigs = append(newSigs, sig)
		if a.orderedBlocks.Add(sig) {
			p.SendMessage(&proto.GetBlockMessage{BlockID: sig})
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
	if !a.orderedBlocks.Contains(block.BlockSignature) {
		return a, UnexpectedBlockErr
	}
	a.orderedBlocks.SetBlock(block)
	return a, nil
}

func (a Internal) Blocks(p types.MessageSender) (Internal, Blocks, Eof) {
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
		p.SendMessage(&proto.GetSignaturesMessage{Signatures: a.respondedSignatures.Signatures()})
		return NewInternal(a.orderedBlocks, a.respondedSignatures, WaitingForSignatures, a.nearEnd), blocks, false
	}
	return NewInternal(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.nearEnd), blocks, false
}

func (a Internal) AvailableCount() int {
	return a.orderedBlocks.AvailableCount()
}

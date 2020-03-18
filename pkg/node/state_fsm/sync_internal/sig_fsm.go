package sync_internal

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	//storage "github.com/wavesplatform/gowaves/pkg/state"
)

type Blocks []*proto.Block
type Eof bool
type BlockApplied bool

const NoSignaturesExpected = false
const WaitingForSignatures = true

var NoSignaturesExpectedErr = errors.New("no signatures expected")
var UnexpectedBlockErr = errors.New("unexpected block")

type SigFSM struct {
	respondedSignatures  *signatures.Signatures
	orderedBlocks        *ordered_blocks.OrderedBlocks
	waitingForSignatures bool
	nearEnd              bool
}

func NewSigFSM(orderedBlocks *ordered_blocks.OrderedBlocks, respondedSignatures *signatures.Signatures, waitingForSignatures bool, nearEnd bool) SigFSM {
	return SigFSM{
		respondedSignatures:  respondedSignatures,
		orderedBlocks:        orderedBlocks,
		waitingForSignatures: waitingForSignatures,
		nearEnd:              nearEnd,
	}
}

func (a SigFSM) Signatures(p types.MessageSender, sigs []crypto.Signature) (SigFSM, error) {
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
	return NewSigFSM(a.orderedBlocks, respondedSignatures, NoSignaturesExpected, respondedSignatures.Len() < 100), nil
}

func (a SigFSM) NearEnd() bool {
	return a.nearEnd
}

func (a SigFSM) Block(block *proto.Block) (SigFSM, error) {
	if !a.orderedBlocks.Contains(block.BlockSignature) {
		return a, UnexpectedBlockErr
	}
	a.orderedBlocks.SetBlock(block)
	return a, nil
}

func (a SigFSM) Blocks(p types.MessageSender) (SigFSM, Blocks) {
	blocks := a.orderedBlocks.PopAll()
	if a.nearEnd {
		return NewSigFSM(a.orderedBlocks, a.respondedSignatures, NoSignaturesExpected, a.nearEnd), blocks
	}
	if a.waitingForSignatures {
		return NewSigFSM(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.nearEnd), blocks
	}
	if a.orderedBlocks.WaitingCount() < 100 {
		p.SendMessage(&proto.GetSignaturesMessage{Signatures: a.respondedSignatures.Signatures()})
		return NewSigFSM(a.orderedBlocks, a.respondedSignatures, WaitingForSignatures, a.nearEnd), blocks
	}
	return NewSigFSM(a.orderedBlocks, a.respondedSignatures, a.waitingForSignatures, a.nearEnd), blocks
}

func (a SigFSM) AvailableCount() int {
	return a.orderedBlocks.AvailableCount()
}

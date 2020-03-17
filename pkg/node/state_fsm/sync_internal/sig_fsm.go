package sync_internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
)

type Blocks []*proto.Block
type Eof bool
type BlockApplied bool

type SigFSM struct {
	storage        storage.State
	lastSignatures *signatures.Signatures
	sigs           *ordered_blocks.OrderedBlocks

	signaturesRequested bool
}

func NewSigFSM(storage storage.State, p Peer) *SigFSM {
	panic("implement me")
}

func (a *SigFSM) Signatures(p Peer, sigs []crypto.Signature) *SigFSM {
	panic("implement me")
	//var newSigs []crypto.Signature
	//for _, sig := range sigs {
	//	if a.lastSignatures.Exists(sig) {
	//		continue
	//	}
	//	newSigs = append(newSigs, sig)
	//	if a.sigs.Add(sig) {
	//		p.SendMessage(&proto.GetBlockMessage{BlockID: sig})
	//	}
	//
	//}
	//a.lastSignatures = signatures.NewSignatures(newSigs...).Revert()
	//a.signaturesRequested = false
}

func (a *SigFSM) Block(p Peer, block *proto.Block) (*SigFSM, Blocks, BlockApplied, Eof) {
	panic("implment me")
	//if !a.sigs.Contains(block.BlockSignature) {
	//	return a, nil, false, false
	//}
	//a.sigs.SetBlock(block)
	//
	//// seems we are near end of blockchain, so no need to ask more
	//if a.lastSignatures.Len() < 100 {
	//	blocks := a.sigs.PopAll()
	//	if a.sigs.WaitingLen() == 0 {
	//		// that is the end, halt
	//		return nil, blocks, true, true
	//	}
	//	return a, blocks, true, false
	//}
	//
	//blocks := a.sigs.PopAll()

}

func (a *SigFSM) requestSignatures(peer Peer) *SigFSM {
	panic("impleent me")
	//if a.signaturesRequested {
	//	return a
	//}
	//// check need to request new signatures, or enough
	//if a.sigs.WaitingLen() < 100 {
	//
	//	peer.SendMessage(&proto.GetSignaturesMessage{Signatures: a.lastSignatures.Signatures()})
	//	a.signaturesRequested = true
	//	a.getSignaturesLastRequest = time.Now()
	//	return a, nil
	//}
	//return a, nil, nil
}

package sync_internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
)

const MINIMUM_COUNT = 50

type BlocksApplier interface {
	Apply(state storage.State, block []*proto.Block) error
}

type syncBlock struct {
	lastSignatures      *Signatures
	signaturesRequested bool
	sigs                *OrderedBlocks
	peerSyncWith        Peer
	// list of blocks received from donor peer
	receivedBlocks []*proto.Block
	//minimum count of block to apply
	//n int

}

type SyncActions interface {
	Signatures(peer Peer, sigs []crypto.Signature)
}
type SyncActionsImpl struct {
	syncBlock syncBlock
	storage   storage.State
	applier   BlocksApplier
}

func (a *SyncActionsImpl) Signatures(peer Peer, sigs []crypto.Signature) {
	var newSigs []crypto.Signature
	for _, sig := range sigs {
		if a.syncBlock.lastSignatures.Exists(sig) {
			continue
		}
		newSigs = append(newSigs, sig)
		if a.syncBlock.sigs.Add(sig) {
			peer.SendMessage(&proto.GetBlockMessage{BlockID: sig})
		}
	}
	a.syncBlock.lastSignatures = NewSignatures(newSigs...).Revert()
	a.syncBlock.signaturesRequested = false
}

func (a *SyncActionsImpl) Block(peer Peer, block *proto.Block) {

}

//func (a *SyncActionsImpl) syncAction(peer Peer, block *proto.Block) error {
//	defer a.storage.Mutex().Lock().Unlock()
//	//if a.syncBlock.peerSyncWith != peer {
//	//	return a, nil, nil
//	//}
//	if !a.syncBlock.sigs.contains(block.BlockSignature) {
//		return nil
//	}
//	a.syncBlock.sigs.SetBlock(block)
//
//	blocks := a.syncBlock.sigs.PopAll()
//	a.syncBlock.receivedBlocks = append(a.syncBlock.receivedBlocks, blocks...)
//	// apply block
//	if len(a.syncBlock.receivedBlocks) >= MINIMUM_COUNT {
//		err := a.applier.Apply(a.storage, a.syncBlock.receivedBlocks)
//		a.syncBlock.receivedBlocks = nil
//		if err != nil {
//			return err
//		}
//	}
//
//	if len(a.syncBlock.receivedBlocks) > 0 && len(a.syncBlock.lastSignatures.Signatures()) < 100 {
//		err := a.applier.Apply(a.storage, a.syncBlock.receivedBlocks)
//		a.syncBlock.receivedBlocks = nil
//		if err != nil {
//			return err
//		}
//	}
//
//	peerSyncWithScore, err := a.baseInfo.peers.Score(peer)
//	if err != nil {
//		return NewIdleFsm(a.baseInfo), nil, err
//	}
//	curScore, err := a.baseInfo.storage.CurrentScore()
//	if err != nil {
//		return NewIdleFsm(a.baseInfo), nil, err
//	}
//	// we are at the end
//	if curScore.Cmp(peerSyncWithScore) >= 0 {
//		return NewNGFsm(a.baseInfo)
//	}
//
//	a.requestSignatures(peer)
//
//	return a, nil, nil
//}

type Signatures struct {
	signatures []crypto.Signature
	unique     map[crypto.Signature]struct{}
}

func (a *Signatures) Signatures() []crypto.Signature {
	return a.signatures
}

func NewSignatures(signatures ...crypto.Signature) *Signatures {
	unique := make(map[crypto.Signature]struct{})
	for _, v := range signatures {
		unique[v] = struct{}{}
	}

	return &Signatures{
		signatures: signatures,
		unique:     unique,
	}
}

func (a *Signatures) Exists(sig crypto.Signature) bool {
	_, ok := a.unique[sig]
	return ok
}

func (a *Signatures) Revert() *Signatures {
	out := make([]crypto.Signature, len(a.signatures))
	for k, v := range a.signatures {
		out[len(a.signatures)-1-k] = v
	}
	return NewSignatures(out...)
}

func LastSignatures(state storage.State) (*Signatures, error) {
	var signatures []crypto.Signature

	height, err := state.Height()
	if err != nil {
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, err := state.HeightToBlockID(height)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}

type OrderedBlocks struct {
	sigSequence    []crypto.Signature
	uniqSignatures map[crypto.Signature]*proto.Block
}

func NewOrderedBlocks() *OrderedBlocks {
	return &OrderedBlocks{
		sigSequence:    nil,
		uniqSignatures: make(map[crypto.Signature]*proto.Block),
	}
}

func (a *OrderedBlocks) contains(sig crypto.Signature) bool {
	_, ok := a.uniqSignatures[sig]
	return ok
}

func (a *OrderedBlocks) SetBlock(b *proto.Block) {
	a.uniqSignatures[b.BlockSignature] = b
}

func (a *OrderedBlocks) pop() (crypto.Signature, *proto.Block, bool) {
	if len(a.sigSequence) == 0 {
		return crypto.Signature{}, nil, false
	}
	firstSig := a.sigSequence[0]
	bts := a.uniqSignatures[firstSig]
	if bts != nil {
		delete(a.uniqSignatures, firstSig)
		a.sigSequence = a.sigSequence[1:]
		return firstSig, bts, true
	}
	return crypto.Signature{}, nil, false
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
func (a *OrderedBlocks) Add(sig crypto.Signature) bool {
	// already contains
	if _, ok := a.uniqSignatures[sig]; ok {
		return false
	}
	a.sigSequence = append(a.sigSequence, sig)
	a.uniqSignatures[sig] = nil
	return true
}

func (a *OrderedBlocks) WaitingLen() int {
	return len(a.sigSequence)
}

package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

const MINIMUM_COUNT = 50

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

//type BlocksApplier interface {
//	Apply(state storage.State, blocks []*proto.Block) error
//}

type SyncFsm struct {
	baseInfo  BaseInfo
	syncBlock syncBlock
	applier   BlocksApplier
}

// ignone microblocks
func (a *SyncFsm) MicroBlock(_ Peer, _ *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

// ignore microblocks
func (a *SyncFsm) MicroBlockInv(_ Peer, _ *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *SyncFsm) Task(task AsyncTask) (FSM, Async, error) {
	zap.S().Debugf("SyncFsm Task: got task type %d, data %+v", task.TaskType, task.Data)
	switch task.TaskType {
	case ASK_PEERS:
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("SyncFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *SyncFsm) PeerError(p Peer, e error) (FSM, Async, error) {
	if a.syncBlock.peerSyncWith == p {
		if len(a.syncBlock.receivedBlocks) > 0 {
			err := a.applier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
			if err != nil {
				zap.S().Error(err)
				return NewIdleFsm(a.baseInfo), nil, err
			}
		}
	}
	a.baseInfo.peers.Disconnect(p)
	return NewIdleFsm(a.baseInfo), nil, nil
}

func (a *SyncFsm) Signatures(peer Peer, sigs []crypto.Signature) (FSM, Async, error) {
	if a.syncBlock.peerSyncWith == peer {
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
	return a, nil, nil
}

func (a *SyncFsm) NewPeer(p Peer) (FSM, Async, error) {
	err := a.baseInfo.peers.NewConnection(p)
	return a, nil, err
}

func (a *SyncFsm) Score(p Peer, score *proto.Score) (FSM, Async, error) {
	// TODO handle new max score
	err := a.baseInfo.peers.UpdateScore(p, score)
	if err != nil {
		return a, nil, err
	}
	return a, nil, nil
}

func (a *SyncFsm) Block(peer Peer, block *proto.Block) (FSM, Async, error) {
	return a.syncAction(peer, block)
}

func (a *SyncFsm) syncAction(peer Peer, block *proto.Block) (FSM, Async, error) {
	defer a.baseInfo.storage.Mutex().Lock().Unlock()
	if a.syncBlock.peerSyncWith != peer {
		return a, nil, nil
	}
	if !a.syncBlock.sigs.contains(block.BlockSignature) {
		return a, nil, nil
		//return FsmBlockRet{
		//	Error: ErrUnexpectedBlock,
		//	State: SYNC,
		//}
	}
	a.syncBlock.sigs.SetBlock(block)

	blocks := a.syncBlock.sigs.PopAll()
	a.syncBlock.receivedBlocks = append(a.syncBlock.receivedBlocks, blocks...)
	// apply block
	if len(a.syncBlock.receivedBlocks) >= MINIMUM_COUNT {
		//_, err := a.baseInfo.storage.AddNewDeserializedBlocks(a.syncBlock.receivedBlocks)
		err := a.applier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
		a.syncBlock.receivedBlocks = nil
		if err != nil {
			zap.S().Error(err)
			return NewIdleFsm(a.baseInfo), nil, err
		}
	}

	if len(a.syncBlock.receivedBlocks) > 0 && len(a.syncBlock.lastSignatures.Signatures()) < 100 {
		//_, err := a.baseInfo.storage.AddNewDeserializedBlocks(a.syncBlock.receivedBlocks)
		err := a.applier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
		a.syncBlock.receivedBlocks = nil
		if err != nil {
			zap.S().Error(err)
			return NewIdleFsm(a.baseInfo), nil, err
		}
	}

	peerSyncWithScore, err := a.baseInfo.peers.Score(peer)
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	curScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	// we are at the end
	if curScore.Cmp(peerSyncWithScore) >= 0 {
		return NewNGFsm(a.baseInfo)
	}

	a.requestSignatures(peer)

	return a, nil, nil
}

func (a *SyncFsm) requestSignatures(peer Peer) {
	if a.syncBlock.signaturesRequested {
		return
	}
	// check need to request new signatures, or enough
	if a.syncBlock.sigs.WaitingLen() < 100 {
		// seems we are near end of blockchain, so no need to ask more
		if len(a.syncBlock.lastSignatures.Signatures()) < 100 {
			return //a, nil, nil
		}
		peer.SendMessage(&proto.GetSignaturesMessage{Blocks: a.syncBlock.lastSignatures.Signatures()})
		a.syncBlock.signaturesRequested = true
	}
}

func NewSyncFsm(baseInfo BaseInfo, syncBlock syncBlock, applier BlocksApplier) *SyncFsm {
	return &SyncFsm{
		baseInfo:  baseInfo,
		syncBlock: syncBlock,
		applier:   applier,
	}
}

func NewIdleToSyncTransition(baseInfo BaseInfo, p Peer) (FSM, Async, error) {
	lastSigs, err := LastSignatures(baseInfo.storage)
	if err != nil {
		return NewIdleFsm(baseInfo), nil, err
	}
	p.SendMessage(&proto.GetSignaturesMessage{Blocks: lastSigs.Signatures()})

	o := NewOrderedBlocks()
	s := syncBlock{
		lastSignatures:      lastSigs,
		signaturesRequested: true,
		sigs:                o,
		peerSyncWith:        p,
	}
	// TODO timeout
	return NewSyncFsm(baseInfo, s, blocks_applier.NewBlocksApplier()), nil, nil
}

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
		zap.S().Error(err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, err := state.HeightToBlockID(height)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}

package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const MINIMUM_COUNT = 50

type syncBlock struct {
	lastSignatures      *signatures.Signatures
	signaturesRequested bool
	sigs                *ordered_blocks.OrderedBlocks
	peerSyncWith        Peer
	// list of blocks received from donor peer
	receivedBlocks []*proto.Block

	// if nothing happens more than N duration, means we stalled, so go to idle and again
	lastReceiveTime time.Time

	timeout time.Duration
}

type SyncFsm struct {
	baseInfo  BaseInfo
	syncBlock syncBlock
}

// ignore microblocks
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
	case PING:
		timeout := a.syncBlock.lastReceiveTime.Add(a.syncBlock.timeout).Before(a.baseInfo.tm.Now())
		if timeout {
			return NewIdleFsm(a.baseInfo), nil, TimeoutErr
		}
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("SyncFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

func (a *SyncFsm) PeerError(p Peer, e error) (FSM, Async, error) {
	if a.syncBlock.peerSyncWith == p {
		if len(a.syncBlock.receivedBlocks) > 0 {
			locked := a.baseInfo.storage.Mutex().Lock()
			zap.S().Debug("PeerError before")
			err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
			zap.S().Debug("PeerError after")
			locked.Unlock()
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
		a.syncBlock.lastSignatures = signatures.NewSignatures(newSigs...).Revert()
		a.syncBlock.signaturesRequested = false
		a.syncBlock.lastReceiveTime = time.Now()
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
	if a.syncBlock.peerSyncWith != peer {
		return a, nil, nil
	}
	if !a.syncBlock.sigs.Contains(block.BlockSignature) {
		return a, nil, nil
	}
	a.syncBlock.sigs.SetBlock(block)
	a.syncBlock.lastReceiveTime = time.Now()

	blocks := a.syncBlock.sigs.PopAll()
	a.syncBlock.receivedBlocks = append(a.syncBlock.receivedBlocks, blocks...)

	// apply block
	if len(a.syncBlock.receivedBlocks) >= MINIMUM_COUNT {
		//zap.S().Debug("MINIMUM_COUNT before")
		lock := a.baseInfo.storage.Mutex().Lock()
		err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
		lock.Unlock()
		//zap.S().Debug("MINIMUM_COUNT after")
		a.syncBlock.receivedBlocks = nil
		if err != nil {
			zap.S().Error(err)
			return NewIdleFsm(a.baseInfo), nil, err
		}
	}

	if len(a.syncBlock.receivedBlocks) > 0 && len(a.syncBlock.lastSignatures.Signatures()) < 100 {
		//zap.S().Debug("sig < 100 before")
		lock := a.baseInfo.storage.Mutex().Lock()
		err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, a.syncBlock.receivedBlocks)
		lock.Unlock()
		//zap.S().Debug("sig < 100 after")
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
	rlock := a.baseInfo.storage.Mutex().RLock()
	curScore, err := a.baseInfo.storage.CurrentScore()
	rlock.Unlock()
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	// we are at the end
	if curScore.Cmp(peerSyncWithScore) >= 0 {
		return NewNGFsm(a.baseInfo)
	}

	return a.requestSignatures(peer)
}

func (a *SyncFsm) requestSignatures(peer Peer) (FSM, Async, error) {
	if a.syncBlock.signaturesRequested {
		return a, nil, nil
	}
	// check need to request new signatures, or enough
	if a.syncBlock.sigs.WaitingCount() < 100 {
		// seems we are near end of blockchain, so no need to ask more
		if len(a.syncBlock.lastSignatures.Signatures()) < 100 {
			return a, nil, nil
		}
		peer.SendMessage(&proto.GetSignaturesMessage{Signatures: a.syncBlock.lastSignatures.Signatures()})
		a.syncBlock.signaturesRequested = true
		return a, nil, nil
	}
	return a, nil, nil
}

func NewSyncFsm(baseInfo BaseInfo, p Peer) (FSM, Async, error) {
	return NewSyncFsmExtended(baseInfo, p, signatures.LastSignaturesImpl{}, 30*time.Second)
}

func NewSyncFsmExtended(baseInfo BaseInfo, p Peer, lastSignatures signatures.LastSignatures, timeout time.Duration) (FSM, Async, error) {
	lastSigs, err := lastSignatures.LastSignatures(baseInfo.storage)
	if err != nil {
		return NewIdleFsm(baseInfo), nil, err
	}
	p.SendMessage(&proto.GetSignaturesMessage{Signatures: lastSigs.Signatures()})

	o := ordered_blocks.NewOrderedBlocks()
	s := syncBlock{
		lastSignatures:      lastSigs,
		signaturesRequested: true,
		sigs:                o,
		peerSyncWith:        p,

		lastReceiveTime: time.Now(),

		timeout: timeout,
	}

	return &SyncFsm{
		baseInfo:  baseInfo,
		syncBlock: s,
	}, nil, nil
}

func NewIdleToSyncTransition(baseInfo BaseInfo, p Peer) (FSM, Async, error) {
	return NewSyncFsm(baseInfo, p)
}

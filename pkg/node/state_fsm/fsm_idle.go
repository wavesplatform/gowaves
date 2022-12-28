package state_fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type InvRequester interface {
	Add2Cache(id []byte) (existed bool)
	Request(p types.MessageSender, id []byte) (existed bool)
}

var (
	idleSkipMessageList = proto.PeerMessageIDs{
		proto.ContentIDSignatures,
		proto.ContentIDBlock,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDBlockIds,
	}
)

type IdleFsm struct {
	baseInfo BaseInfo
}

func (a *IdleFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *IdleFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.baseInfo)
}

func (a *IdleFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return MinedBlockNgTransition(a.baseInfo, block, limits, keyPair, vrf)
}

func (a *IdleFsm) MicroBlock(_ peer.Peer, _ *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) MicroBlockInv(_ peer.Peer, _ *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) Task(task tasks.AsyncTask) (FSM, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return noop(a)
	case tasks.AskPeers:
		zap.S().Debug("[Idle] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro: // Do nothing
		return a, nil, nil
	default:
		return a, nil, a.Errorf(errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s FSM", task.TaskType, task.Data, a.String()))
	}
}

func (a *IdleFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return a.baseInfo.d.PeerError(a, p, a.baseInfo, e)
}

func (a *IdleFsm) BlockIDs(_ peer.Peer, _ []proto.BlockID) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *IdleFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	fsm, as, fsmErr := newPeer(a, p, a.baseInfo.peers)
	if a.baseInfo.peers.ConnectedCount() == a.baseInfo.minPeersMining {
		a.baseInfo.Reschedule()
	}
	sendScore(p, a.baseInfo.storage)
	return fsm, as, fsmErr
}

func (a *IdleFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	metrics.FSMScore("idle", score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	nodeScore, err := a.baseInfo.storage.CurrentScore()
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if score.Cmp(nodeScore) == 1 {
		return syncWithNewPeer(a, a.baseInfo, p)
	}
	return noop(a)
}

func (a *IdleFsm) Block(_ peer.Peer, _ *proto.Block) (FSM, Async, error) {
	return noop(a)
}

func (a *IdleFsm) String() string {
	return "Idle"
}

func (a *IdleFsm) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func NewIdleFsm(info BaseInfo) *IdleFsm {
	info.skipMessageList.SetList(idleSkipMessageList)
	return &IdleFsm{
		baseInfo: info,
	}
}

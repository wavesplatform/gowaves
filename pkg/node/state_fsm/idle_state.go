package state_fsm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

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

type InvRequester interface {
	Add2Cache(id []byte) (existed bool)
	Request(p types.MessageSender, id []byte) (existed bool)
}

type IdleState struct {
	baseInfo BaseInfo
}

func (a *IdleState) String() string {
	return IdleStateName
}

func (a *IdleState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func newIdleState(info BaseInfo) State {
	return &IdleState{
		baseInfo: info,
	}
}

func (a *IdleState) Transaction(p peer.Peer, t proto.Transaction) (State, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *IdleState) PeerError(p peer.Peer, e error) (State, Async, error) {
	return peerError(a, p, a.baseInfo, e)
}

func (a *IdleState) NewPeer(p peer.Peer) (State, Async, error) {
	state, as, fsmErr := newPeer(a, p, a.baseInfo.peers)
	if a.baseInfo.peers.ConnectedCount() == a.baseInfo.minPeersMining {
		a.baseInfo.Reschedule()
	}
	sendScore(p, a.baseInfo.storage)
	return state, as, fsmErr
}

func (a *IdleState) ConnectedNewPeer(_ peer.Peer) (State, Async, error) {
	a.baseInfo.Reschedule()
	return a, nil, nil
}

func (a *IdleState) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (State, Async, error) {
	newA := newNGState(a.baseInfo).(*NGState)
	return newA.MinedBlock(block, limits, keyPair, vrf)
}

func (a *IdleState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return a, nil, nil
	case tasks.AskPeers:
		zap.S().Debug("[Idle] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro: // Do nothing
		return a, nil, nil
	default:
		return a, nil, a.Errorf(errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s State", task.TaskType, task.Data, a.String()))
	}
}

func (a *IdleState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
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
	return a, nil, nil
}

func (a *IdleState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

func initIdleStateInFSM(state *StateData, fsm *stateless.StateMachine, b BaseInfo) {
	fsm.Configure(IdleStateName).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			b.skipMessageList.SetList(idleSkipMessageList)
			return nil
		}).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(BlockIDsEvent).
		Ignore(BlockEvent).
		Ignore(DisconnectedPeerEvent).
		Ignore(ConnectedBestPeerEvent).
		Ignore(DisconnectedBestPeerEvent).
		PermitDynamic(ConnectedPeerEvent, createPermitDynamicCallback(ConnectedPeerEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.ConnectedNewPeer(convertToInterface[peer.Peer](args[0]))
		})).
		PermitDynamic(TransactionEvent, createPermitDynamicCallback(TransactionEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.Transaction(convertToInterface[peer.Peer](args[0]), convertToInterface[proto.Transaction](args[1]))
		})).
		PermitDynamic(ScoreEvent, createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
		})).
		PermitDynamic(TaskEvent, createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.Task(args[0].(tasks.AsyncTask))
		})).
		PermitDynamic(MinedBlockEvent, createPermitDynamicCallback(MinedBlockEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits), args[2].(proto.KeyPair), args[3].([]byte))
		})).
		PermitDynamic(HaltEvent, createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*IdleState)
			return a.Halt()
		}))
}

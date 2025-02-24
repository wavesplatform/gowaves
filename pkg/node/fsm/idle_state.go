package fsm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type InvRequester interface {
	Add2Cache(id proto.BlockID) (existed bool)
	Request(p types.MessageSender, id proto.BlockID) (existed bool)
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
	info.syncPeer.Clear()
	return &IdleState{
		baseInfo: info,
	}
}

func (a *IdleState) Transaction(p peer.Peer, t proto.Transaction) (State, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *IdleState) StartMining() (State, Async, error) {
	a.baseInfo.scheduler.Reschedule()
	return a, nil, nil
}

func (a *IdleState) MinedBlock(
	block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	newA, ok := newNGState(a.baseInfo).(*NGState)
	if !ok {
		return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*NGState'", a.baseInfo))
	}
	return newA.MinedBlock(block, limits, keyPair, vrf)
}

func (a *IdleState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return a, nil, nil
	case tasks.AskPeers:
		zap.S().Named(logging.FSMNamespace).Debug("[Idle] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro: // Do nothing
		return a, nil, nil
	case tasks.SnapshotTimeout:
		return a, nil, nil
	default:
		return a, nil, a.Errorf(errors.Errorf(
			"unexpected internal task '%d' with data '%+v' received by %s State",
			task.TaskType, task.Data, a.String(),
		))
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
		// received score is larger than local score
		return syncWithNewPeer(a, a.baseInfo, p)
	}
	return a, nil, nil
}

func (a *IdleState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

func initIdleStateInFSM(state *StateData, fsm *stateless.StateMachine, b BaseInfo) {
	idleSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDSignatures,
		proto.ContentIDBlock,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDBlockIDs,
		proto.ContentIDBlockSnapshot,
		proto.ContentIDMicroBlockSnapshot,
		proto.ContentIDMicroBlockSnapshotRequest,
	}
	fsm.Configure(IdleStateName).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			b.skipMessageList.SetList(idleSkipMessageList)
			return nil
		}).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(BlockIDsEvent).
		Ignore(BlockEvent).
		Ignore(StopSyncEvent).
		Ignore(ChangeSyncPeerEvent).
		Ignore(StopMiningEvent).
		Ignore(BlockSnapshotEvent).
		Ignore(MicroBlockSnapshotEvent).
		PermitDynamic(StartMiningEvent,
			createPermitDynamicCallback(StartMiningEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.StartMining()
			})).
		PermitDynamic(TransactionEvent,
			createPermitDynamicCallback(TransactionEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.Transaction(convertToInterface[peer.Peer](args[0]),
					convertToInterface[proto.Transaction](args[1]))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(MinedBlockEvent,
			createPermitDynamicCallback(MinedBlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits), args[2].(proto.KeyPair),
					args[3].([]byte))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*IdleState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf("unexpected type '%T' expected '*IdleState'",
						state.State))
				}
				return a.Halt()
			}))
}

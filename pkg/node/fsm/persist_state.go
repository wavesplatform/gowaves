package fsm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"

	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PersistState struct {
	baseInfo BaseInfo
}

func (a *PersistState) String() string {
	return PersistStateName
}

func (a *PersistState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func newPersistState(info BaseInfo) (State, Async, error) {
	info.syncPeer.Clear()
	t := tasks.NewFuncTask(func(ctx context.Context, output chan tasks.AsyncTask) error {
		err := info.storage.PersistAddressTransactions()
		tasks.SendAsyncTask(output, tasks.AsyncTask{
			TaskType: tasks.PersistComplete,
		})
		return err
	}, tasks.PersistComplete)

	return &PersistState{
		baseInfo: info,
	}, tasks.Tasks(t), nil
}

func (a *PersistState) StopMining() (State, Async, error) {
	return newIdleState(a.baseInfo), nil, nil
}

func (a *PersistState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	return a, nil, nil
}

func (a *PersistState) Task(t tasks.AsyncTask) (State, Async, error) {
	switch t.TaskType {
	case tasks.PersistComplete:
		return newIdleState(a.baseInfo), nil, nil
	default:
		return a, nil, nil
	}
}

func (a *PersistState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

func initPersistStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	persistSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDGetBlockIDs,
		proto.ContentIDBlockSnapshot,
		proto.ContentIDMicroBlockSnapshot,
		proto.ContentIDGetBlockSnapshot,
		proto.ContentIDMicroBlockSnapshotRequest,
	}
	fsm.Configure(PersistStateName).
		Ignore(BlockEvent).
		Ignore(MinedBlockEvent).
		Ignore(BlockIDsEvent).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(TransactionEvent).
		Ignore(StartMiningEvent).
		Ignore(ChangeSyncPeerEvent).
		Ignore(StopSyncEvent).
		Ignore(BlockSnapshotEvent).
		Ignore(MicroBlockSnapshotEvent).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			info.skipMessageList.SetList(persistSkipMessageList)
			return nil
		}).
		PermitDynamic(StopMiningEvent,
			createPermitDynamicCallback(StopMiningEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*PersistState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*PersistState'", state.State))
				}
				return a.StopMining()
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*PersistState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*PersistState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*PersistState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*PersistState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*PersistState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*PersistState'", state.State))
				}
				return a.Halt()
			}))
}

package fsm

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	snapshotTimeout = 30 * time.Second
)

type WaitSnapshotState struct {
	baseInfo                BaseInfo
	blocksCache             blockStatesCache
	timeoutTaskOutdated     chan<- struct{}
	blockWaitingForSnapshot *proto.Block

	receivedScores []ReceivedScore
}

type ReceivedScore struct {
	Peer  peer.Peer
	Score *proto.Score
}

func newWaitSnapshotState(baseInfo BaseInfo, block *proto.Block, cache blockStatesCache) (State, tasks.Task) {
	baseInfo.syncPeer.Clear()
	timeoutTaskOutdated := make(chan struct{})
	st := &WaitSnapshotState{
		baseInfo:                baseInfo,
		blocksCache:             cache,
		timeoutTaskOutdated:     timeoutTaskOutdated,
		blockWaitingForSnapshot: block,
		receivedScores:          nil,
	}
	task := tasks.NewBlockSnapshotTimeoutTask(snapshotTimeout, block.BlockID(), timeoutTaskOutdated)
	return st, task
}

func (a *WaitSnapshotState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func (a *WaitSnapshotState) String() string {
	return WaitSnapshotStateName
}

func (a *WaitSnapshotState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return a, nil, nil
	case tasks.AskPeers:
		a.baseInfo.logger.Debug("Requesting peers", "state", a.String())
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.MineMicro:
		return a, nil, nil
	case tasks.SnapshotTimeout:
		t, ok := task.Data.(tasks.SnapshotTimeoutTaskData)
		if !ok {
			return a, nil, a.Errorf(errors.Errorf(
				"unexpected type %T, expected 'tasks.SnapshotTimeoutTaskData'", task.Data))
		}
		switch t.SnapshotTaskType {
		case tasks.BlockSnapshot:
			defer a.cleanupBeforeTransition()
			slog.Error("Timed out to get snapshot for block", "state", a.String(),
				"blockID", t.BlockID)
			return processScoreAfterApplyingOrReturnToNG(a, a.baseInfo, a.receivedScores, a.blocksCache)
		case tasks.MicroBlockSnapshot:
			return a, nil, nil
		default:
			return a, nil, a.Errorf(errors.New("undefined Snapshot Task type"))
		}
	default:
		return a, nil, a.Errorf(errors.Errorf(
			"unexpected internal task '%d' with data '%+v' received by %s State",
			task.TaskType, task.Data, a.String()))
	}
}

func (a *WaitSnapshotState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	metrics.Score(score, p.Handshake().NodeName)
	if len(a.receivedScores) < scoresSliceMaxSize {
		a.receivedScores = append(a.receivedScores, ReceivedScore{Peer: p, Score: score})
	}
	return a, nil, nil
}

func (a *WaitSnapshotState) BlockSnapshot(
	_ peer.Peer,
	blockID proto.BlockID,
	snapshot proto.BlockSnapshot,
) (State, Async, error) {
	if a.blockWaitingForSnapshot.BlockID() != blockID {
		return a, nil, a.Errorf(
			errors.Errorf("new snapshot doesn't match with block %s", a.blockWaitingForSnapshot.BlockID()))
	}

	defer a.cleanupBeforeTransition()
	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(heightErr)
	}
	_, err := a.baseInfo.blocksApplier.ApplyWithSnapshots(
		a.baseInfo.storage,
		[]*proto.Block{a.blockWaitingForSnapshot},
		[]*proto.BlockSnapshot{&snapshot},
	)
	if err != nil {
		slog.Error("Failed to apply block with snapshot", slog.String("state", a.String()),
			logging.Error(err), slog.Any("blockID", a.blockWaitingForSnapshot.BlockID()))
		return processScoreAfterApplyingOrReturnToNG(a, a.baseInfo, a.receivedScores, a.blocksCache)
	}
	metrics.SnapshotBlockApplied(a.blockWaitingForSnapshot, height+1)
	metrics.Utx(a.baseInfo.utx.Count())
	a.baseInfo.logger.Debug("Handle received key block message: block applied to state",
		"state", a.String(), "blockID", blockID)

	a.blocksCache.Clear()
	a.blocksCache.AddBlockState(a.blockWaitingForSnapshot)
	a.blocksCache.AddSnapshot(blockID, snapshot)
	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendBlock(a.blockWaitingForSnapshot)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()
	return processScoreAfterApplyingOrReturnToNG(a, a.baseInfo, a.receivedScores, a.blocksCache)
}

func (a *WaitSnapshotState) cleanupBeforeTransition() {
	a.blockWaitingForSnapshot = nil
	if a.timeoutTaskOutdated != nil {
		close(a.timeoutTaskOutdated)
		a.timeoutTaskOutdated = nil
	}
	a.receivedScores = nil
}

func initWaitSnapshotStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	waitSnapshotSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDGetPeers,
		proto.ContentIDPeers,
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDGetBlockIDs,
	}
	fsm.Configure(WaitSnapshotStateName). //nolint:dupl // it's state setup
						OnEntry(func(_ context.Context, _ ...any) error {
			info.skipMessageList.SetList(waitSnapshotSkipMessageList)
			return nil
		}).
		Ignore(BlockEvent).
		Ignore(MinedBlockEvent).
		Ignore(BlockIDsEvent).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(TransactionEvent).
		Ignore(StopSyncEvent).
		Ignore(StartMiningEvent).
		Ignore(ChangeSyncPeerEvent).
		Ignore(StopMiningEvent).
		Ignore(HaltEvent).
		Ignore(MicroBlockSnapshotEvent).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*WaitSnapshotState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*WaitSnapshotState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(BlockSnapshotEvent,
			createPermitDynamicCallback(BlockSnapshotEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*WaitSnapshotState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*WaitSnapshotState'", state.State))
				}
				return a.BlockSnapshot(
					convertToInterface[peer.Peer](args[0]),
					args[1].(proto.BlockID),
					args[2].(proto.BlockSnapshot),
				)
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*WaitSnapshotState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*WaitSnapshotState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			}))
}

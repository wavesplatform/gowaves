package fsm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type WaitMicroSnapshotState struct {
	baseInfo                     BaseInfo
	blocksCache                  blockStatesCache
	timeoutTaskOutdated          chan<- struct{}
	microBlockWaitingForSnapshot *proto.MicroBlock
}

func newWaitMicroSnapshotState(baseInfo BaseInfo, micro *proto.MicroBlock, cache blockStatesCache) (State, tasks.Task) {
	baseInfo.syncPeer.Clear()
	timeoutTaskOutdated := make(chan struct{})
	st := &WaitMicroSnapshotState{
		baseInfo:                     baseInfo,
		blocksCache:                  cache,
		timeoutTaskOutdated:          timeoutTaskOutdated,
		microBlockWaitingForSnapshot: micro,
	}
	task := tasks.NewMicroBlockSnapshotTimeoutTask(time.Minute, micro.TotalBlockID, timeoutTaskOutdated)
	return st, task
}

func (a *WaitMicroSnapshotState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func (a *WaitMicroSnapshotState) String() string {
	return WaitSnapshotStateName
}

func (a *WaitMicroSnapshotState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.Ping:
		return a, nil, nil
	case tasks.AskPeers:
		zap.S().Named(logging.FSMNamespace).Debug("[WaitSnapshot] Requesting peers")
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
			return a, nil, nil
		case tasks.MicroBlockSnapshot:
			defer a.cleanupBeforeTransition()
			return newNGStateWithCache(a.baseInfo, a.blocksCache), nil, a.Errorf(errors.Errorf(
				"failed to get snapshot for microBlock '%s' - timeout", t.BlockID))
		default:
			return a, nil, a.Errorf(errors.New("undefined Snapshot Task type"))
		}
	default:
		return a, nil, a.Errorf(errors.Errorf(
			"unexpected internal task '%d' with data '%+v' received by %s State",
			task.TaskType, task.Data, a.String()))
	}
}

func (a *WaitMicroSnapshotState) MicroBlockSnapshot(
	_ peer.Peer,
	blockID proto.BlockID,
	snapshot proto.BlockSnapshot,
) (State, Async, error) {
	if a.microBlockWaitingForSnapshot.TotalBlockID != blockID {
		return a, nil, a.Errorf(errors.Errorf(
			"New snapshot doesn't match with microBlock %s", a.microBlockWaitingForSnapshot.TotalBlockID))
	}
	// the TopBlock() is used here
	block, err := a.checkAndAppendMicroBlock(a.microBlockWaitingForSnapshot, &snapshot)
	if err != nil {
		metrics.FSMMicroBlockDeclined("ng", a.microBlockWaitingForSnapshot, err)
		return a, nil, a.Errorf(err)
	}
	defer a.cleanupBeforeTransition()

	zap.S().Named(logging.FSMNamespace).Debugf(
		"[%s] Received snapshot for microblock '%s' successfully applied to state", a, block.BlockID(),
	)
	a.baseInfo.MicroBlockCache.AddMicroBlockWithSnapshot(block.BlockID(), a.microBlockWaitingForSnapshot, &snapshot)
	a.blocksCache.AddBlockState(block)
	a.blocksCache.AddSnapshot(block.BlockID(), snapshot)
	a.baseInfo.scheduler.Reschedule()
	// Notify all connected peers about new microblock, send them microblock inv network message
	if inv, ok := a.baseInfo.MicroBlockInvCache.Get(block.BlockID()); ok {
		//TODO: We have to exclude from recipients peers that already have this microblock
		if err = broadcastMicroBlockInv(a.baseInfo, inv); err != nil {
			return a, nil, a.Errorf(errors.Wrap(err, "failed to handle microblock message"))
		}
	}
	return a, nil, nil
}

func (a *WaitMicroSnapshotState) cleanupBeforeTransition() {
	a.microBlockWaitingForSnapshot = nil
	if a.timeoutTaskOutdated != nil {
		close(a.timeoutTaskOutdated)
		a.timeoutTaskOutdated = nil
	}
}

func (a *WaitMicroSnapshotState) checkAndAppendMicroBlock(
	micro *proto.MicroBlock,
	snapshot *proto.BlockSnapshot,
) (*proto.Block, error) {
	top := a.baseInfo.storage.TopBlock()  // Get the last block
	if top.BlockID() != micro.Reference { // Microblock doesn't refer to last block
		err := errors.Errorf("microblock TBID '%s' refer to block ID '%s' but last block ID is '%s'",
			micro.TotalBlockID.String(), micro.Reference.String(), top.BlockID().String())
		metrics.FSMMicroBlockDeclined("ng", micro, err)
		return &proto.Block{}, proto.NewInfoMsg(err)
	}
	ok, err := micro.VerifySignature(a.baseInfo.scheme)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Errorf("microblock '%s' has invalid signature", micro.TotalBlockID.String())
	}
	newTrs := top.Transactions.Join(micro.Transactions)
	newBlock, err := proto.CreateBlock(newTrs, top.Timestamp, top.Parent, top.GeneratorPublicKey, top.NxtConsensus,
		top.Version, top.Features, top.RewardVote, a.baseInfo.scheme)
	if err != nil {
		return nil, err
	}
	// TODO: check if light node feature activated + 1000 blocks
	newBlock.StateHash = micro.StateHash
	newBlock.BlockSignature = micro.TotalResBlockSigField
	ok, err = newBlock.VerifySignature(a.baseInfo.scheme)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("incorrect signature for applied microblock")
	}
	err = newBlock.GenerateBlockID(a.baseInfo.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "NGState microBlockByID: failed generate block id")
	}
	snapshotsToApply := snapshot

	h, errBToH := a.baseInfo.storage.BlockIDToHeight(top.BlockID())
	if errBToH != nil {
		return nil, errBToH
	}
	topBlockSnapshots, errSAtH := a.baseInfo.storage.SnapshotsAtHeight(h)
	if errSAtH != nil {
		return nil, errSAtH
	}
	for _, sn := range snapshot.TxSnapshots {
		topBlockSnapshots.AppendTxSnapshot(sn)
	}
	snapshotsToApply = &topBlockSnapshots
	err = a.baseInfo.storage.Map(func(state state.State) error {
		_, er := a.baseInfo.blocksApplier.ApplyMicroWithSnapshots(state, newBlock, snapshotsToApply)
		return er
	})

	if err != nil {
		metrics.FSMMicroBlockDeclined("ng", micro, err)
		return nil, errors.Wrap(err, "failed to apply created from micro block")
	}
	metrics.FSMMicroBlockApplied("ng", micro)
	return newBlock, nil
}

func initWaitMicroSnapshotStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	waitSnapshotSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDGetPeers,
		proto.ContentIDPeers,
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDScore,
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
	}
	fsm.Configure(WaitMicroSnapshotStateName). //nolint:dupl // it's state setup
							OnEntry(func(_ context.Context, _ ...interface{}) error {
			info.skipMessageList.SetList(waitSnapshotSkipMessageList)
			return nil
		}).
		Ignore(ScoreEvent).
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
		Ignore(BlockSnapshotEvent).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*WaitMicroSnapshotState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*WaitMicroSnapshotState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(MicroBlockSnapshotEvent,
			createPermitDynamicCallback(MicroBlockSnapshotEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*WaitMicroSnapshotState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*WaitMicroSnapshotState'", state.State))
				}
				return a.MicroBlockSnapshot(
					convertToInterface[peer.Peer](args[0]),
					args[1].(proto.BlockID),
					args[2].(proto.BlockSnapshot),
				)
			}))
}

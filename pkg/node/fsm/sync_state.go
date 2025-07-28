package fsm

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"

	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const defaultMicroblockInterval = 5 * time.Second

type conf struct {
	peerSyncWith peer.Peer
	// if nothing happens more than N duration, means we stalled, so go to idle and again
	lastReceiveTime time.Time

	timeout time.Duration
}

func (c conf) Now(tm types.Time) conf {
	return conf{
		peerSyncWith:    c.peerSyncWith,
		lastReceiveTime: tm.Now(),
		timeout:         c.timeout,
	}
}

type SyncState struct {
	baseInfo BaseInfo
	conf     conf
	internal sync_internal.Internal
}

func (a *SyncState) String() string {
	return SyncStateName
}

func (a *SyncState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func newSyncState(baseInfo BaseInfo, conf conf, internal sync_internal.Internal) State {
	return &SyncState{
		baseInfo: baseInfo,
		conf:     conf,
		internal: internal,
	}
}

func (a *SyncState) Transaction(p peer.Peer, t proto.Transaction) (State, Async, error) {
	return tryBroadcastTransaction(a, a.baseInfo, p, t)
}

func (a *SyncState) StopSync() (State, Async, error) {
	_, blocks, snapshots, _ := a.internal.Blocks()
	if len(blocks) > 0 {
		var err error
		if a.baseInfo.enableLightMode {
			err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
				var errApply error
				_, errApply = a.baseInfo.blocksApplier.ApplyWithSnapshots(s, blocks, snapshots)
				return errApply
			})
		} else {
			err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
				var errApply error
				_, errApply = a.baseInfo.blocksApplier.Apply(s, blocks)
				return errApply
			})
		}
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	return newIdleState(a.baseInfo), nil, nil
}

func (a *SyncState) ChangeSyncPeer(p peer.Peer) (State, Async, error) {
	a.baseInfo.logger.Debug("Sync peer changed", "state", a.String(),
		"peer", p.ID().String())
	return syncWithNewPeer(a, a.baseInfo, p)
}

func (a *SyncState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.AskPeers:
		a.baseInfo.logger.Debug("Requesting peers", "state", a.String())
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.Ping:
		a.baseInfo.logger.Debug("Checking timeout", "state", a.String())
		timeout := a.conf.lastReceiveTime.Add(a.conf.timeout).Before(a.baseInfo.tm.Now())
		if timeout {
			a.baseInfo.logger.Debug("Synchronization with peer timed out", "state", a.String(),
				"timeout", a.conf.timeout.String(), "peer", a.conf.peerSyncWith.ID())
			return newIdleState(a.baseInfo), nil, a.Errorf(TimeoutErr)
		}
		return a, nil, nil
	case tasks.MineMicro:
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

func (a *SyncState) BlockIDs(peer peer.Peer, signatures []proto.BlockID) (State, Async, error) {
	if len(signatures) == 0 {
		a.baseInfo.logger.Debug("Empty IDs received from peer", "state", a.String(),
			"peer", peer.ID().String())
		return a, nil, nil
	}
	a.baseInfo.logger.Debug("Block IDs received from peer", "state", a.String(),
		"from", signatures[0].ShortString(), "to", signatures[len(signatures)-1].ShortString(),
		"peer", peer.ID().String())
	if !peer.Equal(a.conf.peerSyncWith) {
		a.baseInfo.logger.Debug("Block IDs received from incorrect peer", "state", a.String(),
			"peer", peer.ID().String(), "expectedPeer", a.baseInfo.syncPeer.GetPeer().ID().String())
		return a, nil, nil
	}
	internal, err := a.internal.BlockIDs(extension.NewPeerExtension(peer, a.baseInfo.scheme, a.baseInfo.netLogger),
		signatures)
	if err != nil {
		a.baseInfo.logger.Debug("No signatures expected from peer, but received", "state", a.String(),
			"peer", peer.ID().String())
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	if internal.RequestedCount() > 0 {
		// Blocks were requested waiting for them to receive and apply
		a.baseInfo.logger.Debug("Waiting for blocks to receive", "state", a.String(),
			"count", internal.RequestedCount())
		return newSyncState(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal), nil, nil
	}
	a.baseInfo.logger.Debug("Continue to NG state", "state", a.String())
	// No blocks were request, switching to NG working mode
	err = a.baseInfo.storage.StartProvidingExtendedApi()
	if err != nil {
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	return newNGState(a.baseInfo), nil, nil
}

func (a *SyncState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	metrics.Score(score, p.Handshake().NodeName)
	a.baseInfo.logger.Debug("Score message received from peer", "state", a.String(),
		"peer", p.ID().String(), "score", score.String())
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	return a, nil, nil
}

func (a *SyncState) Block(p peer.Peer, block *proto.Block) (State, Async, error) {
	if !p.Equal(a.conf.peerSyncWith) {
		return a, nil, nil
	}
	metrics.BlockReceivedFromExtension(block, p.Handshake().NodeName)
	a.baseInfo.logger.Debug("Block received", "state", a.String(), "peer", p.ID(),
		"blockID", block.ID.String())

	internal, err := a.internal.Block(block)
	if err != nil {
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	return a.applyBlocksWithSnapshots(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal)
}

func (a *SyncState) BlockSnapshot(
	p peer.Peer,
	blockID proto.BlockID,
	snapshot proto.BlockSnapshot,
) (State, Async, error) {
	if !p.Equal(a.conf.peerSyncWith) {
		return a, nil, nil
	}
	a.baseInfo.logger.Debug("Received snapshot for block", "state", a.String(), "peer", p.ID(),
		"blockID", blockID.String())
	internal, err := a.internal.SetSnapshot(blockID, &snapshot)
	if err != nil {
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	return a.applyBlocksWithSnapshots(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal)
}

func (a *SyncState) MinedBlock(
	block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(heightErr)
	}
	metrics.BlockMined(block)
	a.baseInfo.logger.Info("New block mined", "state", a.String(), "blockID", block.ID.String())

	_, err := a.baseInfo.blocksApplier.Apply(
		a.baseInfo.storage,
		[]*proto.Block{block},
	)
	if err != nil {
		slog.Warn("Failed to apply mined block", slog.String("state", a.String()), logging.Error(err))
		return a, nil, nil // We've failed to apply mined block, it's not an error
	}
	metrics.BlockAppliedFromExtension(block, height+1)
	metrics.Utx(a.baseInfo.utx.Count())
	a.baseInfo.scheduler.Reschedule()

	// first we should send block
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	return a, tasks.Tasks(tasks.NewMineMicroTask(defaultMicroblockInterval, block, limits, keyPair, vrf)), nil
}

func (a *SyncState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

func (a *SyncState) isTimeToSwitchPeerWithMaxScore() bool {
	now := a.baseInfo.tm.Now()
	obsolescenceTime := now.Add(-a.baseInfo.obsolescence)
	lastBlock := a.baseInfo.storage.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	return !obsolescenceTime.After(lastBlockTime)
}

func (a *SyncState) changePeerIfRequired() (peer.Peer, bool) {
	if a.isTimeToSwitchPeerWithMaxScore() {
		// Node is getting close to the top of the blockchain, it's time to switch on a node with the highest
		// score every time it updated.
		return a.baseInfo.peers.CheckPeerWithMaxScore(a.baseInfo.syncPeer.GetPeer())
	}
	// Node better continue synchronization with one node, switching to new node happens only if the larger
	// group of nodes with the highest score appears.
	return a.baseInfo.peers.CheckPeerInLargestScoreGroup(a.baseInfo.syncPeer.GetPeer())
}

// TODO suspend peer on state error
func (a *SyncState) applyBlocksWithSnapshots(
	baseInfo BaseInfo, conf conf, internal sync_internal.Internal,
) (State, Async, error) {
	internal, blocks, snapshots, eof := internal.Blocks()
	if len(blocks) == 0 {
		a.baseInfo.logger.Debug("No blocks to apply", "state", a.String())
		return newSyncState(baseInfo, conf, internal), nil, nil
	}
	height, heightErr := a.baseInfo.storage.Height()
	if heightErr != nil {
		return a, nil, a.Errorf(heightErr)
	}
	var err error
	if a.baseInfo.enableLightMode {
		err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
			var errApply error
			_, errApply = a.baseInfo.blocksApplier.ApplyWithSnapshots(s, blocks, snapshots)
			return errApply
		})
	} else {
		err = a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
			var errApply error
			_, errApply = a.baseInfo.blocksApplier.Apply(s, blocks)
			return errApply
		})
	}
	if err != nil {
		if errs.IsValidationError(err) || errs.IsValidationError(errors.Cause(err)) {
			a.baseInfo.logger.Debug("Suspending peer because of blocks application error",
				slog.String("state", a.String()),
				slog.String("peer", a.baseInfo.syncPeer.GetPeer().ID().String()), logging.Error(err))
			a.baseInfo.peers.Suspend(conf.peerSyncWith, time.Now(), err.Error())
		}
		for _, b := range blocks {
			metrics.BlockDeclinedFromExtension(b)
		}
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	for _, b := range blocks {
		metrics.BlockAppliedFromExtension(b, height+1)
		metrics.Utx(a.baseInfo.utx.Count())
		height++
	}
	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	should, err := a.baseInfo.storage.ShouldPersistAddressTransactions()
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if should {
		return newPersistState(a.baseInfo)
	}
	if eof {
		if err = a.baseInfo.storage.StartProvidingExtendedApi(); err != nil {
			return newIdleState(a.baseInfo), nil, a.Errorf(err)
		}
		return newNGState(a.baseInfo), nil, nil
	}
	if np, ok := a.changePeerIfRequired(); ok {
		a.baseInfo.logger.Debug("Changing sync peer", "state", a.String(), "peer", np.ID().String())
		return syncWithNewPeer(a, a.baseInfo, np)
	}
	a.internal.AskBlocksIDs(extension.NewPeerExtension(a.conf.peerSyncWith, a.baseInfo.scheme, a.baseInfo.netLogger))
	return newSyncState(baseInfo, conf, internal), nil, nil
}

func initSyncStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	syncSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDMicroBlockSnapshot,
		proto.ContentIDMicroBlockSnapshotRequest,
	}
	if !info.enableLightMode {
		syncSkipMessageList = append(syncSkipMessageList, proto.ContentIDBlockSnapshot)
	}
	fsm.Configure(SyncStateName).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(StartMiningEvent).
		Ignore(StopMiningEvent).
		Ignore(MicroBlockSnapshotEvent).
		OnEntry(func(_ context.Context, _ ...any) error {
			info.skipMessageList.SetList(syncSkipMessageList)
			return nil
		}).
		PermitDynamic(ChangeSyncPeerEvent,
			createPermitDynamicCallback(ChangeSyncPeerEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.ChangeSyncPeer(convertToInterface[peer.Peer](args[0]))
			})).
		PermitDynamic(StopSyncEvent,
			createPermitDynamicCallback(StopSyncEvent, state, func(_ ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.StopSync()
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(BlockEvent,
			createPermitDynamicCallback(BlockEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Block(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Block))
			})).
		PermitDynamic(BlockIDsEvent,
			createPermitDynamicCallback(BlockIDsEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.BlockIDs(convertToInterface[peer.Peer](args[0]), args[1].([]proto.BlockID))
			})).
		PermitDynamic(MinedBlockEvent,
			createPermitDynamicCallback(MinedBlockEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits),
					args[2].(proto.KeyPair), args[3].([]byte))
			})).
		PermitDynamic(TransactionEvent,
			createPermitDynamicCallback(TransactionEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Transaction(convertToInterface[peer.Peer](args[0]),
					convertToInterface[proto.Transaction](args[1]))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(_ ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Halt()
			})).
		PermitDynamic(BlockSnapshotEvent,
			createPermitDynamicCallback(BlockSnapshotEvent, state, func(args ...any) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.BlockSnapshot(
					convertToInterface[peer.Peer](args[0]),
					args[1].(proto.BlockID),
					args[2].(proto.BlockSnapshot),
				)
			}))
}

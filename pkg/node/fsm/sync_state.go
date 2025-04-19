package fsm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

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
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Changed sync peer to '%s'", p.ID().String())
	return syncWithNewPeer(a, a.baseInfo, p)
}

func (a *SyncState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.AskPeers:
		zap.S().Named(logging.FSMNamespace).Debug("[Sync] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.Ping:
		zap.S().Named(logging.FSMNamespace).Debug("[Sync] Checking timeout")
		timeout := a.conf.lastReceiveTime.Add(a.conf.timeout).Before(a.baseInfo.tm.Now())
		if timeout {
			zap.S().Named(logging.FSMNamespace).Debugf(
				"[Sync] Timed out after %s while synchronizing with peer '%s'",
				a.conf.timeout.String(), a.conf.peerSyncWith.ID())
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
		zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Empty IDs received from peer '%s'",
			peer.ID().String())
		return a, nil, nil
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Block IDs [%s...%s] received from peer %s",
		signatures[0].ShortString(), signatures[len(signatures)-1].ShortString(), peer.ID().String())
	if !peer.Equal(a.conf.peerSyncWith) {
		zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Block IDs received from incorrect peer %s, expected %s",
			peer.ID().String(), a.baseInfo.syncPeer.GetPeer().ID().String())
		return a, nil, nil
	}
	internal, err := a.internal.BlockIDs(extension.NewPeerExtension(peer, a.baseInfo.scheme), signatures)
	if err != nil {
		zap.S().Named(logging.FSMNamespace).Debugf("[Sync] No signatures expected from peer '%s' but received",
			peer.ID().String())
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	if internal.RequestedCount() > 0 {
		// Blocks were requested waiting for them to receive and apply
		zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Waiting for %d blocks to receive",
			internal.RequestedCount())
		return newSyncState(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal), nil, nil
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Continue to NG state")
	// No blocks were request, switching to NG working mode
	err = a.baseInfo.storage.StartProvidingExtendedApi()
	if err != nil {
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	return newNGState(a.baseInfo), nil, nil
}

func (a *SyncState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	metrics.FSMScore("sync", score, p.Handshake().NodeName)
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Score message received from peer '%s', score %s",
		p.ID().String(), score.String())
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	return a, nil, nil
}

func (a *SyncState) Block(p peer.Peer, block *proto.Block) (State, Async, error) {
	if !p.Equal(a.conf.peerSyncWith) {
		return a, nil, nil
	}
	metrics.FSMKeyBlockReceived("sync", block, p.Handshake().NodeName)
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync][%s] Received block %s", p.ID(), block.ID.String())

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
	zap.S().Named(logging.FSMNamespace).Debugf("[Sync][%s] Received snapshot for block %s", p.ID(), blockID.String())
	internal, err := a.internal.SetSnapshot(blockID, &snapshot)
	if err != nil {
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	return a.applyBlocksWithSnapshots(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal)
}

func (a *SyncState) MinedBlock(
	block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte,
) (State, Async, error) {
	metrics.FSMKeyBlockGenerated("sync", block)
	zap.S().Named(logging.FSMNamespace).Infof("[Sync] New block '%s' mined", block.ID.String())
	_, err := a.baseInfo.blocksApplier.Apply(
		a.baseInfo.storage,
		[]*proto.Block{block},
	)
	if err != nil {
		zap.S().Warnf("[Sync] Failed to apply mined block: %v", err)
		return a, nil, nil // We've failed to apply mined block, it's not an error
	}
	metrics.FSMKeyBlockApplied("sync", block)
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
		zap.S().Named(logging.FSMNamespace).Debug("[Sync] No blocks to apply")
		return newSyncState(baseInfo, conf, internal), nil, nil
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
			zap.S().Named(logging.FSMNamespace).Debugf(
				"[Sync] Suspending peer '%s' because of blocks application error: %v",
				a.baseInfo.syncPeer.GetPeer().ID().String(), err)
			a.baseInfo.peers.Suspend(conf.peerSyncWith, time.Now(), err.Error())
		}
		for _, b := range blocks {
			metrics.FSMKeyBlockDeclined("sync", b, err)
		}
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	for _, b := range blocks {
		metrics.FSMKeyBlockApplied("sync", b)
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
		zap.S().Named(logging.FSMNamespace).Debugf("[Sync] Changing sync peer to '%s'", np.ID().String())
		return syncWithNewPeer(a, a.baseInfo, np)
	}
	a.internal.AskBlocksIDs(extension.NewPeerExtension(a.conf.peerSyncWith, a.baseInfo.scheme))
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
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			info.skipMessageList.SetList(syncSkipMessageList)
			return nil
		}).
		PermitDynamic(ChangeSyncPeerEvent,
			createPermitDynamicCallback(ChangeSyncPeerEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.ChangeSyncPeer(convertToInterface[peer.Peer](args[0]))
			})).
		PermitDynamic(StopSyncEvent,
			createPermitDynamicCallback(StopSyncEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.StopSync()
			})).
		PermitDynamic(TaskEvent,
			createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Task(args[0].(tasks.AsyncTask))
			})).
		PermitDynamic(ScoreEvent,
			createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
			})).
		PermitDynamic(BlockEvent,
			createPermitDynamicCallback(BlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Block(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Block))
			})).
		PermitDynamic(BlockIDsEvent,
			createPermitDynamicCallback(BlockIDsEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.BlockIDs(convertToInterface[peer.Peer](args[0]), args[1].([]proto.BlockID))
			})).
		PermitDynamic(MinedBlockEvent,
			createPermitDynamicCallback(MinedBlockEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits),
					args[2].(proto.KeyPair), args[3].([]byte))
			})).
		PermitDynamic(TransactionEvent,
			createPermitDynamicCallback(TransactionEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Transaction(convertToInterface[peer.Peer](args[0]),
					convertToInterface[proto.Transaction](args[1]))
			})).
		PermitDynamic(HaltEvent,
			createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
				a, ok := state.State.(*SyncState)
				if !ok {
					return a, nil, a.Errorf(errors.Errorf(
						"unexpected type '%T' expected '*SyncState'", state.State))
				}
				return a.Halt()
			})).
		PermitDynamic(BlockSnapshotEvent,
			createPermitDynamicCallback(BlockSnapshotEvent, state, func(args ...interface{}) (State, Async, error) {
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

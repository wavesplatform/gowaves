package state_fsm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	syncSkipMessageList = proto.PeerMessageIDs{
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
	}
)

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

type noopWrapper struct{}

func (noopWrapper) AskBlocksIDs([]proto.BlockID) {}

func (noopWrapper) AskBlock(proto.BlockID) {}

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

func (a *SyncState) DisconnectedPeer(p peer.Peer) (State, Async, error) {
	if a.conf.peerSyncWith == p {
		_, blocks, _, _ := a.internal.Blocks(noopWrapper{}, nil)
		if len(blocks) > 0 {
			err := a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
				_, err := a.baseInfo.blocksApplier.Apply(s, blocks)
				return err
			})
			return newIdleState(a.baseInfo), nil, a.Errorf(err)
		}
		return newIdleState(a.baseInfo), nil, nil
	}
	return a, nil, nil
}

func (a *SyncState) ConnectedBestPeer(p peer.Peer) (State, Async, error) {
	if p != a.conf.peerSyncWith {
		return syncWithNewPeer(a, a.baseInfo, p)
	}
	return a, nil, nil
}

func (a *SyncState) Task(task tasks.AsyncTask) (State, Async, error) {
	switch task.TaskType {
	case tasks.AskPeers:
		zap.S().Debug("[Sync] Requesting peers")
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case tasks.Ping:
		zap.S().Debug("[Sync] Checking timeout")
		timeout := a.conf.lastReceiveTime.Add(a.conf.timeout).Before(a.baseInfo.tm.Now())
		if timeout {
			zap.S().Debugf("[Sync] Timeout (%s) while syncronisation with peer '%s'", a.conf.timeout.String(), a.conf.peerSyncWith.ID())
			return newIdleState(a.baseInfo), nil, a.Errorf(TimeoutErr)
		}
		return a, nil, nil
	case tasks.MineMicro:
		return a, nil, nil
	default:
		return a, nil, a.Errorf(errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s State", task.TaskType, task.Data, a.String()))
	}
}

func (a *SyncState) BlockIDs(peer peer.Peer, signatures []proto.BlockID) (State, Async, error) {
	if a.conf.peerSyncWith != peer {
		return a, nil, nil
	}
	internal, err := a.internal.BlockIDs(extension.NewPeerExtension(peer, a.baseInfo.scheme), signatures)
	if err != nil {
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	if internal.RequestedCount() > 0 {
		// Blocks were requested waiting for them to receive and apply
		return newSyncState(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal), nil, nil
	}
	// No blocks were request, switching to NG working mode
	err = a.baseInfo.storage.StartProvidingExtendedApi()
	if err != nil {
		return newIdleState(a.baseInfo), nil, a.Errorf(err)
	}
	return newNGState(a.baseInfo), nil, nil
}

func (a *SyncState) Score(p peer.Peer, score *proto.Score) (State, Async, error) {
	metrics.FSMScore("sync", score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	return a, nil, nil
}

func (a *SyncState) Block(p peer.Peer, block *proto.Block) (State, Async, error) {
	if p != a.conf.peerSyncWith {
		return a, nil, nil
	}
	metrics.FSMKeyBlockReceived("sync", block, p.Handshake().NodeName)
	zap.S().Debugf("[Sync][%s] Received block %s", p.ID(), block.ID.String())
	internal, err := a.internal.Block(block)
	if err != nil {
		return newSyncState(a.baseInfo, a.conf, internal), nil, a.Errorf(err)
	}
	return a.applyBlocks(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal)
}

func (a *SyncState) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (State, Async, error) {
	metrics.FSMKeyBlockGenerated("sync", block)
	zap.S().Infof("New key block '%s' mined", block.ID.String())
	_, err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		return a, nil, nil // We've failed to apply mined block, it's not an error
	}
	metrics.FSMKeyBlockApplied("sync", block)
	a.baseInfo.Reschedule()

	// first we should send block
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	return a, tasks.Tasks(tasks.NewMineMicroTask(5*time.Second, block, limits, keyPair, vrf)), nil
}

func (a *SyncState) Halt() (State, Async, error) {
	return newHaltState(a.baseInfo)
}

func (a *SyncState) getPeerWithMaxScore() (peer.Peer, error) {
	maxScorePeer, err := a.baseInfo.peers.GetPeerWithMaxScore()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get peer with max score")
	}
	maxScore, err := a.baseInfo.peers.Score(maxScorePeer)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get score of peer '%s'", maxScorePeer.ID())
	}

	syncWithPeer := a.conf.peerSyncWith
	peerSyncWithScore, err := a.baseInfo.peers.Score(syncWithPeer)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get score of peer synced with '%s'", maxScorePeer.ID())
	}

	if maxScorePeer != syncWithPeer && maxScore == peerSyncWithScore {
		return syncWithPeer, nil
	}
	return maxScorePeer, nil
}

func (a *SyncState) changePeerSyncWith() (State, Async, error) {
	peer, err := a.getPeerWithMaxScore()
	if err != nil {
		return newIdleState(a.baseInfo), nil, a.Errorf(errors.Wrapf(err, "Failed to change peer for sync"))
	}
	return syncWithNewPeer(a, a.baseInfo, peer)
}

// TODO suspend peer on state error
func (a *SyncState) applyBlocks(baseInfo BaseInfo, conf conf, internal sync_internal.Internal) (State, Async, error) {
	internal, blocks, eof, needToChangePeer := internal.Blocks(
		extension.NewPeerExtension(a.conf.peerSyncWith, a.baseInfo.scheme),
		func() bool {
			peer, err := a.getPeerWithMaxScore()
			return err == nil && peer != a.conf.peerSyncWith
		},
	)
	if needToChangePeer {
		return a.changePeerSyncWith()
	}
	if len(blocks) == 0 {
		return newSyncState(baseInfo, conf, internal), nil, nil
	}
	err := a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		var err error
		_, err = a.baseInfo.blocksApplier.Apply(s, blocks)
		return err
	})
	if err != nil {
		if errs.IsValidationError(err) || errs.IsValidationError(errors.Cause(err)) {
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
	a.baseInfo.Reschedule()
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	should, err := a.baseInfo.storage.ShouldPersistAddressTransactions()
	if err != nil {
		return a, nil, a.Errorf(err)
	}
	if should {
		return newPersistState(a.baseInfo)
	}
	if eof {
		err := a.baseInfo.storage.StartProvidingExtendedApi()
		if err != nil {
			return newIdleState(a.baseInfo), nil, a.Errorf(err)
		}
		return newNGState(a.baseInfo), nil, nil
	}
	return newSyncState(baseInfo, conf, internal), nil, nil
}

func initSyncStateInFSM(state *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	fsm.Configure(SyncStateName).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(ConnectedPeerEvent).
		Ignore(StopMiningEvent).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			info.skipMessageList.SetList(syncSkipMessageList)
			return nil
		}).
		PermitDynamic(ConnectedBestPeerEvent, createPermitDynamicCallback(ConnectedBestPeerEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.ConnectedBestPeer(convertToInterface[peer.Peer](args[0]))
		})).
		PermitDynamic(DisconnectedPeerEvent, createPermitDynamicCallback(DisconnectedPeerEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.DisconnectedPeer(convertToInterface[peer.Peer](args[0]))
		})).
		PermitDynamic(TaskEvent, createPermitDynamicCallback(TaskEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.Task(args[0].(tasks.AsyncTask))
		})).
		PermitDynamic(ScoreEvent, createPermitDynamicCallback(ScoreEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.Score(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Score))
		})).
		PermitDynamic(BlockEvent, createPermitDynamicCallback(BlockEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.Block(convertToInterface[peer.Peer](args[0]), args[1].(*proto.Block))
		})).
		PermitDynamic(BlockIDsEvent, createPermitDynamicCallback(BlockIDsEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.BlockIDs(convertToInterface[peer.Peer](args[0]), args[1].([]proto.BlockID))
		})).
		PermitDynamic(MinedBlockEvent, createPermitDynamicCallback(MinedBlockEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.MinedBlock(args[0].(*proto.Block), args[1].(proto.MiningLimits), args[2].(proto.KeyPair), args[3].([]byte))
		})).
		PermitDynamic(TransactionEvent, createPermitDynamicCallback(TransactionEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.Transaction(convertToInterface[peer.Peer](args[0]), convertToInterface[proto.Transaction](args[1]))
		})).
		PermitDynamic(HaltEvent, createPermitDynamicCallback(HaltEvent, state, func(args ...interface{}) (State, Async, error) {
			a := state.State.(*SyncState)
			return a.Halt()
		}))
}

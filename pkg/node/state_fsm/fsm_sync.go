package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
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

type noopWrapper struct {
}

func (noopWrapper) AskBlocksIDs([]proto.BlockID) {
}

func (noopWrapper) AskBlock(proto.BlockID) {
}

type SyncFsm struct {
	baseInfo BaseInfo
	conf     conf
	internal sync_internal.Internal
}

func (a *SyncFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.baseInfo.utx.Add(t)
	if err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	a.baseInfo.BroadcastTransaction(t, p)
	return a, nil, nil
}

// MicroBlock ignores new microblocks while syncing.
func (a *SyncFsm) MicroBlock(_ peer.Peer, _ *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

// MicroBlockInv ignores microblock requests while syncing.
func (a *SyncFsm) MicroBlockInv(_ peer.Peer, _ *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *SyncFsm) Task(task tasks.AsyncTask) (FSM, Async, error) {
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
			return NewIdleFsm(a.baseInfo), nil, TimeoutErr
		}
		return a, nil, nil
	case tasks.MineMicro:
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("unexpected internal task '%d' with data '%+v' received by %s FSM", task.TaskType, task.Data, a.String())
	}
}

func (a *SyncFsm) PeerError(p peer.Peer, _ error) (FSM, Async, error) {
	a.baseInfo.peers.Disconnect(p)
	if a.conf.peerSyncWith == p {
		_, blocks, _ := a.internal.Blocks(noopWrapper{})
		if len(blocks) > 0 {
			err := a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
				_, err := a.baseInfo.blocksApplier.Apply(s, blocks)
				return err
			})
			return NewIdleFsm(a.baseInfo), nil, err
		}
	}
	return a, nil, nil
}

func (a *SyncFsm) BlockIDs(peer peer.Peer, signatures []proto.BlockID) (FSM, Async, error) {
	if a.conf.peerSyncWith != peer {
		return a, nil, nil
	}
	internal, err := a.internal.BlockIDs(extension.NewPeerExtension(peer, a.baseInfo.scheme), signatures)
	if err != nil {
		return newSyncFsm(a.baseInfo, a.conf, internal), nil, err
	}
	if internal.RequestedCount() > 0 {
		// Blocks were requested waiting for them to receive and apply
		return newSyncFsm(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal), nil, nil
	}
	// No blocks were request, switching to NG working mode
	err = a.baseInfo.storage.StartProvidingExtendedApi()
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	return NewNGFsm12(a.baseInfo), nil, nil
}

func (a *SyncFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	err := a.baseInfo.peers.NewConnection(p)
	if err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	return a, nil, nil
}

func (a *SyncFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	metrics.FSMScore("sync", score, p.Handshake().NodeName)
	if err := a.baseInfo.peers.UpdateScore(p, score); err != nil {
		return a, nil, proto.NewInfoMsg(err)
	}
	//TODO: Handle new higher score
	/*
		nodeScore, err := a.baseInfo.storage.CurrentScore()
		if err != nil {
			return a, nil, err
		}
		if score.Cmp(nodeScore) == 1 {
			lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(a.baseInfo.storage)
			if err != nil {
				return a, nil, err
			}
			internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, a.baseInfo.scheme), lastSignatures)
			c := conf{
				peerSyncWith: p,
				timeout:      30 * time.Second,
			}
			zap.S().Debugf("[Sync] Higher score received, starting synchronisation with peer '%s'", p.ID())
			return NewSyncFsm(a.baseInfo, c.Now(), internal)
		}
	*/
	return noop(a)
}

func (a *SyncFsm) Block(p peer.Peer, block *proto.Block) (FSM, Async, error) {
	if p != a.conf.peerSyncWith {
		return a, nil, nil
	}
	metrics.FSMKeyBlockReceived("sync", block, p.Handshake().NodeName)
	zap.S().Debugf("[Sync][%s] Received block %s", p.ID(), block.ID.String())
	internal, err := a.internal.Block(block)
	if err != nil {
		return newSyncFsm(a.baseInfo, a.conf, internal), nil, err
	}
	return a.applyBlocks(a.baseInfo, a.conf.Now(a.baseInfo.tm), internal)
}

func (a *SyncFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
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

func (a *SyncFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.baseInfo)
}

// TODO suspend peer on state error
func (a *SyncFsm) applyBlocks(baseInfo BaseInfo, conf conf, internal sync_internal.Internal) (FSM, Async, error) {
	internal, blocks, eof := internal.Blocks(extension.NewPeerExtension(conf.peerSyncWith, a.baseInfo.scheme))
	if len(blocks) == 0 {
		return newSyncFsm(baseInfo, conf, internal), nil, nil
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
		return NewIdleFsm(a.baseInfo), nil, err
	}
	for _, b := range blocks {
		metrics.FSMKeyBlockApplied("sync", b)
	}
	a.baseInfo.Reschedule()
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	should, err := a.baseInfo.storage.ShouldPersistAddressTransactions()
	if err != nil {
		return a, nil, err
	}
	if should {
		return NewPersistTransition(a.baseInfo)
	}
	if eof {
		err := a.baseInfo.storage.StartProvidingExtendedApi()
		if err != nil {
			return NewIdleFsm(a.baseInfo), nil, err
		}
		return NewNGFsm12(a.baseInfo), nil, nil
	}
	return newSyncFsm(baseInfo, conf, internal), nil, nil
}

func (a *SyncFsm) String() string {
	return "Sync"
}

func NewSyncFsm(baseInfo BaseInfo, conf conf, internal sync_internal.Internal) (FSM, Async, error) {
	return newSyncFsm(baseInfo, conf, internal), nil, nil
}

func newSyncFsm(baseInfo BaseInfo, conf conf, internal sync_internal.Internal) FSM {
	return &SyncFsm{
		baseInfo: baseInfo,
		conf:     conf,
		internal: internal,
	}
}

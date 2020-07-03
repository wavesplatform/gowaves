package state_fsm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type conf struct {
	peerSyncWith Peer
	// if nothing happens more than N duration, means we stalled, so go to idle and again
	lastReceiveTime time.Time

	timeout time.Duration
}

func (c conf) Now() conf {
	return conf{
		peerSyncWith:    c.peerSyncWith,
		lastReceiveTime: time.Now(),
		timeout:         c.timeout,
	}
}

type SyncFsm struct {
	baseInfo BaseInfo
	conf     conf
	internal sync_internal.Internal
}

func (a *SyncFsm) Transaction(p Peer, t proto.Transaction) (FSM, Async, error) {
	err := a.baseInfo.utx.Add(t)
	if err != nil {
		a.baseInfo.BroadcastTransaction(t, p)
	}
	return a, nil, err
}

// ignore microblocks
func (a *SyncFsm) MicroBlock(_ Peer, _ *proto.MicroBlock) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

// ignore microblocks
func (a *SyncFsm) MicroBlockInv(_ Peer, _ *proto.MicroBlockInv) (FSM, Async, error) {
	return a.baseInfo.d.Noop(a)
}

func (a *SyncFsm) Task(task AsyncTask) (FSM, Async, error) {
	zap.S().Debugf("SyncFsm Task: got task type %d, data %+v", task.TaskType, task.Data)
	switch task.TaskType {
	case ASK_PEERS:
		a.baseInfo.peers.AskPeers()
		return a, nil, nil
	case PING:
		timeout := a.conf.lastReceiveTime.Add(a.conf.timeout).Before(a.baseInfo.tm.Now())
		if timeout {
			return NewIdleFsm(a.baseInfo), nil, TimeoutErr
		}
		return a, nil, nil
	default:
		return a, nil, errors.Errorf("SyncFsm Task: unknown task type %d, data %+v", task.TaskType, task.Data)
	}
}

type noopWrapper struct {
}

func (noopWrapper) AskBlocksIDs([]proto.BlockID) {
}

func (noopWrapper) AskBlock(proto.BlockID) {
}

func (a *SyncFsm) PeerError(p Peer, e error) (FSM, Async, error) {
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
	return NewIdleFsm(a.baseInfo), nil, nil
}

func (a *SyncFsm) BlockIDs(peer Peer, sigs []proto.BlockID) (FSM, Async, error) {
	if a.conf.peerSyncWith != peer {
		return a, nil, nil
	}
	internal, err := a.internal.BlockIDs(extension.NewPeerExtension(peer, a.baseInfo.scheme), sigs)
	if err != nil {
		return newSyncFsm(a.baseInfo, a.conf, internal), nil, err
	}
	if internal.RequestedCount() > 0 {
		// Blocks were requested waiting for them to receive and apply
		return newSyncFsm(a.baseInfo, a.conf.Now(), internal), nil, nil
	}
	// No blocks were request, switching to NG working mode
	err = a.baseInfo.storage.StartProvidingExtendedApi()
	if err != nil {
		return NewIdleFsm(a.baseInfo), nil, err
	}
	return NewNGFsm12(a.baseInfo), nil, nil
}

func (a *SyncFsm) NewPeer(p Peer) (FSM, Async, error) {
	err := a.baseInfo.peers.NewConnection(p)
	return a, nil, err
}

func (a *SyncFsm) Score(p Peer, score *proto.Score) (FSM, Async, error) {
	// TODO handle new max score
	err := a.baseInfo.peers.UpdateScore(p, score)
	if err != nil {
		return a, nil, err
	}
	return a, nil, nil
}

func (a *SyncFsm) Block(p Peer, block *proto.Block) (FSM, Async, error) {
	if p != a.conf.peerSyncWith {
		return a, nil, nil
	}
	metrics.BlockReceivedFromExtension(block, p.Handshake().NodeName)
	zap.S().Debugf("[%s] Received block %s", p.ID(), block.ID.String())
	internal, err := a.internal.Block(block)
	if err != nil {
		return newSyncFsm(a.baseInfo, a.conf, internal), nil, err
	}
	return a.applyBlocks(a.baseInfo, a.conf.Now(), internal)
}

func (a *SyncFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	zap.S().Infof("New key block '%s' mined", block.ID.String())
	h, err := a.baseInfo.blocksApplier.Apply(a.baseInfo.storage, []*proto.Block{block})
	if err != nil {
		return a, nil, nil // We've failed to apply mined block, it's not an error
	}
	metrics.BlockMined(block, h)
	a.baseInfo.Reschedule()

	// first we should send block
	a.baseInfo.actions.SendBlock(block)
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	return a, Tasks(NewMineMicroTask(5*time.Second, block, limits, keyPair, vrf)), nil
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
	var last proto.Height
	err := a.baseInfo.storage.Map(func(s state.NonThreadSafeState) error {
		var err error
		last, err = a.baseInfo.blocksApplier.Apply(s, blocks)
		return err
	})
	if err != nil {
		for _, b := range blocks {
			metrics.BlockDeclinedFromExtension(b)
		}
		return NewIdleFsm(a.baseInfo), nil, err
	}
	for i, b := range blocks {
		metrics.BlockAppliedFromExtension(b, last-uint64(len(blocks)-1-i))
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

func NewIdleToSyncTransition(baseInfo BaseInfo, p Peer) (FSM, Async, error) {
	lastSigs, err := signatures.LastSignaturesImpl{}.LastBlockIDs(baseInfo.storage)
	if err != nil {
		return NewIdleFsm(baseInfo), nil, err
	}
	internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, baseInfo.scheme), lastSigs)
	c := conf{
		peerSyncWith: p,
		timeout:      30 * time.Second,
	}
	return NewSyncFsm(baseInfo, c.Now(), internal)
}

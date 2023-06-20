package state_fsm

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"

	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
	. "github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Async []Task

type BlocksApplier interface {
	BlockExists(state storage.State, block *proto.Block) (bool, error)
	Apply(state storage.State, block []*proto.Block) (proto.Height, error)
	ApplyMicro(state storage.State, block *proto.Block) (proto.Height, error)
}

type BaseInfo struct {
	// peers
	peers peer_manager.PeerManager

	// storage
	storage storage.State

	// ntp time
	tm types.Time

	scheme        proto.Scheme
	invRequester  InvRequester
	blocksApplier BlocksApplier

	// scheduler
	types.Scheduler

	microMiner         *miner.MicroMiner
	MicroBlockCache    services.MicroBlockCache
	MicroBlockInvCache services.MicroBlockInvCache
	microblockInterval time.Duration

	actions Actions

	utx types.UtxPool

	minPeersMining int

	skipMessageList *messages.SkipMessageList
}

func (a *BaseInfo) BroadcastTransaction(t proto.Transaction, receivedFrom peer.Peer) {
	a.peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		if p != receivedFrom {
			_ = extension.NewPeerExtension(p, a.scheme).SendTransaction(t)
		}
	})
}

func (a *BaseInfo) CleanUtx() {
	utxpool.NewCleaner(a.storage, a.utx, a.tm).Clean()
}

// States
const (
	IdleStateName    = "Idle"
	NGStateName      = "NG"
	PersistStateName = "Persist"
	SyncStateName    = "Sync"
	HaltStateName    = "Halt"
)

// Events
const (
	NewPeerEvent       = "NewPeer"
	PeerErrorEvent     = "PeerError"
	ScoreEvent         = "Score"
	BlockEvent         = "Block"
	MinedBlockEvent    = "MinedBlock"
	BlockIDsEvent      = "BlockIDs"
	TaskEvent          = "Task"
	MicroBlockEvent    = "MicroBlock"
	MicroBlockInvEvent = "MicroBlockInv"
	TransactionEvent   = "Transaction"
	HaltEvent          = "Halt"

	DisconnectedPeerEvent  = "DisconnectedPeer"
	StopMiningEvent        = "StopMining"
	ConnectedPeerEvent     = "ConnectedPeer"
	ConnectedBestPeerEvent = "ConnectedBestPeer"
)

type FSM struct {
	fsm      *stateless.StateMachine
	baseInfo BaseInfo
	State    *StateData
}

type State interface {
	String() string
	Errorf(error) error
}

type StateData struct {
	Name  stateless.State
	State State
}

func NewFSM(services services.Services, microblockInterval time.Duration) (*FSM, Async, error) {
	if microblockInterval <= 0 {
		return nil, nil, errors.New("microblock interval must be positive")
	}
	info := BaseInfo{
		peers:   services.Peers,
		storage: services.State,
		tm:      services.Time,
		scheme:  services.Scheme,

		//
		invRequester:  ng.NewInvRequester(),
		blocksApplier: services.BlocksApplier,

		Scheduler: services.Scheduler,

		microMiner: miner.NewMicroMiner(services),

		MicroBlockCache:    services.MicroBlockCache,
		MicroBlockInvCache: microblock_cache.NewMicroblockInvCache(),
		microblockInterval: microblockInterval,

		actions: &ActionsImpl{services: services},

		utx: services.UtxPool,

		minPeersMining: services.MinPeersMining,

		skipMessageList: services.SkipMessageList,
	}

	info.Scheduler.Reschedule()

	state := &StateData{
		Name:  IdleStateName,
		State: newIdleState(info),
	}

	// default tasks
	t := Async{
		// ask about peers for every 5 minutes
		NewAskPeersTask(askPeersInterval),
		NewPingTask(),
	}
	fsm := stateless.NewStateMachineWithExternalStorage(func(_ context.Context) (stateless.State, error) {
		return state.Name, nil
	}, func(_ context.Context, s stateless.State) error {
		state.Name = s
		return nil
	}, stateless.FiringQueued)

	initIdleStateInFSM(state, fsm, info)
	initHaltStateInFSM(state, fsm, info)
	initNGStateInFSM(state, fsm, info)
	initPersistStateInFSM(state, fsm, info)
	initSyncStateInFSM(state, fsm, info)

	return &FSM{
		fsm:      fsm,
		baseInfo: info,
		State:    state,
	}, t, nil
}

func (f *FSM) NewPeer(p peer.Peer) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(NewPeerEvent, asyncRes, p)
	return *asyncRes, err
}

func (f *FSM) PeerError(p peer.Peer, e error) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(PeerErrorEvent, asyncRes, p, e)
	return *asyncRes, err
}

func (f *FSM) Score(p peer.Peer, score *proto.Score) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(ScoreEvent, asyncRes, p, score)
	return *asyncRes, err
}

func (f *FSM) Task(task AsyncTask) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(TaskEvent, asyncRes, task)
	return *asyncRes, err
}

func (f *FSM) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(MinedBlockEvent, asyncRes, block, limits, keyPair, vrf)
	return *asyncRes, err
}

func (f *FSM) Block(p peer.Peer, block *proto.Block) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(BlockEvent, asyncRes, p, block)
	return *asyncRes, err
}

// BlockIDs receives signatures that was requested by GetSignatures
func (f *FSM) BlockIDs(peer peer.Peer, signatures []proto.BlockID) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(BlockIDsEvent, asyncRes, peer, signatures)
	return *asyncRes, err
}

func (f *FSM) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(MicroBlockEvent, asyncRes, p, micro)
	return *asyncRes, err
}

func (f *FSM) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(MicroBlockInvEvent, asyncRes, p, inv)
	return *asyncRes, err
}

func (f *FSM) Transaction(p peer.Peer, t proto.Transaction) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(TransactionEvent, asyncRes, p, t)
	return *asyncRes, err
}

func (f *FSM) Halt() (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(HaltEvent, asyncRes)
	return *asyncRes, err
}

func (f *FSM) DisconnectedPeer(p peer.Peer) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(DisconnectedPeerEvent, asyncRes, p)
	return *asyncRes, err
}

func (f *FSM) StopMining() (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(StopMiningEvent, asyncRes)
	return *asyncRes, err
}

func (f *FSM) ConnectedPeer(p peer.Peer) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(ConnectedPeerEvent, asyncRes, p)
	return *asyncRes, err
}

func (f *FSM) ConnectedBestPeer(p peer.Peer) (Async, error) {
	asyncRes := &Async{}
	err := f.fsm.Fire(ConnectedBestPeerEvent, asyncRes, p)
	return *asyncRes, err
}

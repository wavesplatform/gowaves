package state_fsm

import (
	"errors"
	"time"

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

	// default behaviour
	d Default

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

type FromBaseInfo interface {
	FromBaseInfo(b BaseInfo) FSM
}

type FSM interface {
	NewPeer(p peer.Peer) (FSM, Async, error)
	PeerError(p peer.Peer, e error) (FSM, Async, error)
	Score(p peer.Peer, score *proto.Score) (FSM, Async, error)
	Block(p peer.Peer, block *proto.Block) (FSM, Async, error)
	MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error)

	// BlockIDs receives signatures that was requested by GetSignatures
	BlockIDs(peer.Peer, []proto.BlockID) (FSM, Async, error)
	Task(task AsyncTask) (FSM, Async, error)

	MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error)
	MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error)

	Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error)

	Halt() (FSM, Async, error)

	String() string

	Errorf(err error) error
}

func NewFsm(services services.Services, microblockInterval time.Duration) (FSM, Async, error) {
	if microblockInterval <= 0 {
		return nil, nil, errors.New("microblock interval must be positive")
	}
	b := BaseInfo{
		peers:   services.Peers,
		storage: services.State,
		tm:      services.Time,
		scheme:  services.Scheme,

		//
		invRequester:  ng.NewInvRequester(),
		blocksApplier: services.BlocksApplier,

		// TODO: need better way
		d: DefaultImpl{},

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

	b.Scheduler.Reschedule()

	// default tasks
	tasks := Async{
		// ask about peers for every 5 minutes
		NewAskPeersTask(askPeersInterval),
		NewPingTask(),
	}

	return NewIdleFsm(b), tasks, nil
}

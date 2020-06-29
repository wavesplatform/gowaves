package state_fsm

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
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
	Apply(state storage.State, block []*proto.Block) error
}

type BaseInfo struct {
	// peers
	peers peer_manager.PeerManager

	// storage
	storage storage.State

	// ntp time
	tm types.Time

	// outdate period
	outdatePeriod proto.Timestamp

	scheme proto.Scheme

	//
	invRequester  InvRequester
	blocksApplier BlocksApplier

	// default behaviour
	d Default

	// scheduler
	types.Scheduler

	microMiner *miner.MicroMiner

	MicroBlockCache    services.MicroBlockCache
	MicroBlockInvCache services.MicroBlockInvCache

	actions Actions

	utx types.UtxPool

	minPeersMining int
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

	// Received signatures after asking by GetSignatures
	BlockIDs(peer.Peer, []proto.BlockID) (FSM, Async, error)
	Task(task AsyncTask) (FSM, Async, error)

	// micro
	MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error)
	MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error)

	Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error)

	//
	Halt() (FSM, Async, error)
}

func NewFsm(
	services services.Services,
	outdatePeriod proto.Timestamp,
) (FSM, Async, error) {
	b := BaseInfo{
		peers:         services.Peers,
		storage:       services.State,
		tm:            services.Time,
		outdatePeriod: outdatePeriod,
		scheme:        services.Scheme,

		//
		invRequester:  ng.NewInvRequester(),
		blocksApplier: services.BlocksApplier,

		// TODO: need better way
		d: DefaultImpl{},

		Scheduler: services.Scheduler,

		microMiner: miner.NewMicroMiner(services),

		MicroBlockCache:    services.MicroBlockCache,
		MicroBlockInvCache: microblock_cache.NewMicroblockInvCache(),

		actions: &ActionsImpl{services: services},

		utx: services.UtxPool,

		minPeersMining: services.MinPeersMining,
	}

	b.Scheduler.Reschedule()

	// default tasks
	tasks := Async{
		// ask about peers for every 5 minutes
		NewAskPeersTask(5 * time.Minute),
		NewPingTask(),
	}

	return NewIdleFsm(b), tasks, nil
}

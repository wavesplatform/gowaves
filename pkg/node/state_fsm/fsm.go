package state_fsm

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// TODO send score

const (
	PING = iota + 1
	ASK_PEERS

	SYNC_WAIT_SIGNATURES_TIMEOUT
	SYNC_WAIT_BLOCK_TIMEOUT
)

type BlocksApplier interface {
	Apply(state storage.State, block []*proto.Block) error
}

type TaskType int

type Async []Task

type AsyncTask struct {
	TaskType int
	Data     interface{}
}

type Task interface {
	Run(ctx context.Context, output chan AsyncTask) error
}

type BaseInfo struct {
	// too old state
	//outdated bool

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
	blockCreater  types.BlockCreater
	blocksApplier BlocksApplier

	// default behaviour
	d Default
}

type FSM interface {
	NewPeer(p peer.Peer) (FSM, Async, error)
	PeerError(peer.Peer, error) (FSM, Async, error)
	Score(p peer.Peer, score *proto.Score) (FSM, Async, error)
	Block(peer peer.Peer, block *proto.Block) (FSM, Async, error)

	// Received signatures after asking by GetSignatures
	Signatures(peer peer.Peer, sigs []crypto.Signature) (FSM, Async, error)
	//GetPeers(peer peer.Peer) (FSM, Async, error)
	Task(task AsyncTask) (FSM, Async, error)

	// micro
	MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error)
	MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error)
}

func NewFsm(
	s storage.State,
	peers peer_manager.PeerManager,
	tm types.Time,
	outdatePeriod proto.Timestamp,
	scheme proto.Scheme,
	invRequester InvRequester,
	blockCreater types.BlockCreater,
	blocksApplier BlocksApplier,

) (FSM, Async, error) {
	b := BaseInfo{
		peers:         peers,
		storage:       s,
		tm:            tm,
		outdatePeriod: outdatePeriod,
		scheme:        scheme,

		//
		invRequester:  invRequester,
		blockCreater:  blockCreater,
		blocksApplier: blocksApplier,

		//
		d: DefaultImpl{},
	}

	// default tasks
	tasks := Async{
		// ask about peers for every 5 minutes
		NewAskPeersTask(5 * time.Minute),
	}

	return NewIdleFsm(b), tasks, nil
}

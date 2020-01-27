package scoresender

import (
	"context"
	"math/big"
	"time"

	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
	"go.uber.org/zap"
)

// state interface contains subset of State
type state interface {
	CurrentScore() (*big.Int, error)
	Mutex() *lock.RwMutex
}

// eachConnected interface contains subset of PeerManager
type eachConnected interface {
	EachConnected(func(peer peer.Peer, score *proto.Score))
}

// sender contains all logic about sending current node score to other peers.
// its made like service in intention not to spam other nodes frequently
type sender struct {
	ch       chan struct{}
	peers    eachConnected
	state    state
	duration time.Duration
	runner   runner.Runner
}

// New creates new sender instance
func New(peers eachConnected, state state, duration time.Duration, runner runner.Runner) *sender {
	return &sender{
		ch:       make(chan struct{}, 1),
		peers:    peers,
		state:    state,
		duration: duration,
		runner:   runner,
	}
}

// Run implements non priority sending
func (a *sender) Run(ctx context.Context) {
	tick := time.NewTicker(a.duration)
	update := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.ch:
			update = true
		case <-tick.C:
			if update {
				a.sendScore()
			}
			update = false
		}
	}
}

// NonPriority method used when node synchronization in progress.
// In such case node score changes very often, but such information cah be send
// only once in some duration.
func (a *sender) NonPriority() {
	select {
	case a.ch <- struct{}{}:
	default:
	}
}

// Priority we need send score asap. For example in miner, or maybe some other cases.
func (a *sender) Priority() {
	a.runner.Go(a.sendScore)
}

// sendScore get and send score to peers.
func (a *sender) sendScore() {
	locked := a.state.Mutex().RLock()
	curScore, err := a.state.CurrentScore()
	locked.Unlock()
	if err != nil {
		zap.S().Debugf("ScoreSender: %q", err)
		return
	}
	a.peers.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(&proto.ScoreMessage{Score: curScore.Bytes()})
	})
}

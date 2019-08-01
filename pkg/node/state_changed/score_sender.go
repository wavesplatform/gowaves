package state_changed

import (
	"math/big"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type eachConnected interface {
	EachConnected(func(peer.Peer, *proto.Score))
}

// Sends ScoreMessage to all connected peers on score change
type ScoreSender struct {
	peers     eachConnected
	state     state.State
	prevScore *proto.Score
	mu        sync.Mutex
}

func NewScoreSender(peers eachConnected, state state.State) *ScoreSender {
	return &ScoreSender{peers: peers, state: state, prevScore: big.NewInt(0)}
}

func (a *ScoreSender) Handle() {
	a.mu.Lock()
	prevScore := *a.prevScore
	a.mu.Unlock()
	curScore, err := a.state.CurrentScore()
	if err != nil {
		zap.S().Error(err)
		return
	}
	// same score
	if prevScore.Cmp(curScore) == 0 {
		return
	}
	a.mu.Lock()
	a.prevScore = curScore
	a.mu.Unlock()
	a.peers.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(&proto.ScoreMessage{
			Score: curScore.Bytes(),
		})
	})
}

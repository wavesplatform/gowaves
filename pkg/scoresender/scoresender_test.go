package scoresender

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

type peers struct {
	peer peer.Peer
}

func (a *peers) EachConnected(f func(peer peer.Peer, score *proto.Score)) {
	f(a.peer, nil)
}

type state_ struct {
	score *big.Int
	mu    *sync.RWMutex
}

func (a *state_) CurrentScore() (*big.Int, error) {
	return a.score, nil
}

func (a *state_) Mutex() *lock.RwMutex {
	return lock.NewRwMutex(a.mu)
}

func TestSender_Priority(t *testing.T) {
	peer := mock.NewPeer()
	s := &state_{
		score: big.NewInt(100500),
		mu:    &sync.RWMutex{},
	}

	sender := New(&peers{
		peer: peer,
	}, s, 100*time.Millisecond, runner.NewSync())

	sender.Priority()

	require.Equal(t, &proto.ScoreMessage{
		Score: big.NewInt(100500).Bytes(),
	}, peer.SendMessageCalledWith[0])

}

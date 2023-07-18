package peers

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type mockPeerID struct {
	id string
}

func (pid *mockPeerID) String() string {
	return pid.id
}

func (pid *mockPeerID) ID() peerID {
	return peerID(pid.id)
}

func TestScoreSelectorPushOnce(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	score100 := big.NewInt(100)
	ss.push(peer1, score100)

	g, ok := ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1}, g.peers)
	assert.Equal(t, score100, g.score)
	sk, ok := ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
}

func TestScoreSelectorPushTwice(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	score100 := big.NewInt(100)

	ss.push(peer1, score100)
	g, ok := ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1}, g.peers)
	assert.Equal(t, score100, g.score)
	sk, ok := ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)

	ss.push(peer1, score100)
	g, ok = ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1}, g.peers)
	assert.Equal(t, score100, g.score)
	sk, ok = ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
}

func TestMultiplePushOneScore(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	peer2 := &mockPeerID{"peer2"}
	peer3 := &mockPeerID{"peer3"}
	score100 := big.NewInt(100)

	ss.push(peer1, score100)
	ss.push(peer2, score100)
	ss.push(peer3, score100)
	g, ok := ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1, peer2, peer3}, g.peers)
	assert.Equal(t, score100, g.score)
	sk, ok := ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
	sk, ok = ss.peerToScoreKey[peer2.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
	sk, ok = ss.peerToScoreKey[peer3.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)

	ss.push(peer2, score100)
	g, ok = ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1, peer2, peer3}, g.peers)
	assert.Equal(t, score100, g.score)
	sk, ok = ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
	sk, ok = ss.peerToScoreKey[peer2.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
	sk, ok = ss.peerToScoreKey[peer3.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
}

func TestNewScore(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	peer2 := &mockPeerID{"peer2"}
	peer3 := &mockPeerID{"peer3"}
	score100 := big.NewInt(100)
	score200 := big.NewInt(200)

	ss.push(peer1, score100)
	ss.push(peer2, score100)
	ss.push(peer3, score100)

	ss.push(peer2, score200)
	g, ok := ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer1, peer3}, g.peers)
	assert.Equal(t, score100, g.score)
	g, ok = ss.scoreKeyToGroup["200"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer2}, g.peers)
	assert.Equal(t, score200, g.score)

	sk, ok := ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)
	sk, ok = ss.peerToScoreKey[peer2.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
	sk, ok = ss.peerToScoreKey[peer3.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)

	ss.push(peer1, score200)
	g, ok = ss.scoreKeyToGroup["100"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer3}, g.peers)
	assert.Equal(t, score100, g.score)
	g, ok = ss.scoreKeyToGroup["200"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer2, peer1}, g.peers)
	assert.Equal(t, score200, g.score)

	sk, ok = ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
	sk, ok = ss.peerToScoreKey[peer2.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
	sk, ok = ss.peerToScoreKey[peer3.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("100"), sk)

	ss.push(peer3, score200)
	_, ok = ss.scoreKeyToGroup["100"]
	assert.False(t, ok)
	g, ok = ss.scoreKeyToGroup["200"]
	assert.True(t, ok)
	assert.Equal(t, []peer.ID{peer2, peer1, peer3}, g.peers)
	assert.Equal(t, score200, g.score)

	sk, ok = ss.peerToScoreKey[peer1.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
	sk, ok = ss.peerToScoreKey[peer2.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
	sk, ok = ss.peerToScoreKey[peer3.ID()]
	assert.True(t, ok)
	assert.Equal(t, scoreKey("200"), sk)
}

func TestSelectionSinglePeer(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	score100 := big.NewInt(100)
	ss.push(peer1, score100)
	best, score := ss.selectBestPeer(nil)
	require.NotNil(t, best)
	assert.Equal(t, peer1, best)
	require.NotNil(t, score)
	assert.Equal(t, score, score100)
	best, score = ss.selectBestPeer(best)
	require.NotNil(t, best)
	assert.Equal(t, peer1, best)
	require.NotNil(t, score)
	assert.Equal(t, score, score100)

	score200 := big.NewInt(200)
	ss.push(peer1, score200)
	best, score = ss.selectBestPeer(best)
	require.NotNil(t, best)
	assert.Equal(t, peer1, best)
	require.NotNil(t, score)
	assert.Equal(t, score, score200)
}

func TestSelectionMultiplePeers(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	peer2 := &mockPeerID{"peer2"}
	peer3 := &mockPeerID{"peer3"}
	score100 := big.NewInt(100)
	score200 := big.NewInt(200)

	ss.push(peer1, score100)
	ss.push(peer2, score100)
	ss.push(peer3, score100)

	best1, score1 := ss.selectBestPeer(nil)
	require.NotNil(t, best1)
	assert.True(t, best1 == peer1 || best1 == peer2 || best1 == peer3)
	require.NotNil(t, score1)
	assert.Equal(t, score1, score100)

	best2, score2 := ss.selectBestPeer(best1)
	require.NotNil(t, best2)
	assert.Equal(t, best1, best2)
	require.NotNil(t, score2)
	assert.Equal(t, score2, score1)

	ss.push(peer1, score200)
	best3, score3 := ss.selectBestPeer(best2)
	require.NotNil(t, best3)
	assert.True(t, best3 == peer2 || best3 == peer3)
	require.NotNil(t, score3)
	assert.Equal(t, score3, score2)

	ss.push(peer3, score200)
	best4, score4 := ss.selectBestPeer(best3)
	require.NotNil(t, best4)
	assert.True(t, best4 == peer1 || best4 == peer3)
	require.NotNil(t, score4)
	assert.Equal(t, score4, score200)

	ss.push(peer2, score200)
	best5, score5 := ss.selectBestPeer(best4)
	require.NotNil(t, best5)
	assert.Equal(t, best4, best5)
	require.NotNil(t, score5)
	assert.Equal(t, score5, score4)
}

func TestPushDeleteMultiplyTimes(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	peer2 := &mockPeerID{"peer2"}
	score100 := big.NewInt(100)
	score200 := big.NewInt(200)

	ss.push(peer1, score100)
	ss.push(peer2, score100)
	assert.Equal(t, 2, len(ss.peerToScoreKey))
	ss.push(peer1, score100)
	ss.push(peer2, score100)
	assert.Equal(t, 2, len(ss.peerToScoreKey))

	ss.delete(peer1)
	ss.delete(peer2)
	assert.Equal(t, 0, len(ss.peerToScoreKey))
	ss.delete(peer1)
	ss.delete(peer2)
	assert.Equal(t, 0, len(ss.peerToScoreKey))

	ss.push(peer1, score100)
	ss.push(peer2, score200)
	assert.Equal(t, 2, len(ss.peerToScoreKey))
	ss.push(peer1, score100)
	ss.push(peer2, score200)
	assert.Equal(t, 2, len(ss.peerToScoreKey))
}

// BenchmarkPush100 results (Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz)
// BenchmarkPush100-12           	   76863	     13848 ns/op.
func BenchmarkPush100(b *testing.B) {
	peers := make([]peer.ID, 100)
	scores := make([]*proto.Score, 100)
	for i := 0; i < 100; i++ {
		peers[i] = &mockPeerID{id: fmt.Sprintf("peer%d", i)}
		scores[i] = big.NewInt(int64(i/10) + 100)
	}
	ss := newScoreSelector()
	for i := 0; i < b.N; i++ {
		for i, p := range peers {
			ss.push(p, scores[i])
		}
	}
}

// BenchmarkSelect100 results (Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz)
// BenchmarkSelect100-12         	  146244	      8447 ns/op.
func BenchmarkSelect100(b *testing.B) {
	peers := make([]peer.ID, 100)
	scores := make([]*proto.Score, 100)
	ss := newScoreSelector()
	for i := 0; i < 100; i++ {
		peers[i] = &mockPeerID{id: fmt.Sprintf("peer%d", i)}
		scores[i] = big.NewInt(int64(i/10) + 100)
		ss.push(peers[i], scores[i])
	}
	for i := 0; i < b.N; i++ {
		for _, p := range peers {
			bp, s := ss.selectBestPeer(p)
			_ = bp
			_ = s
		}
	}
}

// BenchmarkPushSelect1000 results (Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz)
// BenchmarkPushSelect1000-12    	    1785	    672910 ns/op.
func BenchmarkPushSelect1000(b *testing.B) {
	peers := make([]peer.ID, 1000)
	scores := make([]*proto.Score, 1000)
	for i := 0; i < 1000; i++ {
		peers[i] = &mockPeerID{id: fmt.Sprintf("peer%d", i)}
		scores[i] = big.NewInt(int64(i/100) + 1000)
	}
	ss := newScoreSelector()
	for i := 0; i < b.N; i++ {
		for i, p := range peers {
			ss.push(p, scores[i])
			bp, s := ss.selectBestPeer(p)
			_ = bp
			_ = s
		}
	}
}

package peer_manager

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"net"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
)

func genFakeAddr() string {
	buf := make([]byte, 4)
	ip := rand.Uint32()
	binary.LittleEndian.PutUint32(buf, ip)
	return net.IP(buf).String()
}

func genPeers(n uint32) []*mock.Peer {
	dupMap := make(map[string]struct{})
	out := make([]*mock.Peer, n)

	for i := uint32(0); i < n; i++ {
		for {
			addr := genFakeAddr()
			if _, ok := dupMap[addr]; !ok {
				dupMap[addr] = struct{}{}
				out[i] = &mock.Peer{Addr: addr}
				break
			}
		}
	}
	return out
}

func validateActivePeersInternal(t *testing.T, active *activePeers) {
	require.Equalf(t, len(active.m), len(active.sortedByScore), "m len (%d) != sortedByScore len (%d)", len(active.m), len(active.sortedByScore))
	require.True(
		t,
		sort.SliceIsSorted(
			active.sortedByScore,
			func(i, j int) bool {
				return active.m[active.sortedByScore[i]].score.Cmp(active.m[active.sortedByScore[j]].score) == 1
			},
		),
		"sortedByScore isn't in descending order",
	)
}

func TestAdd(t *testing.T) {
	active := newActivePeers()
	peersSlice := genPeers(60)

	maxScorePeer := peersSlice[0]
	for i, p := range peersSlice {
		active.add(p)
		validateActivePeersInternal(t, &active)

		info, ok := active.getPeerWithMaxScore()
		require.Truef(t, ok, "%d", i)
		require.Equalf(t, maxScorePeer, info.peer, "%d", i)

		err := active.updateScore(p.ID(), big.NewInt(int64(i)))
		require.NoError(t, err, "%d", i)
		validateActivePeersInternal(t, &active)

		info, ok = active.getPeerWithMaxScore()
		require.Truef(t, ok, "%d", i)
		require.Equalf(t, p, info.peer, "%d", i)

		maxScorePeer = p
	}
}

func TestRemoveByScore(t *testing.T) {
	active := newActivePeers()
	peersSlice := genPeers(60)

	for _, p := range peersSlice {
		active.add(p)
		require.NoError(t, active.updateScore(p.ID(), big.NewInt(int64(rand.Uint32()))))
	}

	removeCheck := func(t *testing.T, i int) {
		peerID := active.m[active.sortedByScore[i]].peer.ID()
		active.remove(active.m[active.sortedByScore[i]].peer.ID())
		_, ok := active.get(peerID)
		require.False(t, ok, "removed peer still exists")
		validateActivePeersInternal(t, &active)
	}
	// remove peer with the highest score
	removeCheck(t, 0)

	// remove peer with the median score
	removeCheck(t, active.size()/2)

	// remove peer with the lowest score
	removeCheck(t, active.size()-1)
}

func TestStableInsertion(t *testing.T) {
	active := newActivePeers()

	_, ok := active.getPeerWithMaxScore()
	assert.False(t, ok)

	peer1 := &mock.Peer{
		Addr: "127.0.0.1",
	}

	peer2 := &mock.Peer{
		Addr: "192.186.0.0",
	}

	active.add(peer1)
	active.add(peer2)

	info, ok := active.getPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer1, info.peer)

	err := active.updateScore(peer2.ID(), big.NewInt(100))
	assert.NoError(t, err)
	info, ok = active.getPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer2, info.peer)

	err = active.updateScore(peer1.ID(), big.NewInt(100))
	assert.NoError(t, err)
	info, ok = active.getPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer2, info.peer)
}

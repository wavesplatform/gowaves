package peer_manager

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
)

func TestBasic(t *testing.T) {
	active := NewActivePeers()

	_, ok := active.GetPeerWithMaxScore()
	assert.False(t, ok)

	peer1 := &mock.Peer{
		Addr: "127.0.0.1",
	}

	peer2 := &mock.Peer{
		Addr: "192.186.0.0",
	}

	active.Add(peer1)
	active.Add(peer2)

	info, ok := active.GetPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer1, info.peer)

	err := active.UpdateScore(peer2.ID(), big.NewInt(100))
	assert.NoError(t, err)
	info, ok = active.GetPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer2, info.peer)

	err = active.UpdateScore(peer1.ID(), big.NewInt(100))
	assert.NoError(t, err)
	info, ok = active.GetPeerWithMaxScore()
	assert.True(t, ok)
	assert.Equal(t, peer2, info.peer)
}

package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/node/peers"
)

func TestMinerConsensus(t *testing.T) {
	m := peers.NewMockPeerManager(t)

	m.EXPECT().ConnectedCount().Return(1).Once()
	a := NewMinerConsensus(m, 1)
	assert.True(t, a.IsMiningAllowed())

	m.EXPECT().ConnectedCount().Return(0).Once()
	a = NewMinerConsensus(m, 1)
	assert.False(t, a.IsMiningAllowed())
}

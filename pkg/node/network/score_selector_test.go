package network

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPeerID struct {
	id string
}

func (pid *mockPeerID) String() string {
	return pid.id
}

func TestSelection(t *testing.T) {
	ss := newScoreSelector()
	peer1 := &mockPeerID{"peer1"}
	score100 := big.NewInt(100)
	ss.push(peer1, score100)
	best, score := ss.pop()
	require.NotNil(t, best)
	assert.Equal(t, peer1, best)
	require.NotNil(t, score)
	assert.Equal(t, 100, score.Int64())
}

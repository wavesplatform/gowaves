package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
)

func TestAddr2Peers(t *testing.T) {
	addr := "127.0.0.1"
	p := NewAddr2Peers()
	require.NotNil(t, p)
	assert.False(t, p.Exists(addr))

	p.Add(addr, &mock.Peer{})
	assert.True(t, p.Exists(addr))

	assert.Equal(t, 1, len(p.Addresses()))
}

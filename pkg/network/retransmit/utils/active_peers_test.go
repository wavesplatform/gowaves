package utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddr2Peers(t *testing.T) {
	addr := "127.0.0.1"
	p := NewAddr2Peers()
	require.NotNil(t, p)
	assert.False(t, p.Exists(addr))

	p.Add(addr, &PeerInfo{})
	assert.True(t, p.Exists(addr))

	assert.Equal(t, 1, len(p.Addresses()))
}

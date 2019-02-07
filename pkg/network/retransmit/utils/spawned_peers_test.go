package utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSpawnedPeers(t *testing.T) {
	addr := "127.0.0.1"
	p := NewSpawnedPeers()
	require.NotNil(t, p)
	assert.Equal(t, 0, len(p.GetAll()))
	assert.False(t, p.Exists(addr))

	p.Add(addr)
	assert.Equal(t, 1, len(p.GetAll()))
	assert.True(t, p.Exists(addr))

	p.Delete(addr)
	assert.Equal(t, 0, len(p.GetAll()))
	assert.False(t, p.Exists(addr))
}

package proto

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeyPair(t *testing.T) {
	k := NewKeyPair([]byte("test"))
	require.NotEmpty(t, k.Public())
	require.NotEmpty(t, k.Private())
}

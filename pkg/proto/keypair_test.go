package proto

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeyPair(t *testing.T) {
	k := NewKeyPair([]byte("test"))
	pub, err := k.Public()
	require.NoError(t, err)
	require.NotEmpty(t, pub)
	priv, err := k.Public()
	require.NoError(t, err)
	require.NotEmpty(t, priv)
}

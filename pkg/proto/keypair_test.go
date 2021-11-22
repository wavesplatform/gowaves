package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyPair(t *testing.T) {
	k := MustKeyPair([]byte("test"))
	pub := k.Public
	require.NotEmpty(t, pub)
	sk := k.Secret
	require.NotEmpty(t, sk)
}

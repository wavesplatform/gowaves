package nullable

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestSignature(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("4xK251JFM824zi9pmBRQixXp5BEjPKZuBQESSQU4f5379rwebwBfnstZzc4bPvzkLvy3nf2F98oLmRTX2H88oB3f")
	s := NewSignature(sig)
	require.False(t, s.Null())
	require.Equal(t, s.Sig(), sig)

	s = NewNullSignature()
	require.True(t, s.Null())
}

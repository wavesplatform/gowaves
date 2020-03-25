package nullable

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSignature(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("4xK251JFM824zi9pmBRQixXp5BEjPKZuBQESSQU4f5379rwebwBfnstZzc4bPvzkLvy3nf2F98oLmRTX2H88oB3f")
	id := proto.NewBlockIDFromSignature(sig)
	s := NewBlockID(id)
	require.False(t, s.Null())
	require.Equal(t, s.ID(), id)

	s = NewNullBlockID()
	require.True(t, s.Null())
}

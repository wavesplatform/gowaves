package node

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"testing"
)

func TestExpectedBlocks(t *testing.T) {

	s1, _ := crypto.NewSignatureFromBase58("22gRwjusnFYDoS31hRFEpFq21FjPCca2bUYtwicUH41GwzVkEAv7G22pAbRisu5s3bbhpzRRUpwF5png6ooKkb1n")
	s2, _ := crypto.NewSignatureFromBase58("3YzRwee4k7ddfXK9FtMtZs9V4r8sxThVLUAF6ATfz1Efrxv29CjoHnw2oCz8uvjFhgPMgrsKMmgSyVZ3nw5Hswme")

	ch := make(chan blockBytes, 2)
	e := newExpectedBlocks([]crypto.Signature{s1, s2}, ch)

	require.True(t, e.hasNext())

	// first we add second bytes
	require.NoError(t, e.add(s2.Bytes()))
	require.True(t, e.hasNext())

	select {
	case <-ch:
		t.Fatal("received unexpected block")
	default:
	}

	// then add first
	require.NoError(t, e.add(s1.Bytes()))
	require.False(t, e.hasNext(), "we received all expected messages, no more should arrive")

	select {
	case rs := <-ch:
		require.Equal(t, s1.Bytes(), []byte(rs))
	default:
		t.Fatal("no block")
	}

	select {
	case rs := <-ch:
		require.Equal(t, s2.Bytes(), []byte(rs))
	default:
		t.Fatal("no block")
	}
}

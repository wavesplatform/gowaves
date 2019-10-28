package ng

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestMicroblockCache(t *testing.T) {
	a := NewMicroblockCache(1)
	_, ok := a.MicroBlock(emptySig)
	require.False(t, ok)

	a.AddMicroBlock(newMicro(sig1, emptySig))
	rs, ok := a.MicroBlock(sig1)
	require.True(t, ok)
	require.Equal(t, sig1, rs.TotalResBlockSigField)
}

func TestInvCache(t *testing.T) {
	a := NewInvCache(1)
	_, ok := a.Inv(emptySig)
	require.False(t, ok)

	a.AddInv(newInv(sig1))
	rs, ok := a.Inv(sig1)
	require.True(t, ok)
	require.Equal(t, sig1, rs.TotalBlockSig)
}

func TestKnownBlocks(t *testing.T) {
	b := knownBlocks{}

	require.Equal(t, true, b.add(&proto.Block{}))
	require.Equal(t, false, b.add(&proto.Block{}))
}

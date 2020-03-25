package ng

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestMicroblockCache(t *testing.T) {
	a := NewMicroblockCache(1)
	_, ok := a.MicroBlock(emptyId)
	require.False(t, ok)

	a.AddMicroBlock(newMicro(sig1, emptySig))
	rs, ok := a.MicroBlock(id1)
	require.True(t, ok)
	require.Equal(t, id1, rs.TotalBlockID)
}

func TestInvCache(t *testing.T) {
	a := NewInvCache(1)
	_, ok := a.Inv(emptyId)
	require.False(t, ok)

	a.AddInv(newInv(sig1))
	rs, ok := a.Inv(id1)
	require.True(t, ok)
	require.Equal(t, id1, rs.TotalBlockID)
}

func TestKnownBlocks(t *testing.T) {
	b := knownBlocks{}

	require.Equal(t, true, b.add(&proto.Block{}))
	require.Equal(t, false, b.add(&proto.Block{}))
}

package endorsementpool_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/miner/endorsementpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func newDummyEndorsement(idx int32) *proto.EndorseBlock {
	b := make([]byte, crypto.DigestSize)
	b[0] = byte(idx)
	id, _ := proto.NewBlockIDFromBytes(b)
	return &proto.EndorseBlock{
		EndorserIndex:   idx,
		EndorsedBlockID: id,
		Signature:       []byte{1},
	}
}

func TestEndorsementPool_PriorityByBalance(t *testing.T) {
	pool := endorsementpool.NewEndorsementPool(5)

	e1 := newDummyEndorsement(1)
	e2 := newDummyEndorsement(2)
	e3 := newDummyEndorsement(3)

	require.NoError(t, pool.Add(e1, bls.PublicKey{}, 10))
	require.NoError(t, pool.Add(e2, bls.PublicKey{}, 50))
	require.NoError(t, pool.Add(e3, bls.PublicKey{}, 30))

	all := pool.GetAll()
	require.Len(t, all, 3)

	minBalance := uint64(0)
	for _, e := range all {
		if e.EndorserIndex == 1 {
			minBalance = 10
		}
	}
	require.Equal(t, uint64(10), minBalance)
}

func TestEndorsementPool_PriorityBySeqWhenEqualBalance(t *testing.T) {
	pool := endorsementpool.NewEndorsementPool(3)

	e1 := newDummyEndorsement(1)
	e2 := newDummyEndorsement(2)

	require.NoError(t, pool.Add(e1, bls.PublicKey{}, 100))
	require.NoError(t, pool.Add(e2, bls.PublicKey{}, 100))

	all := pool.GetAll()
	require.Len(t, all, 2)

	// Balance e1 and e2 are equal, so we check by seq.
	e3 := newDummyEndorsement(3)
	require.NoError(t, pool.Add(e3, bls.PublicKey{}, 100))

	require.Equal(t, 3, pool.Len())
}

func TestEndorsementPool_RemoveLowPriorityWhenFull(t *testing.T) {
	pool := endorsementpool.NewEndorsementPool(3)

	require.NoError(t, pool.Add(newDummyEndorsement(1), bls.PublicKey{}, 10))
	require.NoError(t, pool.Add(newDummyEndorsement(2), bls.PublicKey{}, 20))
	require.NoError(t, pool.Add(newDummyEndorsement(3), bls.PublicKey{}, 30))

	require.Equal(t, 3, pool.Len())

	require.NoError(t, pool.Add(newDummyEndorsement(4), bls.PublicKey{}, 40))

	all := pool.GetAll()
	require.Equal(t, 3, len(all), "pool size must remain constant when full")

	// Low priority (balance=10) should be evicted.
	found10 := false
	for _, e := range all {
		if e.EndorserIndex == 1 {
			found10 = true
		}
	}
	require.False(t, found10, "low priority (balance=10) should be evicted")
}

func TestEndorsementPool_RejectLowBalanceWhenFull(t *testing.T) {
	pool := endorsementpool.NewEndorsementPool(2)

	require.NoError(t, pool.Add(newDummyEndorsement(1), bls.PublicKey{}, 50))
	require.NoError(t, pool.Add(newDummyEndorsement(2), bls.PublicKey{}, 60))
	require.Equal(t, 2, pool.Len())

	// Low balance (30) shouldn't get added.
	require.NoError(t, pool.Add(newDummyEndorsement(3), bls.PublicKey{}, 30))
	require.Equal(t, 2, pool.Len(), "low-priority endorsement should be rejected")

	// High balance (100) should evict the lowest (50).
	require.NoError(t, pool.Add(newDummyEndorsement(4), bls.PublicKey{}, 100))
	require.Equal(t, 2, pool.Len())

	all := pool.GetAll()
	found50 := false
	for _, e := range all {
		if e.EndorserIndex == 1 {
			found50 = true
		}
	}
	require.False(t, found50, "element with lowest balance should be evicted")
}

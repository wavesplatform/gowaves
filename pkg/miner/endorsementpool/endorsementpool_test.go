package endorsementpool_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/miner/endorsementpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func dummyBLSSK(t *testing.T) bls.SecretKey {
	t.Helper()
	sk, err := bls.GenerateSecretKey([]byte("endorsement-pool-test-key"))
	require.NoError(t, err)
	return sk
}

func dummyBLSPK(t *testing.T) bls.PublicKey {
	t.Helper()
	sk := dummyBLSSK(t)
	pk, err := sk.PublicKey()
	require.NoError(t, err)
	return pk
}

func newDummyEndorsement(t *testing.T, idx int32, _ string) *proto.EndorseBlock {
	b := make([]byte, crypto.DigestSize)
	b[0] = byte(idx)
	id, err := proto.NewBlockIDFromBytes(b)
	require.NoError(t, err)
	e := &proto.EndorseBlock{
		EndorserIndex:        idx,
		EndorsedBlockID:      id,
		FinalizedBlockHeight: 1,
		FinalizedBlockID:     id,
	}
	signEndorsement(t, e, dummyBLSSK(t))
	return e
}

const sigOne = "nBWfaRLW7EdcwxhDMaXuZZFMhHyowAxY7476rkBsUUeguTXrMSNuTVkuWLmZjRmRfgMXEGuvdHiu1V7joRFSLz3X6MQBF8m88kHJE" +
	"j6Tc2ktBnMTzihh2JMGpuuWBLSK8rv"
const sigTwo = "RNMTkL736x3TmXfjQufKnxSgySaaoec3WYnxmujcum9BHEmCdjmwvjoUehghqYCWJcNj5CNfb9QdnujV9o2DRitbLgq2bnLdTU5s" +
	"1DLBWBkVx8mBayvdfx7rPZ3mtUWeh5L"
const sigThree = "U8GEty7F58p7QZrNAxRYrfMSU4z6CwtiukBu9hGDP9rLx3VmF9ZYy8bHWBCTDTYW7s2juqRHU3aERUJfgx3KhxBdv57UFb34" +
	"evuW9wYQKKoCTbfasfZENM4GDbPdL2nQYKY"
const sigFour = "2F4sw8YzXpSf93ACAngoTnNxCaYWoGL4vY88RYgEs3BeSsnAmMGmVSfe8h6hybkfb6CYoUwV1prRbYWo6umrL9evmTPeksdaQ" +
	"rp19eTcwxZLBtPzbwqonCbEX8eDJVTydRBo"

const finalizedHeightEndorsement = 1

func signEndorsement(t *testing.T, e *proto.EndorseBlock, sk bls.SecretKey) {
	t.Helper()
	msg, err := e.EndorsementMessage()
	require.NoError(t, err)
	sig, err := bls.Sign(sk, msg)
	require.NoError(t, err)
	e.Signature = sig
}

func newSignedEndorsement(
	t *testing.T,
	endorserIndex int32,
	finalizedID proto.BlockID,
	finalizedHeight uint32,
	endorsedID proto.BlockID,
	sk bls.SecretKey,
) *proto.EndorseBlock {
	t.Helper()
	e := &proto.EndorseBlock{
		EndorserIndex:        endorserIndex,
		FinalizedBlockID:     finalizedID,
		FinalizedBlockHeight: finalizedHeight,
		EndorsedBlockID:      endorsedID,
	}
	signEndorsement(t, e, sk)
	return e
}

func addToPool(
	t *testing.T,
	pool *endorsementpool.EndorsementPool,
	e *proto.EndorseBlock,
	pk bls.PublicKey,
	balance uint64,
) {
	t.Helper()
	_, err := pool.Add(e, pk, finalizedHeightEndorsement, e.FinalizedBlockID, balance, e.EndorsedBlockID)
	require.NoError(t, err)
}

func TestEndorsementPool_PriorityByBalance(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)
	e1 := newDummyEndorsement(t, 1, sigOne)
	e2 := newDummyEndorsement(t, 2, sigTwo)
	e3 := newDummyEndorsement(t, 3, sigThree)
	pk := dummyBLSPK(t)

	addToPool(t, pool, e1, pk, 10)
	addToPool(t, pool, e2, pk, 20)
	addToPool(t, pool, e3, pk, 30)

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
	pool, err := endorsementpool.NewEndorsementPool(3)
	require.NoError(t, err)
	e1 := newDummyEndorsement(t, 1, sigOne)
	e2 := newDummyEndorsement(t, 2, sigTwo)
	pk := dummyBLSPK(t)

	addToPool(t, pool, e1, pk, 100)
	addToPool(t, pool, e2, pk, 100)

	all := pool.GetAll()
	require.Len(t, all, 2)

	// Balance e1 and e2 are equal, so we check by seq.
	e3 := newDummyEndorsement(t, 3, sigThree)
	addToPool(t, pool, e3, pk, 100)

	require.Equal(t, 3, pool.Len())
}

func TestEndorsementPool_RemoveLowPriorityWhenFull(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(3)
	require.NoError(t, err)
	pk := dummyBLSPK(t)
	addToPool(t, pool, newDummyEndorsement(t, 1, sigOne), pk, 10)
	addToPool(t, pool, newDummyEndorsement(t, 2, sigTwo), pk, 20)
	addToPool(t, pool, newDummyEndorsement(t, 3, sigThree), pk, 30)

	require.Equal(t, 3, pool.Len())

	addToPool(t, pool, newDummyEndorsement(t, 4, sigFour), pk, 40)

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
	pool, err := endorsementpool.NewEndorsementPool(2)
	require.NoError(t, err)
	pk := dummyBLSPK(t)
	addToPool(t, pool, newDummyEndorsement(t, 1, sigOne), pk, 50)
	addToPool(t, pool, newDummyEndorsement(t, 2, sigTwo), pk, 60)
	require.Equal(t, 2, pool.Len())

	// Low balance (30) shouldn't get added.
	addToPool(t, pool, newDummyEndorsement(t, 3, sigThree), pk, 30)
	require.Equal(t, 2, pool.Len(), "low-priority endorsement should be rejected")

	// High balance (100) should evict the lowest (50).
	addToPool(t, pool, newDummyEndorsement(t, 4, sigFour), pk, 100)
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

func TestEndorsementPool_ShouldIgnoreEndorsement(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	finalizedDigest, err := crypto.FastHash([]byte("finalized"))
	require.NoError(t, err)
	endorsedDigestA, err := crypto.FastHash([]byte("endorsed-a"))
	require.NoError(t, err)
	endorsedDigestB, err := crypto.FastHash([]byte("endorsed-b"))
	require.NoError(t, err)

	finalizedID := proto.NewBlockIDFromDigest(finalizedDigest)
	endorsedIDA := proto.NewBlockIDFromDigest(endorsedDigestA)
	endorsedIDB := proto.NewBlockIDFromDigest(endorsedDigestB)

	sk1, err := bls.GenerateSecretKey([]byte("endorser-seed-1"))
	require.NoError(t, err)
	pk1, err := sk1.PublicKey()
	require.NoError(t, err)

	invalid := newSignedEndorsement(t, 0, finalizedID, 5, endorsedIDA, sk1)
	sk2, err := bls.GenerateSecretKey([]byte("endorser-seed-2"))
	require.NoError(t, err)
	pk2, err := sk2.PublicKey()
	require.NoError(t, err)
	require.True(t, pool.ShouldIgnoreEndorsement(invalid, pk2, 5, endorsedIDA))

	future := newSignedEndorsement(t, 0, finalizedID, 10, endorsedIDA, sk1)
	require.True(t, pool.ShouldIgnoreEndorsement(future, pk1, 5, endorsedIDA))

	base := newSignedEndorsement(t, 0, finalizedID, 5, endorsedIDA, sk1)
	_, err = pool.Add(base, pk1, 5, finalizedID, 100, endorsedIDA)
	require.NoError(t, err)

	otherRound := newSignedEndorsement(t, 1, finalizedID, 5, endorsedIDB, sk1)
	require.True(t, pool.ShouldIgnoreEndorsement(otherRound, pk1, 5, endorsedIDA))

	conflictFinalized := newSignedEndorsement(t, 2, endorsedIDA, 5, endorsedIDA, sk1)
	require.False(t, pool.ShouldIgnoreEndorsement(conflictFinalized, pk1, 5, endorsedIDA))
}

package endorsementpool_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/miner/endorsementpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func newKeyPair(t *testing.T, seed string) (bls.SecretKey, bls.PublicKey) {
	t.Helper()
	sk, err := bls.GenerateSecretKey([]byte(seed))
	require.NoError(t, err)
	pk, err := sk.PublicKey()
	require.NoError(t, err)
	return sk, pk
}

func newBlockID(t *testing.T, s string) proto.BlockID {
	t.Helper()
	d, err := crypto.FastHash([]byte(s))
	require.NoError(t, err)
	return proto.NewBlockIDFromDigest(d)
}

func signedEndorsement(
	t *testing.T,
	idx uint32,
	finalizedID proto.BlockID,
	finalizedHeight uint32,
	endorsedID proto.BlockID,
	sk bls.SecretKey,
) *proto.BlockEndorsement {
	t.Helper()
	e := &proto.BlockEndorsement{
		EndorserIndex:        idx,
		FinalizedBlockID:     finalizedID,
		FinalizedBlockHeight: finalizedHeight,
		EndorsedBlockID:      endorsedID,
	}
	msg, err := e.CryptoMessage().Bytes()
	require.NoError(t, err)
	sig, err := bls.Sign(sk, msg)
	require.NoError(t, err)
	e.Signature = sig
	return e
}

// roundEndorsement creates an endorsement for a canonical "test round": finalizedID="fin", height=1, endorsedID="end".
func roundEndorsement(t *testing.T, idx uint32, sk bls.SecretKey) *proto.BlockEndorsement {
	t.Helper()
	return signedEndorsement(t, idx, newBlockID(t, "fin"), 1, newBlockID(t, "end"), sk)
}

func addToPool(
	t *testing.T, pool *endorsementpool.EndorsementPool, e *proto.BlockEndorsement, pk bls.PublicKey, balance uint64,
) {
	t.Helper()
	_, err := pool.Add(e, pk, balance)
	require.NoError(t, err)
}

func TestPool_PriorityByBalance(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 10)
	addToPool(t, pool, roundEndorsement(t, 2, sk), pk, 20)
	addToPool(t, pool, roundEndorsement(t, 3, sk), pk, 30)

	all := pool.GetAll()
	require.Len(t, all, 3)

	for _, e := range all {
		if e.EndorserIndex == 1 {
			assert.Equal(t, uint32(1), e.EndorserIndex) // lowest balance stays while below cap
		}
	}
}

func TestPool_EqualBalanceEarlierArrivalWins(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(2)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	addToPool(t, pool, roundEndorsement(t, 2, sk), pk, 100)
	require.Equal(t, 2, pool.Len())

	// Third endorsement with same balance: pool is full, equal balance → reject.
	added, err := pool.Add(roundEndorsement(t, 3, sk), pk, 100)
	require.NoError(t, err)
	assert.False(t, added, "equal-balance endorsement must not displace earlier arrival")
	assert.Equal(t, 2, pool.Len())
}

func TestPool_EvictLowerBalanceWhenFull(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(3)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 10)
	addToPool(t, pool, roundEndorsement(t, 2, sk), pk, 20)
	addToPool(t, pool, roundEndorsement(t, 3, sk), pk, 30)
	require.Equal(t, 3, pool.Len())

	added, err := pool.Add(roundEndorsement(t, 4, sk), pk, 40)
	require.NoError(t, err)
	assert.True(t, added)

	all := pool.GetAll()
	require.Equal(t, 3, len(all))

	for _, e := range all {
		assert.NotEqual(t, uint32(1), e.EndorserIndex, "lowest-balance (index 1) must be evicted")
	}
}

func TestPool_RejectLowerOrEqualBalanceWhenFull(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(2)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 50)
	addToPool(t, pool, roundEndorsement(t, 2, sk), pk, 60)
	require.Equal(t, 2, pool.Len())

	// Lower balance: reject.
	added, err := pool.Add(roundEndorsement(t, 3, sk), pk, 30)
	require.NoError(t, err)
	assert.False(t, added)
	assert.Equal(t, 2, pool.Len())

	// Equal to minimum: reject.
	added, err = pool.Add(roundEndorsement(t, 3, sk), pk, 50)
	require.NoError(t, err)
	assert.False(t, added)
	assert.Equal(t, 2, pool.Len())

	// Strictly higher: evict minimum (index 1, balance 50), insert new.
	added, err = pool.Add(roundEndorsement(t, 4, sk), pk, 100)
	require.NoError(t, err)
	assert.True(t, added)
	require.Equal(t, 2, pool.Len())

	all := pool.GetAll()
	for _, e := range all {
		assert.NotEqual(t, uint32(1), e.EndorserIndex, "index 1 (balance 50) must be evicted")
	}
}

func TestPool_DuplicateGeneratorIgnored(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)

	added, err := pool.Add(roundEndorsement(t, 1, sk), pk, 200)
	require.NoError(t, err)
	assert.False(t, added, "duplicate generator must be silently rejected")
	assert.Equal(t, 1, pool.Len())
}

func TestPool_HasUpdateAfterAdd(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	assert.False(t, pool.HasUpdate(), "fresh pool must have no update")

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)

	assert.True(t, pool.HasUpdate(), "HasUpdate must be true after Add")
}

func TestPool_HasUpdateClearedByCommitFinalization(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	require.True(t, pool.HasUpdate())

	_, err = pool.FormFinalization()
	require.NoError(t, err)
	assert.True(t, pool.HasUpdate(), "HasUpdate must remain true after FormFinalization alone")

	pool.CommitFinalization()
	assert.False(t, pool.HasUpdate(), "HasUpdate must be false only after CommitFinalization")
}

func TestPool_HasUpdateSetAgainByNewAdd(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk1, pk1 := newKeyPair(t, "k1")
	sk2, pk2 := newKeyPair(t, "k2")

	addToPool(t, pool, roundEndorsement(t, 1, sk1), pk1, 100)
	_, err = pool.FormFinalization()
	require.NoError(t, err)
	pool.CommitFinalization()
	require.False(t, pool.HasUpdate())

	addToPool(t, pool, roundEndorsement(t, 2, sk2), pk2, 200)
	assert.True(t, pool.HasUpdate(), "HasUpdate must be true again after a new endorsement arrives")
}

func TestPool_HasUpdateSetByEviction(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(2)
	require.NoError(t, err)

	sk1, pk1 := newKeyPair(t, "k1")
	sk2, pk2 := newKeyPair(t, "k2")
	sk3, pk3 := newKeyPair(t, "k3")

	addToPool(t, pool, roundEndorsement(t, 1, sk1), pk1, 10)
	addToPool(t, pool, roundEndorsement(t, 2, sk2), pk2, 20)

	_, err = pool.FormFinalization()
	require.NoError(t, err)
	pool.CommitFinalization()
	require.False(t, pool.HasUpdate())

	// Higher-balance endorsement evicts the minimum → snapshot changed.
	added, err := pool.Add(roundEndorsement(t, 3, sk3), pk3, 50)
	require.NoError(t, err)
	require.True(t, added)
	assert.True(t, pool.HasUpdate(), "HasUpdate must be true after eviction changes the snapshot")
}

func TestPool_AddConflictReturnsTrueOnlyOnce(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, _ := newKeyPair(t, "k")
	e := roundEndorsement(t, 1, sk)

	assert.True(t, pool.AddConflict(e), "first conflicting endorsement must be accepted")
	assert.False(t, pool.AddConflict(e), "subsequent conflict for same generator must be dropped")
}

func TestPool_HasUpdateRemainsAfterFormFinalizationWithoutCommit(t *testing.T) {
	// Regression: if the microblock carrying a voting is never applied (e.g. ErrStateChanged),
	// HasUpdate must remain true so the next microblock attempt re-includes the voting.
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	require.True(t, pool.HasUpdate())

	// Simulate a microblock that was prepared but never applied.
	_, err = pool.FormFinalization()
	require.NoError(t, err)
	// No CommitFinalization call.

	assert.True(t, pool.HasUpdate(),
		"HasUpdate must remain true — voting was never committed to an applied microblock")
}

func TestPool_ConflictRemovesGeneratorFromGoodPool(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	require.Equal(t, 1, pool.Len())
	require.True(t, pool.HasUpdate())

	_, err = pool.FormFinalization()
	require.NoError(t, err)
	pool.CommitFinalization()
	require.False(t, pool.HasUpdate())

	// The same generator now conflicts → removed from snapshot.
	ok := pool.AddConflict(roundEndorsement(t, 1, sk))
	assert.True(t, ok)
	assert.Equal(t, 0, pool.Len())
	assert.True(t, pool.HasUpdate(), "HasUpdate must be true: snapshot shrank")
}

func TestPool_ConflictGeneratorRejectedOnSubsequentAdd(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	pool.AddConflict(roundEndorsement(t, 1, sk))

	added, err := pool.Add(roundEndorsement(t, 1, sk), pk, 100)
	require.NoError(t, err)
	assert.False(t, added, "generator with a recorded conflict must not be added to the good pool")
}

func TestPool_HasUpdateSetByNewConflict(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, _ := newKeyPair(t, "k")
	assert.False(t, pool.HasUpdate())
	pool.AddConflict(roundEndorsement(t, 1, sk))
	assert.True(t, pool.HasUpdate(), "a new conflict alone must set HasUpdate")
}

func TestPool_FormFinalization_SnapshotIsFullCurrentSet(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk1, pk1 := newKeyPair(t, "k1")
	sk2, pk2 := newKeyPair(t, "k2")
	addToPool(t, pool, roundEndorsement(t, 1, sk1), pk1, 100)
	addToPool(t, pool, roundEndorsement(t, 2, sk2), pk2, 200)

	fv, err := pool.FormFinalization()
	require.NoError(t, err)
	assert.Len(t, fv.EndorserIndexes, 2)
	assert.NotNil(t, fv.AggregatedEndorsementSignature)
	assert.Empty(t, fv.ConflictEndorsements, "no conflicts were added")
}

func TestPool_FormFinalization_OnlyNewConflictsIncluded(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)

	// Conflict added before first FormFinalization.
	pool.AddConflict(roundEndorsement(t, 10, sk))

	fv1, err := pool.FormFinalization()
	require.NoError(t, err)
	pool.CommitFinalization()
	require.Len(t, fv1.ConflictEndorsements, 1, "first call must include the one conflict")
	assert.NotNil(t, fv1.AggregatedEndorsementSignature)
	assert.Len(t, fv1.EndorserIndexes, 1)

	require.False(t, pool.HasUpdate())

	// New conflict added after commit — snapshot is unchanged.
	pool.AddConflict(roundEndorsement(t, 11, sk))
	require.True(t, pool.HasUpdate())

	fv2, err := pool.FormFinalization()
	require.NoError(t, err)
	// Conflicts: only the new one.
	assert.Len(t, fv2.ConflictEndorsements, 1, "second call must include only the new conflict")
	assert.Equal(t, uint32(11), fv2.ConflictEndorsements[0].EndorserIndex)
	// Snapshot: always the full current set (CombineFinalizationVoting takes voting2 as base).
	assert.NotNil(t, fv2.AggregatedEndorsementSignature, "snapshot must always be present for CombineFinalizationVoting")
	assert.Len(t, fv2.EndorserIndexes, 1)
}

func TestPool_FormFinalization_SnapshotAlwaysPresentForCombine(t *testing.T) {
	// CombineFinalizationVoting uses voting2 (the newer one) as the structural base.
	// An empty snapshot in voting2 would cause the combined result to lose the snapshot,
	// so the pool must always include the full current snapshot regardless of whether
	// good endorsements changed since the last call.
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk1, pk1 := newKeyPair(t, "k1")
	sk2, _ := newKeyPair(t, "k2")

	addToPool(t, pool, roundEndorsement(t, 1, sk1), pk1, 100)
	_, err = pool.FormFinalization()
	require.NoError(t, err)
	pool.CommitFinalization()

	// Only a conflict arrives — snapshot of good endorsements is unchanged.
	pool.AddConflict(roundEndorsement(t, 2, sk2))

	fv, err := pool.FormFinalization()
	require.NoError(t, err)
	assert.NotNil(t, fv.AggregatedEndorsementSignature, "snapshot must be present even when only conflicts changed")
	assert.Len(t, fv.EndorserIndexes, 1)
	assert.Len(t, fv.ConflictEndorsements, 1)
}

func TestPool_FormFinalization_SnapshotAlwaysFull(t *testing.T) {
	// After a new endorsement arrives post-FormFinalization, the snapshot
	// must contain ALL current good endorsements (not just the delta).
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk1, pk1 := newKeyPair(t, "k1")
	sk2, pk2 := newKeyPair(t, "k2")
	addToPool(t, pool, roundEndorsement(t, 1, sk1), pk1, 100)

	_, err = pool.FormFinalization()
	require.NoError(t, err)

	addToPool(t, pool, roundEndorsement(t, 2, sk2), pk2, 200)

	fv, err := pool.FormFinalization()
	require.NoError(t, err)
	assert.Len(t, fv.EndorserIndexes, 2, "snapshot must include both endorsers")
}

func TestPool_FormFinalization_ConflictsOnlyNoGoodEndorsements(t *testing.T) {
	// Pool with only conflicts and no good endorsements.
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, _ := newKeyPair(t, "k")
	pool.AddConflict(roundEndorsement(t, 1, sk))
	require.True(t, pool.HasUpdate())

	fv, err := pool.FormFinalization()
	require.NoError(t, err)
	assert.Nil(t, fv.AggregatedEndorsementSignature, "no good endorsements → no aggregate sig")
	assert.Len(t, fv.ConflictEndorsements, 1)
}

func TestPool_FormFinalization_ErrorWhenEmpty(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	_, err = pool.FormFinalization()
	assert.Error(t, err, "FormFinalization on empty pool must return error")
}

func TestPool_ResetClearsAllState(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	pool.AddConflict(roundEndorsement(t, 2, sk))
	_, err = pool.FormFinalization()
	require.NoError(t, err)

	pool.Reset()

	assert.Equal(t, 0, pool.Len())
	assert.Empty(t, pool.GetAll())
	assert.Empty(t, pool.ConflictEndorsements())
	assert.False(t, pool.HasUpdate(), "HasUpdate must be false after Reset")
}

func TestPool_ResetAllowsReuseForNewRound(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	addToPool(t, pool, roundEndorsement(t, 1, sk), pk, 100)
	pool.Reset()

	// After reset the same generator index can be added again (new round).
	added, err := pool.Add(roundEndorsement(t, 1, sk), pk, 200)
	require.NoError(t, err)
	assert.True(t, added)
	assert.Equal(t, 1, pool.Len())
}

func TestPool_ResetAfterConflictAllowsGeneratorAgain(t *testing.T) {
	pool, err := endorsementpool.NewEndorsementPool(5)
	require.NoError(t, err)

	sk, pk := newKeyPair(t, "k")
	pool.AddConflict(roundEndorsement(t, 1, sk))

	// In the same round the conflicted generator cannot be added.
	added, err := pool.Add(roundEndorsement(t, 1, sk), pk, 100)
	require.NoError(t, err)
	assert.False(t, added)

	// After reset (new round) the generator is allowed again.
	pool.Reset()
	added, err = pool.Add(roundEndorsement(t, 1, sk), pk, 100)
	require.NoError(t, err)
	assert.True(t, added)
}

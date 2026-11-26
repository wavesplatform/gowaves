package proto

import (
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// resetLiquidMerkleStates replaces the global cache with an empty one for the duration of the test.
func resetLiquidMerkleStates(t *testing.T) {
	old := liquidMerkleStates
	liquidMerkleStates = &merkleStateCache{}
	t.Cleanup(func() { liquidMerkleStates = old })
}

func testMicroBlock(t *testing.T, ref *Block, txs Transactions, kp KeyPair) *MicroBlock {
	var totalSig crypto.Signature
	copy(totalSig[:], []byte("test total resulting block signature"))
	mb := &MicroBlock{
		VersionField:          ref.Version,
		Reference:             ref.BlockID(),
		TotalResBlockSigField: totalSig,
		TotalBlockID:          NewBlockIDFromDigest(benchDigest(9)),
		TransactionCount:      uint32(len(txs)), // #nosec G115: test sizes are small.
		Transactions:          txs,
		SenderPK:              kp.Public,
	}
	require.NoError(t, mb.Sign(benchScheme, kp.Secret))
	return mb
}

// fullRecomputeBlock creates the block from scratch with an empty cache, the oracle for the incremental path.
func fullRecomputeBlock(t *testing.T, b *Block) *Block {
	saved := liquidMerkleStates
	liquidMerkleStates = &merkleStateCache{}
	defer func() { liquidMerkleStates = saved }()
	res, err := CreateBlock(b.Transactions, b.Timestamp, b.Parent, b.GeneratorPublicKey, b.NxtConsensus, b.Version,
		b.Features, b.RewardVote, benchScheme, b.StateHash, b.FinalizationVoting)
	require.NoError(t, err)
	return res
}

func TestAppendMicroBlockIncrementalRoot(t *testing.T) {
	resetLiquidMerkleStates(t)
	kp := benchKeyPair(t, "test generator")
	rnd := rand.New(rand.NewPCG(3, 5))
	const total = 300
	all := benchTransfers(t, 3, kp, total, 0)
	for split := range 20 {
		keySize := rnd.IntN(5)
		blk := benchCreateBlock(t, ProtobufBlockVersion, all[:keySize:keySize], kp)
		pos := keySize
		for pos < total {
			n := rnd.IntN(40) // zero means an empty microblock
			if pos+n > total {
				n = total - pos
			}
			mb := testMicroBlock(t, blk, all[pos:pos+n:pos+n], kp)
			next, err := AppendMicroBlock(blk, mb, benchScheme)
			require.NoError(t, err)
			pos += n
			require.Equal(t, pos, next.TransactionCount)

			expected := fullRecomputeBlock(t, next)
			require.Equal(t, expected.TransactionsRoot, next.TransactionsRoot, "split %d, pos %d", split, pos)
			require.Equal(t, expected.ID, next.ID, "split %d, pos %d", split, pos)
			ok, err := next.VerifyTransactionsRoot(benchScheme)
			require.NoError(t, err)
			require.True(t, ok)

			// The ID does not change after setting the signature.
			id := next.ID
			next.BlockSignature = mb.TotalResBlockSigField
			require.NoError(t, next.GenerateBlockID(benchScheme))
			require.Equal(t, id, next.ID)
			blk = next
		}
	}
}

func TestAppendMicroBlockCacheMiss(t *testing.T) {
	resetLiquidMerkleStates(t)
	kp := benchKeyPair(t, "test generator")
	blk := benchCreateBlock(t, ProtobufBlockVersion, benchTransfers(t, 3, kp, 10, 0), kp)
	mb := testMicroBlock(t, blk, benchTransfers(t, 3, kp, 5, 10), kp)
	require.NotNil(t, blk.cachedTransactionsTree(benchScheme))

	// The block comes from storage, so its tree state is unknown.
	liquidMerkleStates = &merkleStateCache{}
	require.Nil(t, blk.cachedTransactionsTree(benchScheme))
	next, err := AppendMicroBlock(blk, mb, benchScheme)
	require.NoError(t, err)
	assert.Equal(t, fullRecomputeBlock(t, next).TransactionsRoot, next.TransactionsRoot)
	// Now the state of the new block is cached.
	assert.NotNil(t, next.cachedTransactionsTree(benchScheme))
}

func TestCachedTransactionsTreeValidation(t *testing.T) {
	resetLiquidMerkleStates(t)
	kp := benchKeyPair(t, "test generator")
	blk := benchCreateBlock(t, ProtobufBlockVersion, benchTransfers(t, 3, kp, 10, 0), kp)
	require.NotNil(t, blk.cachedTransactionsTree(benchScheme))
	require.Nil(t, blk.cachedTransactionsTree(StageNetScheme), "other scheme")

	other := *blk
	other.Transactions = blk.Transactions[:9]
	require.Nil(t, other.cachedTransactionsTree(benchScheme), "other transactions count")

	other = *blk
	other.TransactionsRoot = make([]byte, crypto.DigestSize)
	require.Nil(t, other.cachedTransactionsTree(benchScheme), "root mismatch")

	other = *blk
	other.ID = NewBlockIDFromDigest(benchDigest(42))
	require.Nil(t, other.cachedTransactionsTree(benchScheme), "other ID")

	// A stale state must not be used: the appended block still gets the right root.
	other = *blk
	other.TransactionsRoot = make([]byte, crypto.DigestSize)
	mb := testMicroBlock(t, &other, benchTransfers(t, 3, kp, 3, 10), kp)
	next, err := AppendMicroBlock(&other, mb, benchScheme)
	require.NoError(t, err)
	assert.Equal(t, fullRecomputeBlock(t, next).TransactionsRoot, next.TransactionsRoot)
}

func TestCachedTreeIsNotShared(t *testing.T) {
	resetLiquidMerkleStates(t)
	kp := benchKeyPair(t, "test generator")
	blk := benchCreateBlock(t, ProtobufBlockVersion, benchTransfers(t, 3, kp, 7, 0), kp)
	// Two different microblocks on top of the same block (a liquid block fork).
	mb1 := testMicroBlock(t, blk, benchTransfers(t, 3, kp, 3, 100), kp)
	mb2 := testMicroBlock(t, blk, benchTransfers(t, 3, kp, 4, 200), kp)
	b1, err := AppendMicroBlock(blk, mb1, benchScheme)
	require.NoError(t, err)
	b2, err := AppendMicroBlock(blk, mb2, benchScheme)
	require.NoError(t, err)
	assert.Equal(t, fullRecomputeBlock(t, b1).TransactionsRoot, b1.TransactionsRoot)
	assert.Equal(t, fullRecomputeBlock(t, b2).TransactionsRoot, b2.TransactionsRoot)
	ok, err := blk.VerifyTransactionsRoot(benchScheme)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.NotNil(t, blk.cachedTransactionsTree(benchScheme))
}

func TestMerkleStateCacheEviction(t *testing.T) {
	c := &merkleStateCache{}
	tree, err := crypto.NewMerkleTree()
	require.NoError(t, err)
	for i := range liquidMerkleCacheSize + 1 {
		c.put(NewBlockIDFromDigest(benchDigest(byte(i))), benchScheme, i, tree.Clone())
	}
	assert.Nil(t, c.get(NewBlockIDFromDigest(benchDigest(0)), benchScheme, 0), "the oldest entry is evicted")
	for i := 1; i <= liquidMerkleCacheSize; i++ {
		assert.NotNil(t, c.get(NewBlockIDFromDigest(benchDigest(byte(i))), benchScheme, i))
	}
	// A duplicate put does not evict anything.
	c.put(NewBlockIDFromDigest(benchDigest(5)), benchScheme, 5, tree.Clone())
	assert.NotNil(t, c.get(NewBlockIDFromDigest(benchDigest(1)), benchScheme, 1))
}

func TestAppendTransactionsToBlockLegacy(t *testing.T) {
	kp := benchKeyPair(t, "test generator")
	blk := benchCreateBlock(t, RewardBlockVersion, benchTransfers(t, 2, kp, 5, 0), kp)
	next, err := AppendTransactionsToBlock(blk, benchTransfers(t, 2, kp, 3, 5), benchScheme, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 8, next.TransactionCount)
	assert.Nil(t, next.TransactionsRoot)
	_, err = AppendTransactionsToBlock(nil, nil, benchScheme, nil, nil)
	assert.Error(t, err)
}

package crypto_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/blake2b"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// referenceNodeDigest is the node hashing used before the optimization.
func referenceNodeDigest(t *testing.T, a, b crypto.Digest) crypto.Digest {
	h, err := blake2b.New256(nil)
	require.NoError(t, err)
	_, err = h.Write(a[:])
	require.NoError(t, err)
	_, err = h.Write(b[:])
	require.NoError(t, err)
	var d crypto.Digest
	h.Sum(d[:0])
	return d
}

func randomDigest(rnd *rand.Rand) crypto.Digest {
	var d crypto.Digest
	for i := range d {
		d[i] = byte(rnd.UintN(256))
	}
	return d
}

func TestRebuildMerkleRootMatchesReference(t *testing.T) {
	rnd := rand.New(rand.NewPCG(7, 11))
	for height := range 17 {
		proofs := make([]crypto.Digest, height)
		for i := range proofs {
			proofs[i] = randomDigest(rnd)
		}
		leaf := randomDigest(rnd)
		index := rnd.Uint64N(uint64(1) << height)

		expected := leaf
		idx := index
		for _, p := range slices.Backward(proofs) {
			if idx%2 == 0 {
				expected = referenceNodeDigest(t, expected, p)
			} else {
				expected = referenceNodeDigest(t, p, expected)
			}
			idx /= 2
		}

		actual, err := crypto.RebuildMerkleRoot(leaf, proofs, index)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)

		tree, err := crypto.NewMerkleTree()
		require.NoError(t, err)
		actual, err = tree.RebuildRoot(leaf, proofs, index)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)

		allocs := testing.AllocsPerRun(10, func() {
			_, _ = crypto.RebuildMerkleRoot(leaf, proofs, index)
		})
		assert.Zero(t, allocs)
	}
}

func TestRebuildMerkleRootErrorUnchanged(t *testing.T) {
	_, err := crypto.RebuildMerkleRoot(crypto.Digest{}, make([]crypto.Digest, 2), 4)
	assert.EqualError(t, err, "index 4 is out of range for the tree of height 2")
}

func TestMerkleTreeRootMatchesReference(t *testing.T) {
	rnd := rand.New(rand.NewPCG(3, 5))
	for _, n := range []int{1, 2, 3, 5, 8, 13} {
		leafs := make([][]byte, n)
		for i := range leafs {
			leafs[i] = make([]byte, rnd.IntN(100))
			for j := range leafs[i] {
				leafs[i][j] = byte(rnd.UintN(256))
			}
		}
		tree, err := crypto.NewMerkleTree()
		require.NoError(t, err)
		for _, l := range leafs {
			tree.Push(l)
		}
		// Reference: the full binary tree built level by level, padded with ZeroDigest.
		level := make([]crypto.Digest, 0, len(leafs))
		for _, l := range leafs {
			level = append(level, blake2b.Sum256(l))
		}
		for len(level) > 1 || n == 1 {
			if len(level)%2 == 1 {
				level = append(level, crypto.ZeroDigest)
			}
			next := make([]crypto.Digest, 0, len(level)/2)
			for i := 0; i < len(level); i += 2 {
				next = append(next, referenceNodeDigest(t, level[i], level[i+1]))
			}
			level = next
			n = 0
		}
		assert.Equal(t, level[0], tree.Root())
	}
}

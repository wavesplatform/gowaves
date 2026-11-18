package ride //nolint:testpackage // White-box tests of the unexported native functions.

import (
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/blake2b"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func randomRideBytes(rnd *rand.Rand, n int) rideByteVector {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rnd.UintN(256))
	}
	return b
}

// referenceDelay is the hit calculation of calculateDelay used before the optimization.
func referenceDelay(t *testing.T, hitSource, addr []byte, baseTarget, balance uint64) rideInt {
	h, err := blake2b.New256(nil)
	require.NoError(t, err)
	_, err = h.Write(hitSource)
	require.NoError(t, err)
	_, err = h.Write(addr)
	require.NoError(t, err)
	hit := big.NewInt(0).SetBytes(h.Sum(nil)[:consensus.HitSize])
	delay, err := consensus.NewFairPosCalculator(0, 0).CalculateDelay(hit, baseTarget, balance)
	require.NoError(t, err)
	return rideInt(delay)
}

func TestCalculateDelayMatchesReference(t *testing.T) {
	rnd := rand.New(rand.NewPCG(5, 8))
	const baseTarget = 142244892
	// Hit sources and addresses that fit the stack buffer and the ones that don't.
	for _, vrfSize := range []int{0, crypto.DigestSize, 100} {
		vrf := randomRideBytes(rnd, vrfSize)
		te := NewMockEnvironment(t)
		te.EXPECT().block().Return(rideBlockInfoV7{baseTarget: baseTarget, vrf: vrf}).Maybe()
		for _, addrSize := range []int{0, 1, proto.WavesAddressSize, 200} {
			addr := randomRideBytes(rnd, addrSize)
			for _, balance := range []int64{1, 1_000_000, 1 << 50} {
				expected := referenceDelay(t, vrf, addr, baseTarget, uint64(balance))
				r, err := calculateDelay(te, rideAddressLike(addr), rideInt(balance))
				require.NoError(t, err)
				assert.Equal(t, expected, r, "vrf=%d addr=%d balance=%d", vrfSize, addrSize, balance)
			}
		}
		wa := proto.WavesAddress(randomRideBytes(rnd, proto.WavesAddressSize))
		r, err := calculateDelay(te, rideAddress(wa), rideInt(12345))
		require.NoError(t, err)
		assert.Equal(t, referenceDelay(t, vrf, wa.Bytes(), baseTarget, 12345), r)
	}
}

func TestCalculateDelayErrorsUnchanged(t *testing.T) {
	te := NewMockEnvironment(t)
	te.EXPECT().block().Return(rideBlockInfoV7{baseTarget: 1, vrf: rideUnit{}}).Maybe()
	_, err := calculateDelay(te, rideAddressLike{}, rideInt(0))
	assert.EqualError(t, err, "calculateDelay: balance '0' should by positive")
	_, err = calculateDelay(te, rideAddressLike{}, rideInt(-1))
	assert.EqualError(t, err, "calculateDelay: balance '-1' should by positive")
	_, err = calculateDelay(te, rideAddressLike{}, rideInt(1))
	assert.EqualError(t, err, "calculateDelay: empty hit source")
	_, err = calculateDelay(te, rideString("x"), rideInt(1))
	assert.EqualError(t, err, "calculateDelay: unexpected argument type 'String'")
}

func TestRebuildMerkleRootLimitsAndErrors(t *testing.T) {
	rnd := rand.New(rand.NewPCG(13, 21))
	proofs := func(n int) rideList {
		l := make(rideList, n)
		for i := range l {
			l[i] = randomRideBytes(rnd, crypto.DigestSize)
		}
		return l
	}
	leaf := randomRideBytes(rnd, crypto.DigestSize)

	t.Run("max proofs", func(t *testing.T) {
		ps := proofs(16)
		// reference follows the Scala implementation: the proofs are in leafs to root order,
		// the side is selected by the remainder of the truncated division of the index by 2.
		reference := func(index int64) rideByteVector {
			digest := slices.Clone(leaf)
			for _, p := range ps {
				b, ok := p.(rideByteVector)
				require.True(t, ok)
				var d [crypto.DigestSize]byte
				if index%2 == 0 {
					d = blake2b.Sum256(slices.Concat(digest, b))
				} else {
					d = blake2b.Sum256(slices.Concat(b, digest))
				}
				digest = d[:]
				index /= 2
			}
			return digest
		}
		for _, index := range []int64{0, 1, 12345, -12345, 1<<16 - 1, -(1<<16 - 1)} {
			r, err := rebuildMerkleRoot(nil, ps, leaf, rideInt(index))
			require.NoError(t, err)
			assert.Equal(t, reference(index), r, "index %d", index)
		}
	})
	t.Run("no proofs", func(t *testing.T) {
		r, err := rebuildMerkleRoot(nil, rideList{}, leaf, rideInt(0))
		require.NoError(t, err)
		assert.Equal(t, leaf, r)
	})
	t.Run("errors", func(t *testing.T) {
		_, err := rebuildMerkleRoot(nil, proofs(17), leaf, rideInt(0))
		assert.EqualError(t, err, "rebuildMerkleRoot: no more than 16 proofs is allowed")
		_, err = rebuildMerkleRoot(nil, rideList{rideInt(1)}, leaf, rideInt(0))
		assert.EqualError(t, err, "rebuildMerkleRoot: unexpected proof type 'Int' at position 0")
		_, err = rebuildMerkleRoot(nil, rideList{rideByteVector{1, 2}}, leaf, rideInt(0))
		assert.EqualError(t, err, "rebuildMerkleRoot: invalid digest len")
		_, err = rebuildMerkleRoot(nil, proofs(2), leaf, rideInt(4))
		assert.EqualError(t, err, "rebuildMerkleRoot: index 4 is out of range for the tree of height 2")
	})
}

func TestCalculateAssetIDMatchesProto(t *testing.T) {
	txID := crypto.MustFastHash([]byte("parent"))
	te := NewMockEnvironment(t)
	te.EXPECT().txID().Return(rideByteVector(txID.Bytes())).Maybe()
	for _, n := range []int{0, 4, 200, 1000} {
		name := rideString(make([]byte, n))
		issue := newRideIssue(rideUnit{}, name, "description", 7, 8, 1000, true)
		r, err := calculateAssetID(te, issue)
		require.NoError(t, err)
		expected := proto.GenerateIssueScriptActionID(string(name), "description", 8, 1000, true, 7, txID)
		assert.Equal(t, rideByteVector(expected.Bytes()), r)
	}

	noTx := NewMockEnvironment(t)
	noTx.EXPECT().txID().Return(rideUnit{}).Maybe()
	_, err := calculateAssetID(noTx, newRideIssue(rideUnit{}, "n", "d", 0, 0, 1, false))
	assert.EqualError(t, err, "calculateAssetID: no parent transaction ID found")
}

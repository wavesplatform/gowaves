package crypto_test

import (
	"math/rand/v2"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/blake2b"

	"github.com/wavesplatform/gowaves/pkg/ride/crypto"
)

// referenceMerkleRootHash is the implementation used before the optimization.
func referenceMerkleRootHash(leaf, proof []byte) ([]byte, error) {
	leafData := make([]byte, len(leaf)+1)
	copy(leafData[1:], leaf)
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, err
	}
	_, _ = h.Write(leafData)
	hash := h.Sum(nil)
	for pos := 0; len(proof[pos:]) > 0; {
		if len(proof[pos:]) < 2 {
			return nil, errors.New("merkle: invalid proof length")
		}
		side := proof[pos]
		pos++
		l := int(proof[pos])
		pos++
		if len(proof[pos:]) < l {
			return nil, errors.New("merkle: invalid proof length")
		}
		other := proof[pos : pos+l]
		pos += l
		h.Reset()
		_, _ = h.Write([]byte{0x01})
		switch side {
		case 0:
			_, _ = h.Write(hash)
			_, _ = h.Write(other)
		case 1:
			_, _ = h.Write(other)
			_, _ = h.Write(hash)
		default:
			return nil, errors.Errorf("merkle: invalid side value %d", side)
		}
		hash = h.Sum(nil)
	}
	return hash, nil
}

func randomBytes(rnd *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rnd.UintN(256))
	}
	return b
}

func TestMerkleRootHashMatchesReference(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	// Leaf sizes around the stack buffer size (1 + 32 + 255 bytes) and sibling sizes up to the maximum of 255 bytes.
	leafSizes := []int{0, 1, 4, 32, 286, 287, 288, 1000}
	siblingSizes := []int{0, 1, 32, 100, 254, 255}
	for _, ls := range leafSizes {
		leaf := randomBytes(rnd, ls)
		var proof []byte
		for i, ss := range siblingSizes {
			proof = append(proof, byte(i%2), byte(ss))
			proof = append(proof, randomBytes(rnd, ss)...)
		}
		for end := 0; end <= len(proof); end++ { // All prefixes, including the invalid ones.
			expected, expectedErr := referenceMerkleRootHash(leaf, proof[:end])
			actual, err := crypto.MerkleRootHash(leaf, proof[:end])
			if expectedErr != nil {
				assert.EqualError(t, err, expectedErr.Error())
				continue
			}
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		}
	}
}

func TestMerkleRootHashInvalidSide(t *testing.T) {
	_, err := crypto.MerkleRootHash([]byte{1}, []byte{2, 1, 0})
	assert.EqualError(t, err, "merkle: invalid side value 2")
	_, err = crypto.MerkleRootHash([]byte{1}, []byte{0})
	assert.EqualError(t, err, "merkle: invalid proof length")
	_, err = crypto.MerkleRootHash([]byte{1}, []byte{0, 2, 0})
	assert.EqualError(t, err, "merkle: invalid proof length")
}

func TestMerkleRootHashAllocations(t *testing.T) {
	leaf := []byte{0, 0, 29, 33}
	proof := append([]byte{0, 32}, make([]byte, 32)...)
	allocs := testing.AllocsPerRun(10, func() {
		_, _ = crypto.MerkleRootHash(leaf, proof)
	})
	assert.LessOrEqual(t, allocs, 1.0) // Only the result.
}

package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyTreeRoot(t *testing.T) {
	tree, err := NewMerkleTree()
	require.NoError(t, err)
	expected, err := FastHash([]byte{0})
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, tree.Root())
}

func TestMerkleTreeRoot(t *testing.T) {
	for _, test := range []struct {
		leafs [][]byte
		root  string
	}{
		{[][]byte{{0}, {1}, {2}}, "0000000000000000000000000000000000000000000000000000000000000000"},
	} {
		db, err := hex.DecodeString(test.root)
		require.NoError(t, err)
		ed, err := NewDigestFromBytes(db)
		require.NoError(t, err)
		tree, err := NewMerkleTree()
		require.NoError(t, err)
		for _, l := range test.leafs {
			tree.Push(l)
		}
		r := tree.Root()
		assert.ElementsMatch(t, ed, r)
	}
}

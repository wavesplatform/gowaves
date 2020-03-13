package crypto

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/mr-tron/base58"
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
	for i, test := range []struct {
		leafs [][]byte
		root  string
	}{
		{[][]byte{}, "D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu"},
		{[][]byte{{0x01, 0x02, 0x03}}, "AwohNCpXvMWhFD5XgM9ox9jquXwTAQrzdNgTJG8zP9Es"},
		{[][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}}, "CT3jS3cgFQw8DVoD4HzJ2xsjP6aVRDUGesWWe4pF5Mbg"},
		{[][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}, {0x07, 0x08, 0x09}}, "C7w1DCGggWHaeYJ81QsNyxsEvpmedFoD85fDDJ9JZ4Jq"},
		{[][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}, {0x07, 0x08, 0x09}, {0x0a, 0x0b, 0x0c}}, "E1V1VHDEiDLbSRcTsgfipyzj1Rr1pP1CfMeteJ95Ef1H"},
		{[][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}, {0x07, 0x08, 0x09}, {0x0a, 0x0b, 0x0c}, {0x0d, 0x0e, 0x0f}}, "Ap9P5bmavaRGSqBSFXHR5THvEzwjG5umxUxTRuCpXs86"},
		{[][]byte{{0x01}, {0x02}, {0x03}}, "41DDtujqe8KxBDcmkSzXcJWCRaZzBF5b9t5zArBnUmrt"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}}, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}}, "9Lf8o7a5ccSGXzBW11ZvLVwNkdj6vBazJYG7LwLYu4Q8"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}}, "Q646xuqCsvKbVmrUCsenwQaWwQqaKMW64KtSxshmnNv"},
	} {
		db, err := base58.Decode(test.root)
		require.NoError(t, err)
		ed, err := NewDigestFromBytes(db)
		require.NoError(t, err)
		tree, err := NewMerkleTree()
		require.NoError(t, err)
		for _, l := range test.leafs {
			tree.Push(l)
		}
		r := tree.Root()
		assert.ElementsMatch(t, ed, r, fmt.Sprintf("#%d", i+1))
	}
}

func BenchmarkMerkleTreeEven(b *testing.B) {
	a := make([][]byte, 2048)
	for i := range a {
		v := make([]byte, 2048)
		rand.Read(v)
		a[i] = v
	}
	tree, err := NewMerkleTree()
	require.NoError(b, err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, v := range a {
			tree.Push(v)
		}
		d := tree.Root()
		_ = d
	}
}

func BenchmarkMerkleTreeOdd(b *testing.B) {
	a := make([][]byte, 2123)
	for i := range a {
		v := make([]byte, 2048)
		rand.Read(v)
		a[i] = v
	}
	tree, err := NewMerkleTree()
	require.NoError(b, err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, v := range a {
			tree.Push(v)
		}
		d := tree.Root()
		_ = d
	}
}

func BenchmarkMerkleTreeRootEven(b *testing.B) {
	a := make([][]byte, 2048)
	for i := range a {
		v := make([]byte, 2048)
		rand.Read(v)
		a[i] = v
	}
	tree, err := NewMerkleTree()
	require.NoError(b, err)
	for _, v := range a {
		tree.Push(v)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := tree.Root()
		_ = d
	}
}

func BenchmarkMerkleTreeRootOdd(b *testing.B) {
	a := make([][]byte, 2123)
	for i := range a {
		v := make([]byte, 2048)
		rand.Read(v)
		a[i] = v
	}
	tree, err := NewMerkleTree()
	require.NoError(b, err)
	for _, v := range a {
		tree.Push(v)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := tree.Root()
		_ = d
	}
}

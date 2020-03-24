package crypto

import (
	"fmt"
	"math/rand"
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
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}}, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}}, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}}, "9Lf8o7a5ccSGXzBW11ZvLVwNkdj6vBazJYG7LwLYu4Q8"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}}, "Q646xuqCsvKbVmrUCsenwQaWwQqaKMW64KtSxshmnNv"},
	} {
		ed, err := NewDigestFromBase58(test.root)
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

func TestMerkleTreeRebuildRoot(t *testing.T) {
	for i, test := range []struct {
		index  uint64
		root   string
		leaf   string
		proofs []string
	}{
		{0, "834CiPdUXEvQwQbFVNYdMca4HbeqQBrfiYthgcYgHaW2", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn", []string{"D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu"}},
		{0, "75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn", []string{"DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR"}},
		{1, "75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR", []string{"H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn"}},
		{0, "41DDtujqe8KxBDcmkSzXcJWCRaZzBF5b9t5zArBnUmrt", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn", []string{"GP1h9LyC6TbpBfi4fMf4qWUNGCFGrJFcBYp1aaVeUGAU", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR"}},
		{1, "41DDtujqe8KxBDcmkSzXcJWCRaZzBF5b9t5zArBnUmrt", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR", []string{"GP1h9LyC6TbpBfi4fMf4qWUNGCFGrJFcBYp1aaVeUGAU", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn"}},
		{2, "41DDtujqe8KxBDcmkSzXcJWCRaZzBF5b9t5zArBnUmrt", "GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9", []string{"75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu"}},
		{0, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn", []string{"5pZuB8CRjdAuedfhTpkcvwRxCioTL9Fu2dDdJAPMRHPg", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR"}},
		{1, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR", []string{"5pZuB8CRjdAuedfhTpkcvwRxCioTL9Fu2dDdJAPMRHPg", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn"}},
		{2, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh", "GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9", []string{"75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "7jsrwD9Xi7TjVoksaV1CDDUWYhFaz7HQmAoWwLEiZa6D"}},
		{3, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh", "7jsrwD9Xi7TjVoksaV1CDDUWYhFaz7HQmAoWwLEiZa6D", []string{"75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9"}},
		{0, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn", []string{"q1u2PJhro1cwZw5mUuujXm94f245tGS5vbP5yNwLbEv", "5pZuB8CRjdAuedfhTpkcvwRxCioTL9Fu2dDdJAPMRHPg", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR"}},
		{1, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ", "DbiFyDrmumGtTu39Hh1GjK14oKvLdsgUuwqm51JhcbnR", []string{"q1u2PJhro1cwZw5mUuujXm94f245tGS5vbP5yNwLbEv", "5pZuB8CRjdAuedfhTpkcvwRxCioTL9Fu2dDdJAPMRHPg", "H2NvG7X3qQK7Uptsgoe514hUciZK81qsYiswxSXeiLKn"}},
		{2, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ", "GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9", []string{"q1u2PJhro1cwZw5mUuujXm94f245tGS5vbP5yNwLbEv", "75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "7jsrwD9Xi7TjVoksaV1CDDUWYhFaz7HQmAoWwLEiZa6D"}},
		{3, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ", "7jsrwD9Xi7TjVoksaV1CDDUWYhFaz7HQmAoWwLEiZa6D", []string{"q1u2PJhro1cwZw5mUuujXm94f245tGS5vbP5yNwLbEv", "75Aaexax3uEQNg5HAb137jC3TK64RG1S6xrBGvuupWXp", "GemGCop1arCvTY447FLH8tDQF7knvzNCocNTHqKQBus9"}},
		{4, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ", "HujdCWdbCbGyzriVW8Aeu6ojcZaghwJNwSotqJmt2CcU", []string{"2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh", "D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu", "D4bn122GiEqs99z526GdhYETJqctLHGSmWokypEo9qu"}},
	} {
		r, err := NewDigestFromBase58(test.root)
		require.NoError(t, err)
		l, err := NewDigestFromBase58(test.leaf)
		require.NoError(t, err)
		pfs := make([]Digest, len(test.proofs))
		for i, p := range test.proofs {
			d, err := NewDigestFromBase58(p)
			require.NoError(t, err)
			pfs[i] = d
		}
		tree, err := NewMerkleTree()
		require.NoError(t, err)
		root := tree.RebuildRoot(l, pfs, test.index)
		assert.Equal(t, r, root, fmt.Sprintf("#%d", i+1))
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

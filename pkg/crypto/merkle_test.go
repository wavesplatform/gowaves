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
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}}, "2AYMXo9fKWK6swVeAx4DnLuW2wKP8u3S8Ypax6MVWkNh"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}}, "6tt3obq44UqC4QwLhrKX2KsXV9GRBfhiNvzor2BQfgYZ"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}}, "9Lf8o7a5ccSGXzBW11ZvLVwNkdj6vBazJYG7LwLYu4Q8"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}}, "Q646xuqCsvKbVmrUCsenwQaWwQqaKMW64KtSxshmnNv"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}}, "EhU5omsQqPuv6AuHqDG2YBhKWcTv4kqri91rTzvw7svN"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}, {0x0b}}, "CKK4qPvKZ795itvpCogAaK2HYccdPavHZgmjrHuvagMY"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}, {0x0b}, {0x0c}}, "9FUGJH3b5rqZiWC5owYbkb6Wbc2Y8Tjeu4ExnR9EFAi"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}, {0x0b}, {0x0c}, {0x0d}}, "H3D8kj7u36BiGckLWxKauxUH2ZLuPiSkod74M2r8KNAw"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}, {0x0b}, {0x0c}, {0x0d}, {0x0e}}, "CPCSLbLBhe8pGeQ315pgTE9nwHz8tkNaHasAqs3AzxXM"},
		{[][]byte{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08}, {0x09}, {0x0a}, {0x0b}, {0x0c}, {0x0d}, {0x0e}, {0x0f}}, "F6GbdhHAF8NCwMDpdqbvgaEzzKZrixDmFy3YmAB8PT33"},
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

func TestStagenetFailure(t *testing.T) {
	tree, err := NewMerkleTree()
	require.NoError(t, err)
	for _, s := range []string{
		"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",
	} {
		b, err := base58.Decode(s)
		require.NoError(t, err)
		tree.Push(b)
	}
	expectedRH, err := base58.Decode("EhoUT3g4VAJgBRrCxcYfqnHTcTj4BbAvvSHsb9ZfAK2A")
	require.NoError(t, err)
	rh := tree.Root()
	assert.ElementsMatch(t, expectedRH, rh[:], fmt.Sprintf("RH: %s", base58.Encode(rh[:])))
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

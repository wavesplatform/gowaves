package crypto

import (
	"bytes"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/vectors_wycheproof.jsonl
	vectorsWycheproof embed.FS
)

type testVectorView struct {
	X       string `json:"x"`
	Y       string `json:"y"`
	R       string `json:"r"`
	S       string `json:"s"`
	Hash    string `json:"hash"`
	Valid   bool   `json:"valid"`
	Msg     string `json:"msg"`
	Comment string `json:"comment"`
}

func unmarshalTestDataToView(t *testing.T, fs embed.FS, testFileName, keccakHexChecksum string) []testVectorView {
	fileData, err := fs.ReadFile(filepath.Clean(testFileName))
	require.NoError(t, err)
	dataChecksum := hex.EncodeToString(MustKeccak256(fileData).Bytes())
	require.Equal(t, keccakHexChecksum, dataChecksum, "test data checksum mismatch")
	sep := []byte{'\n'}
	n := bytes.Count(fileData, sep) // approx number of records
	res := make([]testVectorView, 0, n)
	for record := range bytes.SplitSeq(fileData, sep) {
		record = bytes.TrimSpace(record)
		if len(record) == 0 {
			continue // skip empty lines
		}
		var tv testVectorView
		jsErr := json.Unmarshal(record, &tv)
		require.NoError(t, jsErr)
		res = append(res, tv)
	}
	return res
}

type testVector struct {
	PublicKey []byte // uncompressed SEC1, or raw (X||Y)
	Signature []byte
	Digest    []byte
	Valid     bool
}

func appendRawPubKey(t *testing.T, out []byte, x, y string) []byte {
	const coordSize = secP256r1RawPubKeySize / 2
	out = slices.Grow(out, len(out)+secP256r1RawPubKeySize)
	xBytes, err := hex.DecodeString(x)
	require.NoError(t, err)
	require.Len(t, xBytes, coordSize)
	yBytes, err := hex.DecodeString(y)
	require.NoError(t, err)
	require.Len(t, yBytes, coordSize)
	out = append(out, xBytes...)
	out = append(out, yBytes...)
	return out
}

func appendUncompressedPubKey(t *testing.T, out []byte, x, y string) []byte {
	out = slices.Grow(out, len(out)+secP256r1UncompressedPubKeySize)
	out = append(out, secP256r1UncompressedPubKeyPrefix)
	return appendRawPubKey(t, out, x, y)
}

func appendSignature(t *testing.T, out []byte, r, s string) []byte {
	out = slices.Grow(out, sec2562r1P1363SignatureSize)
	rBytes, err := hex.DecodeString(r)
	require.NoError(t, err)
	sBytes, err := hex.DecodeString(s)
	require.NoError(t, err)
	out = append(out, rBytes...)
	out = append(out, sBytes...)
	require.Len(t, out, sec2562r1P1363SignatureSize)
	return out
}

func transformViewsToVectors(t *testing.T, v []testVectorView) []testVector {
	res := make([]testVector, 0, len(v)*2) // *2 for both pubkey formats
	for _, tv := range v {
		rawPK := appendRawPubKey(t, nil, tv.X, tv.Y)
		uncompressedPK := appendUncompressedPubKey(t, nil, tv.X, tv.Y)
		sig := appendSignature(t, nil, tv.R, tv.S)
		digest, err := hex.DecodeString(tv.Hash)
		require.NoError(t, err)
		require.Len(t, digest, DigestSize)
		res = append(res,
			testVector{
				PublicKey: rawPK,
				Signature: sig,
				Digest:    digest,
				Valid:     tv.Valid,
			},
			testVector{
				PublicKey: uncompressedPK,
				Signature: sig,
				Digest:    digest,
				Valid:     tv.Valid,
			},
		)
	}
	return res
}

func TestSecp256Verify(t *testing.T) {
	const (
		testFileName                    = "testdata/vectors_wycheproof.jsonl"
		vectorsWycheproofKeccakChecksum = "d7e23f35ae6e092eda970e14c53d3e30261eb84a18389cc65041466ba5cb4c98"
	)
	vectorsView := unmarshalTestDataToView(t, vectorsWycheproof, testFileName, vectorsWycheproofKeccakChecksum)
	vectors := transformViewsToVectors(t, vectorsView)
	for i, tv := range vectors {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			ok, err := SecP256Verify(tv.Digest, tv.PublicKey, tv.Signature)
			if tv.Valid {
				require.NoError(t, err, "valid vector should not return error")
				require.True(t, ok, "valid vector should verify")
			} else {
				// Invalid vectors may return error or ok==false
				require.False(t, ok, "valid vector should not verify")
				if err != nil {
					// Error is acceptable for invalid vector
					t.Logf("invalid vector returned error as expected: %v", err)
				}
			}
		})
	}
}

func BenchmarkSecp256Verify(b *testing.B) {
	x, err := hex.DecodeString("2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838")
	require.NoError(b, err)
	y, err := hex.DecodeString("c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e")
	require.NoError(b, err)
	r, err := hex.DecodeString("5291deaf24659ffbbce6e3c26f6021097a74abdbb69be4fb10419c0c496c9466")
	require.NoError(b, err)
	s, err := hex.DecodeString("65d6fcf336d27cc7cdb982bb4e4ecef5827f84742f29f10abf83469270a03dc3")
	require.NoError(b, err)
	hash, err := hex.DecodeString("0eaae8641084fa979803efbfb8140732f4cdcf66c3f78a000000003c278a6b21")
	require.NoError(b, err)
	pk := make([]byte, 64)
	copy(pk, x)
	copy(pk[32:], y)
	sig := make([]byte, 64)
	copy(sig, r)
	copy(sig[32:], s)
	for b.Loop() {
		ok, vErr := SecP256Verify(hash, pk, sig)
		require.NoError(b, vErr)
		assert.True(b, ok)
	}
}

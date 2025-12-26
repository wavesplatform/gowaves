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

func unmarshalTestDataToView(t *testing.T, fs embed.FS, testFileName string) []testVectorView {
	fileData, err := fs.ReadFile(filepath.Clean(testFileName))
	require.NoError(t, err)
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
	PublicKey []byte // uncompressed SEC1, // TODO: what about 64-byte raw?
	Signature []byte
	Message   []byte
	Valid     bool
}

func appendRawPubKey(t *testing.T, out []byte, x, y string) []byte {
	const coordSize = secp256r1RawPubKeySize / 2
	out = slices.Grow(out, len(out)+secp256r1RawPubKeySize)
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
	out = slices.Grow(out, len(out)+secp256r1UncompressedPubKeySize)
	out = append(out, secp256r1UncompressedPubKeyPrefix)
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
		msgBytes, err := hex.DecodeString(tv.Msg)
		require.NoError(t, err)
		res = append(res,
			testVector{
				PublicKey: rawPK,
				Signature: sig,
				Message:   msgBytes,
				Valid:     tv.Valid,
			},
			testVector{
				PublicKey: uncompressedPK,
				Signature: sig,
				Message:   msgBytes,
				Valid:     tv.Valid,
			},
		)
	}
	return res
}

func TestSecp256r1verify(t *testing.T) {
	const testFileName = "testdata/vectors_wycheproof.jsonl"
	vectorsView := unmarshalTestDataToView(t, vectorsWycheproof, testFileName)
	vectors := transformViewsToVectors(t, vectorsView)
	for i, tv := range vectors {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			ok, err := secp256r1verify(tv.Message, tv.PublicKey, tv.Signature)
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

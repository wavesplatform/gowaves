package serialization

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// withCheckSum appends a valid checksum to the script body, so malformed bodies reach the parser.
func withCheckSum(tb testing.TB, body []byte) []byte {
	d, err := crypto.SecureHash(body)
	require.NoError(tb, err)
	r := make([]byte, 0, len(body)+4)
	r = append(r, body...)
	return append(r, d[:4]...)
}

func concat(parts ...[]byte) []byte {
	var r []byte
	for _, p := range parts {
		r = append(r, p...)
	}
	return r
}

func TestParseMalformed(t *testing.T) {
	// V1 dApp header (content type application, library version 3) with empty meta.
	dAppV1Header := []byte{0, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0}
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"V1NegativeBytesLength", []byte{1, tokenBytes, 0xff, 0xff, 0xff, 0xff}},
		{"V1HugeBytesLength", []byte{1, tokenBytes, 0x7f, 0xff, 0xff, 0xff, 1, 2, 3}},
		{"V1TruncatedBytes", []byte{1, tokenBytes, 0, 0, 0, 10, 1, 2, 3}},
		{"V1NegativeStringLength", []byte{1, tokenString, 0x80, 0, 0, 0}},
		{"V1TruncatedLength", []byte{1, tokenString, 0, 0}},
		{"V1NegativeFunctionCallArgumentsCount", []byte{1, tokenFunctionCall, functionTypeNative, 0, 0,
			0xff, 0xff, 0xff, 0xff}},
		{"V1HugeFunctionCallArgumentsCount", []byte{1, tokenFunctionCall, functionTypeNative, 0, 0,
			0x7f, 0xff, 0xff, 0xff, tokenTrue}},
		{"V1TruncatedFunctionCallArguments", []byte{1, tokenFunctionCall, functionTypeNative, 0, 0,
			0, 0, 0, 2, tokenTrue}},
		{"V1UnknownFunctionType", []byte{1, tokenFunctionCall, 7, 0, 0}},
		{"V1UnknownToken", []byte{1, 0x77}},
		{"V1NegativeDeclarationsCount", concat(dAppV1Header, []byte{0xff, 0xff, 0xff, 0xff})},
		{"V1HugeDeclarationsCount", concat(dAppV1Header, []byte{0x7f, 0xff, 0xff, 0xff})},
		{"V1NegativeFunctionsCount", concat(dAppV1Header, []byte{0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff})},
		{"V1HugeFunctionsCount", concat(dAppV1Header, []byte{0, 0, 0, 0, 0x7f, 0xff, 0xff, 0xff})},
		{"V1NegativeDeclarationArgumentsCount", concat(dAppV1Header, []byte{0, 0, 0, 1,
			declarationTypeFunction, 0, 0, 0, 1, 'f', 0xff, 0xff, 0xff, 0xff})},
		{"V1HugeDeclarationArgumentsCount", concat(dAppV1Header, []byte{0, 0, 0, 1,
			declarationTypeFunction, 0, 0, 0, 1, 'f', 0x7f, 0xff, 0xff, 0xff})},
		{"V1NegativeMetaLength", []byte{0, 2, 3, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff}},
		{"V2HugeBytesLength", []byte{6, 1, tokenBytes, 0xff, 0xff, 0xff, 0xff, 0x07}},
		{"V2TruncatedBytes", []byte{6, 1, tokenBytes, 0x10, 1, 2}},
		{"V2HugeFunctionCallArgumentsCount", []byte{6, 1, tokenFunctionCall, functionTypeNative, 0,
			0xff, 0xff, 0xff, 0xff, 0x07}},
		{"V2HugeDeclarationsCount", []byte{6, 2, 0, 0xff, 0xff, 0xff, 0xff, 0x07}},
		{"V2HugeMetaLength", []byte{6, 2, 0xff, 0xff, 0xff, 0xff, 0x07}},
		{"V2VarintOverflow", []byte{6, 1, tokenBytes, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0x01}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := withCheckSum(t, tc.body)
			assert.NotPanics(t, func() {
				tree, err := Parse(src)
				assert.Error(t, err)
				assert.Nil(t, tree)
			})
		})
	}
}

func TestParseValidCountsAtTheEdge(t *testing.T) {
	// Expression `f()` without arguments followed only by the checksum.
	src := withCheckSum(t, []byte{1, tokenFunctionCall, functionTypeUser, 0, 0, 0, 1, 'f', 0, 0, 0, 0})
	tree, err := Parse(src)
	require.NoError(t, err)
	require.NotNil(t, tree.Verifier)
	// Empty byte vector.
	src = withCheckSum(t, []byte{1, tokenBytes, 0, 0, 0, 0})
	tree, err = Parse(src)
	require.NoError(t, err)
	require.NotNil(t, tree.Verifier)
}

func fuzzRideSeeds() []string {
	return []string{
		"AQa3b8tH",
		"AwZd0cYf",
		"AQQAAAABaQAAAAAAAAAAAQQAAAABcwIAAAAGc3RyaW5nCQAAAAAAAAIJAAGkAAAAAQUAAAABaQUAAAABcwIsH74=",
		"AwoBAAAAA2FicwAAAAEAAAABaQMJAABnAAAAAgUAAAABaQAAAAAAAAAAAAUAAAABaQkBAAAAAS0AAAABBQAAAAFpCQAAAAAAAAIJ" +
			"AQAAAANhYnMAAAABAP/////////2AAAAAAAAAAAKmp8BWw==",
		"AAIDAAAAAAAAAAgIARIECgIBAQAAAAIAAAAACG1lc3NhZ2VzCQAETAAAAAICAAAABElORk8JAARMAAAAAgIAAAAEV0FSTgUAAAAD" +
			"bmlsAQAAAANtc2cAAAABAAAAAWkJAAGRAAAAAgUAAAAIbWVzc2FnZXMFAAAAAWkAAAABAAAAAWkBAAAABnRlbGxtZQAAAAIAAAAB" +
			"eAAAAAF5CQEAAAAIV3JpdGVTZXQAAAABCQAETAAAAAIJAQAAAAlEYXRhRW50cnkAAAACAgAAAAFtCQEAAAADbXNnAAAAAQUAAAAB" +
			"eAUAAAADbmlsAAAAAD8Tlfs=",
		"BgIECAISAAABAWkBA2JhcgAEA3JlcwkA/AcECQEHQWRkcmVzcwEBGgFXSbIqC+dSm+dDCCL8KamODy9oLyPQygrLAgNmb28FA25p" +
			"bAkAzAgCCQEPQXR0YWNoZWRQYXltZW50AgUEdW5pdAD0AwUDbmlsAwkAAAIFA3JlcwARBQNuaWwJAAIBAhJCYWQgcmV0dXJuZWQg" +
			"dmFsdWUAN8s8Bg==",
	}
}

func FuzzRideParse(f *testing.F) {
	for _, s := range fuzzRideSeeds() {
		src, err := base64.StdEncoding.DecodeString(s)
		require.NoError(f, err)
		_, err = Parse(src)
		require.NoError(f, err)
		f.Add(src[:len(src)-4]) // The fuzzer mutates the body, the checksum is recalculated.
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) == 0 {
			return
		}
		_, _ = Parse(withCheckSum(t, body))
	})
}

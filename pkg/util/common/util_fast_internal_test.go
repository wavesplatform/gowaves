package common

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Old implementations kept as oracles.

func replaceInvalidUtf8CharsOracle(s string) string {
	var b strings.Builder
	for _, v := range s {
		b.WriteRune(v)
	}
	return b.String()
}

func toBase58JSONOracle(b []byte) []byte {
	s := base58.Encode(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func toBase64JSONOracle(b []byte) []byte {
	s := base64.StdEncoding.EncodeToString(b)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteRune('"')
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes()
}

func TestReplaceInvalidUtf8CharsMatchesOracle(t *testing.T) {
	for _, s := range []string{
		"",
		"ascii",
		"привет мир",
		"emoji 🚀",
		string([]byte{0xE0, 0x80, 0x80}),
		string([]byte{'a', 0xff, 'b'}),
		string([]byte{0xf0, 0x9f, 0x9a}), // truncated 4-byte sequence
		"valid" + string([]byte{0xc3}),
		"\x00\x01",
	} {
		assert.Equal(t, replaceInvalidUtf8CharsOracle(s), ReplaceInvalidUtf8Chars(s), "input %q", s)
	}
}

func TestReplaceInvalidUtf8CharsNoAllocOnValid(t *testing.T) {
	s := "valid string with юникод"
	allocs := testing.AllocsPerRun(100, func() { _ = ReplaceInvalidUtf8Chars(s) })
	assert.Zero(t, allocs)
}

func TestBase58Base64JSONMatchesOracle(t *testing.T) {
	inputs := [][]byte{nil, {}, {0}, {0, 0, 1}, {0xff}, bytes.Repeat([]byte{0xab}, 32), bytes.Repeat([]byte{7}, 65)}
	for _, in := range inputs {
		require.Equal(t, toBase58JSONOracle(in), ToBase58JSON(in), "base58 input %x", in)
		require.Equal(t, toBase64JSONOracle(in), ToBase64JSON(in), "base64 input %x", in)
	}
}

func TestBase64JSONRoundTrip(t *testing.T) {
	in := []byte{0, 1, 2, 3, 250, 251}
	out := ToBase64JSON(in)
	require.Equal(t, byte('"'), out[0])
	require.Equal(t, byte('"'), out[len(out)-1])
	res, err := base64.StdEncoding.DecodeString(string(out[1 : len(out)-1]))
	require.NoError(t, err)
	assert.Equal(t, in, res)
}

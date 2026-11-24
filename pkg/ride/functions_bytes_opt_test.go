package ride //nolint:testpackage // White-box tests of the unexported bytes helpers.

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeBase16AndBase64MatchStdlib(t *testing.T) {
	sizes := []int{
		0, 1, 2, 3, 4, 5, 255, 256, 257, 383, 384, 385, 511, 512, 513, 767, 768, 769, 1023, 1024, 1025, 1536, 2049, 8192,
	}
	for _, size := range sizes {
		b := make([]byte, size)
		for i := range b {
			b[i] = byte(i*31 + 7)
		}
		assert.Equal(t, hex.EncodeToString(b), encodeBase16(b), "base16 size %d", size)
		assert.Equal(t, base64.StdEncoding.EncodeToString(b), encodeBase64(b), "base64 size %d", size)
	}
}

func TestToBase16AndBase64OutputLimits(t *testing.T) {
	r, err := toBase16Limited(limits{0, 4}, rideByteVector{1, 2})
	require.NoError(t, err)
	assert.Equal(t, rideString("0102"), r)
	_, err = toBase16Limited(limits{0, 4}, rideByteVector{1, 2, 3})
	assert.EqualError(t, err, "toBase16: string size=6 exceeds 4 bytes")
	r, err = toBase64Limited(limits{0, 4}, rideByteVector{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, rideString("AQID"), r)
	_, err = toBase64Limited(limits{0, 4}, rideByteVector{1, 2, 3, 4})
	assert.EqualError(t, err, "toBase64: string size=8 exceeds 4 bytes")
	_, err = toBase64Limited(limits{2, 4}, rideByteVector{1, 2, 3, 4})
	assert.EqualError(t, err, "toBase64: input is too long (4), limit is 2")
	r, err = toBase64Limited(limits{0, 0}, rideUnit{})
	require.NoError(t, err)
	assert.Equal(t, rideString(""), r)
}

func TestBytesToUTF8StringLimitsAndInvalidUTF8(t *testing.T) {
	for _, test := range []struct {
		b   rideByteVector
		lim int
		r   rideType
		err string
	}{
		{rideByteVector{}, 0, rideString(""), ""},
		{rideByteVector("привет"), 12, rideString("привет"), ""},
		{rideByteVector("привет"), 11, nil, "bytesToUTF8String: string size=12 exceeds 11 bytes"},
		{rideByteVector("ok\xff"), 0, nil, "invalid UTF-8 sequence"},
		{rideByteVector("\xe2\x82"), 1, nil, "invalid UTF-8 sequence"},
		{rideByteVector(strings.Repeat("😀", 10)), 40, rideString(strings.Repeat("😀", 10)), ""},
	} {
		r, err := bytesToUTF8StringLimited(limits{0, test.lim}, test.b)
		if test.err != "" {
			assert.EqualError(t, err, test.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.r, r)
	}
	// The result must not share memory with the argument.
	b := rideByteVector("abc")
	r, err := bytesToUTF8String(nil, b)
	require.NoError(t, err)
	b[0] = 'x'
	assert.Equal(t, rideString("abc"), r)
}

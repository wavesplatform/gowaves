package common

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInt64(t *testing.T) {
	zero := int64(0)
	max := int64(math.MaxInt64)
	min := int64(math.MinInt64)
	one := int64(1)
	for _, test := range []struct {
		x, y int64
		err  bool
		r    int64
	}{
		{zero, max, false, max},
		{one, max, true, zero},
		{-one, -min, true, zero},
		{zero, min, false, min},
		{one, -max, false, -int64(math.MaxInt64 - 1)},
		{-one, -max, false, min},
		{-(one * 2), -max, true, zero},
		{one, one, false, int64(2)},
		{one, -one, false, zero},
		{-one, -one, false, int64(-2)},
	} {
		r, err := AddInt64(test.x, test.y)
		if test.err {
			assert.Error(t, err, "AddInt64 did not fail with arguments causing an overflow")
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestAddUint64(t *testing.T) {
	a0 := uint64(math.MaxUint64)
	a1 := uint64(0)
	_, err := AddUint64(a0, a1)
	assert.NoError(t, err, "AddUint64 failed with arguments not causing an overflow")
	a1 = 1
	_, err = AddUint64(a1, a0)
	assert.Error(t, err, "AddUint64 did not fail with arguments causing an overflow")
}

func TestDup(t *testing.T) {
	a := []byte{1, 2, 3}
	b := Dup(a)
	require.Equal(t, a, b)

	a[0] = 0
	require.EqualValues(t, 1, b[0])
}

func TestReplaceInvalidUtf8Chars(t *testing.T) {
	s := string([]byte{0xE0, 0x80, 0x80})
	s2 := ReplaceInvalidUtf8Chars(s)
	require.Equal(t, "���", s2)
}

func TestUnixMillisUtils(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	tsMillis := ts.UnixMilli()

	require.Equal(t, tsMillis, UnixMillisFromTime(ts))
	require.Equal(t, ts.String(), UnixMillisToTime(tsMillis).String())
}

func TestBase58JSONUtils(t *testing.T) {
	var (
		expectedBytes  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 144, 255}
		expectedString = "\"1FVk6iLh9oT6aYn\""
	)
	actualString := ToBase58JSON(expectedBytes)
	require.Equal(t, expectedString, string(actualString))
	res, err := FromBase58JSON(actualString, len(expectedBytes), "TestBase58JSONUtils")
	require.NoError(t, err)
	require.Equal(t, expectedBytes, res)
}

func TestHexJSONUtils(t *testing.T) {
	var (
		expectedBytes  = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 144, 255}
		expectedString = "\"0x0001020304050607080990ff\""
	)
	actualString := ToHexJSON(expectedBytes)
	require.Equal(t, expectedString, string(actualString))
	res, err := FromHexJSON(actualString, len(expectedBytes), "TestHexJSONUtils")
	require.NoError(t, err)
	require.Equal(t, expectedBytes, res)
}

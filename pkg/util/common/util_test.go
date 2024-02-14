package common

import (
	"fmt"
	"math"
	"math/big"
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
		r, err := AddInt(test.x, test.y)
		if test.err {
			assert.Error(t, err, "AddInt did not fail with arguments causing an overflow")
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestAddUint64(t *testing.T) {
	a0 := uint64(math.MaxUint64)
	a1 := uint64(0)
	_, err := AddInt(a0, a1)
	assert.NoError(t, err, "AddInt failed with arguments not causing an overflow")
	a1 = 1
	_, err = AddInt(a1, a0)
	assert.Error(t, err, "AddInt did not fail with arguments causing an overflow")
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

func Test2CBigInt(t *testing.T) {
	tests := []struct {
		num      *big.Int
		expected []int8
	}{
		{big.NewInt(0), []int8{0}},
		{big.NewInt(1), []int8{1}},
		{big.NewInt(-1), []int8{-1}},
		{big.NewInt(127), []int8{127}},
		{big.NewInt(-127), []int8{-127}},
		{big.NewInt(128), []int8{0, -128}},
		{big.NewInt(-128), []int8{-128}},
		{big.NewInt(129), []int8{0, -127}},
		{big.NewInt(-129), []int8{-1, 127}},
		{big.NewInt(255), []int8{0, -1}},
		{big.NewInt(-255), []int8{-1, 1}},
		{big.NewInt(256), []int8{1, 0}},
		{big.NewInt(-256), []int8{-1, 0}},
		{big.NewInt(257), []int8{1, 1}},
		{big.NewInt(-257), []int8{-2, -1}},
		{big.NewInt(32767), []int8{127, -1}},
		{big.NewInt(-32767), []int8{-128, 1}},
		{big.NewInt(32768), []int8{0, -128, 0}},
		{big.NewInt(-32768), []int8{-128, 0}},
		{big.NewInt(32769), []int8{0, -128, 1}},
		{big.NewInt(-32769), []int8{-1, 127, -1}},
		{big.NewInt(65535), []int8{0, -1, -1}},
		{big.NewInt(-65535), []int8{-1, 0, 1}},
		{big.NewInt(65536), []int8{1, 0, 0}},
		{big.NewInt(-65536), []int8{-1, 0, 0}},
		{big.NewInt(65537), []int8{1, 0, 1}},
		{big.NewInt(-65537), []int8{-2, -1, -1}},
	}
	for _, test := range tests {
		name := fmt.Sprintf("number=%d", test.num)
		t.Run(name, func(t *testing.T) {
			expectedBytes := make([]byte, len(test.expected))
			for i, v := range test.expected {
				expectedBytes[i] = byte(v)
			}
			t.Run("Encode", func(t *testing.T) {
				actual := Encode2CBigInt(test.num)
				assert.Equal(t, expectedBytes, actual)
			})
			t.Run("Decode", func(t *testing.T) {
				actual := Decode2CBigInt(expectedBytes)
				assert.True(t, test.num.Cmp(actual) == 0, "expected %d, but got %d", test.num, actual)
			})
		})
	}
}

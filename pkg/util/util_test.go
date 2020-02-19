package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInt64(t *testing.T) {
	a0 := int64(math.MaxInt64)
	a1 := int64(0)
	_, err := AddInt64(a0, a1)
	assert.NoError(t, err, "AddInt64 failed with arguments not causing an overflow")
	a1 = 1
	_, err = AddInt64(a0, a1)
	assert.Error(t, err, "AddInt64 did not fail with arguments causing an overflow")
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

func TestParseDuration(t *testing.T) {
	t.Run("valid format", func(t *testing.T) {
		rs, err := ParseDuration("1d1h1m")
		require.NoError(t, err)
		require.EqualValues(t, int64(90060), rs)
	})

	t.Run("invalid format 1", func(t *testing.T) {
		_, err := ParseDuration("invalid")
		require.Error(t, err)
	})

	t.Run("invalid format 2", func(t *testing.T) {
		_, err := ParseDuration("m")
		require.Error(t, err)
	})

	t.Run("invalid format 3", func(t *testing.T) {
		_, err := ParseDuration("1")
		require.Error(t, err)
	})

	t.Run("invalid format 4", func(t *testing.T) {
		_, err := ParseDuration("1h1")
		require.Error(t, err)
	})
	t.Run("invalid format 5", func(t *testing.T) {
		_, err := ParseDuration("j")
		require.Error(t, err)
	})
	t.Run("empty", func(t *testing.T) {
		_, err := ParseDuration("")
		require.Error(t, err)
	})
}

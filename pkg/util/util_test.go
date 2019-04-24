package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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

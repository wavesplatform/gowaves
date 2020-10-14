package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDuplicateChecker_Add(t *testing.T) {
	d := NewDuplicateChecker()
	require.True(t, d.Add([]byte{1}))
	require.False(t, d.Add([]byte{1}))
	require.True(t, d.Add([]byte{2}))
	require.False(t, d.Add([]byte{2}))
}

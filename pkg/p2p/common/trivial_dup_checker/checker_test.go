package trivialdupchecker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDuplicateChecker_Add(t *testing.T) {
	d := NewDuplicateChecker()
	require.True(t, d.Add("1", []byte{1}))
	require.False(t, d.Add("1", []byte{1}))
	require.True(t, d.Add("1", []byte{2}))
	require.True(t, d.Add("2", []byte{2}))
	require.False(t, d.Add("2", []byte{2}))
}

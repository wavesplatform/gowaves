package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// incorrect test will be catched by race detector
func TestSynchronous_Go(t *testing.T) {
	i := 0
	s := NewSync()
	s.Go(func() {
		i++
	})
	require.Equal(t, 1, i)
}

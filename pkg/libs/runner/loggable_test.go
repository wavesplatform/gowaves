package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggableRunner(t *testing.T) {
	require.NotNil(t, NewLogRunner(nil))
}

func TestLog_Named(t *testing.T) {
	a := NewLogRunner(NewAsync())
	require.Len(t, a.Running(), 0)

	started := make(chan int, 1)
	ended := make(chan int, 1)

	require.Len(t, a.Running(), 0)

	done := a.Named("some", func() {
		started <- 1
		<-ended
	})

	<-started
	require.Len(t, a.Running(), 1)
	ended <- 1
	<-done
	require.Len(t, a.Running(), 0)
}

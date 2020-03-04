package runner

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggableRunner(t *testing.T) {
	require.NotNil(t, NewLogRunner(nil))
}

func TestLog_Named(t *testing.T) {
	a := NewLogRunner(NewAsync())
	require.Len(t, a.Running(), 0)

	wg := &sync.WaitGroup{}
	started := make(chan int, 1)
	ended := make(chan int, 1)

	require.Len(t, a.Running(), 0)

	wg.Add(1)
	a.Named("some", func() {
		started <- 1
		<-ended
		wg.Done()
	})

	<-started
	require.Len(t, a.Running(), 1)
	ended <- 1
	wg.Wait()
	require.Len(t, a.Running(), 0)
}

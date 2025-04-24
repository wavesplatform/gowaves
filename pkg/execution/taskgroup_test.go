package execution_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/wavesplatform/gowaves/pkg/execution"
)

func TestBasic(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Verify that the group works at all.
	var g execution.TaskGroup
	g.Run(work(25, nil))
	err := g.Wait()
	require.NoError(t, err)

	// Verify that the group can be reused.
	g.Run(work(50, nil))
	g.Run(work(75, nil))
	err = g.Wait()
	require.NoError(t, err)

	// Verify that error is propagated without an error handler.
	g.Run(work(50, errors.New("expected error")))
	err = g.Wait()
	require.Error(t, err)
}

func TestErrorsPropagation(t *testing.T) {
	defer goleak.VerifyNone(t)

	expected := errors.New("expected error")

	var g execution.TaskGroup
	g.Run(func() error { return expected })
	err := g.Wait()
	require.ErrorIs(t, err, expected)

	g.OnError(func(error) error { return nil }) // discard all error
	g.Run(func() error { return expected })
	err = g.Wait()
	require.NoError(t, err)
}

func TestCancelPropagation(t *testing.T) {
	defer goleak.VerifyNone(t)

	const numTasks = 64

	var errs []error
	g := execution.NewTaskGroup(func(err error) error {
		errs = append(errs, err) // Only collect non-nil errors and suppress them.
		return nil
	})

	errOther := errors.New("something is wrong")
	ctx, cancel := context.WithCancel(context.Background())
	var numOK int32
	for range numTasks {
		g.Run(func() error {
			d1 := randomDuration(2)
			d2 := randomDuration(2)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d1):
				return errOther
			case <-time.After(d2):
				atomic.AddInt32(&numOK, 1) // Count successful executions.
				return nil
			}
		})
	}
	cancel()

	err := g.Wait()
	require.NoError(t, err) // No captured error is expected, should be suppressed.

	var numCanceled, numOther int
	for _, e := range errs {
		switch {
		case errors.Is(e, context.Canceled):
			numCanceled++
		case errors.Is(e, errOther):
			numOther++
		default:
			require.FailNowf(t, "No error is expected", "unexpected error: %v", e)
		}
	}

	total := int(numOK) + numCanceled + numOther
	assert.Equal(t, numTasks, total)
}

func TestWaitingForFinish(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())

	failure := errors.New("failure")
	exec := func() error {
		select {
		case <-ctx.Done():
			return work(50, nil)()
		case <-time.After(60 * time.Millisecond):
			return failure
		}
	}

	var g execution.TaskGroup
	g.Run(exec)
	g.Run(exec)
	g.Run(exec)

	cancel()

	err := g.Wait()
	require.NoError(t, err)
}

func TestRegression(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("WaitRace", func(_ *testing.T) {
		ready := make(chan struct{})
		var g execution.TaskGroup
		g.Run(func() error {
			<-ready
			return nil
		})

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			err := g.Wait()
			require.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			err := g.Wait()
			require.NoError(t, err)
		}()

		close(ready)
		wg.Wait()
	})
	t.Run("WaitUnstarted", func(t *testing.T) {
		require.NotPanics(t, func() {
			var g execution.TaskGroup
			err := g.Wait()
			require.NoError(t, err)
		})
	})
}

func TestActivateRace(t *testing.T) {
	var g execution.TaskGroup
	start := make(chan struct{})
	const goroutines = 1000
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			g.Run(func() error {
				return nil
			})
		}()
	}
	close(start)
	wg.Wait()
	err := g.Wait()
	require.NoError(t, err)
}

func BenchmarkRunParallel(b *testing.B) {
	const parallelism = 100
	for n := 0; n < b.N; n++ {
		var g execution.TaskGroup
		var wg sync.WaitGroup
		wg.Add(parallelism)
		start := make(chan struct{})
		for i := 0; i < parallelism; i++ {
			go func() {
				defer wg.Done()
				<-start
				g.Run(func() error {
					return nil
				})
			}()
		}
		close(start)
		wg.Wait()
		err := g.Wait()
		require.NoError(b, err)
	}
}

func randomDuration(n int64) time.Duration {
	return time.Duration(rand.Int64N(n)) * time.Millisecond
}

// work returns an execution function that does nothing for random number of ms with [n] ms upper limit and returns err.
func work(n int64, err error) func() error {
	return func() error { time.Sleep(randomDuration(n)); return err }
}

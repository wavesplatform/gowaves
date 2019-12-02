package cancellable

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"sync"
	"testing"
	"time"
)

func TestAfter(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		ch := make(chan time.Time)
		bool := atomic.NewBool(false)
		cancel := after(ch, func() {
			bool.Store(true)
		})
		cancel()
		close(ch)
		require.False(t, bool.Load())
	})
	t.Run("completed", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		ch := make(chan time.Time)
		bool := atomic.NewBool(false)
		_ = after(ch, func() {
			bool.Store(true)
			wg.Done()
		})
		close(ch)
		wg.Wait()
		require.True(t, bool.Load())
	})
}

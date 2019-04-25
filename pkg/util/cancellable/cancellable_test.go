package cancellable

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
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
		<-time.After(1 * time.Millisecond)
		require.False(t, bool.Load())
	})
	t.Run("completed", func(t *testing.T) {
		ch := make(chan time.Time)
		bool := atomic.NewBool(false)
		_ = after(ch, func() {
			bool.Store(true)
		})
		close(ch)
		<-time.After(1 * time.Millisecond)
		require.True(t, bool.Load())
	})
}

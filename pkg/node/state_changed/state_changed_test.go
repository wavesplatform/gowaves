package state_changed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestNewStateChanged(t *testing.T) {
	require.NotNil(t, NewStateChanged())
}

func TestStateChanged_AddHandler(t *testing.T) {
	s := NewStateChanged()
	require.Equal(t, 0, s.Len())
	s.AddHandler(nil)
	require.Equal(t, 1, s.Len())
}

type notify struct {
	executed atomic.Bool
}

func (a *notify) Handle() {
	a.executed.Store(true)
}

func TestStateChanged_Notify(t *testing.T) {
	s := NewStateChanged()
	n := &notify{}
	s.AddHandler(n)
	s.Handle()
	<-time.After(1 * time.Millisecond)
	require.True(t, n.executed.Load())
}

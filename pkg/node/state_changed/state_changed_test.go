package state_changed

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	executed chan struct{}
}

func (a *notify) Handle() {
	a.executed <- struct{}{}
}

func TestStateChanged_Notify(t *testing.T) {
	ch := make(chan struct{}, 1)
	s := NewStateChanged()
	n := &notify{
		executed: ch,
	}
	s.AddHandler(n)
	s.Handle()
	<-ch
}

package lock

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRwMutex(t *testing.T) {
	s := NewRwMutex(&sync.RWMutex{})

	locked := s.Lock()
	locked.Unlock()

	rlocked := s.RLock()
	rlocked.Unlock()
}

func TestRwMutex_MapR(t *testing.T) {
	s := NewRwMutex(&sync.RWMutex{})

	rs1 := s.MapR(func() error {
		return nil
	})
	require.NoError(t, rs1)

	rs2 := s.MapR(func() error {
		return errors.New("some err")
	})
	require.Error(t, rs2)
}

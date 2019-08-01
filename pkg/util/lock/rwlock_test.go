package lock

import (
	"sync"
	"testing"
)

func TestRwMutex(t *testing.T) {
	s := NewRwMutex(&sync.RWMutex{})

	locked := s.Lock()
	locked.Unlock()

	rlocked := s.RLock()
	rlocked.Unlock()
}

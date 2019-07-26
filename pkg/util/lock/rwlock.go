package lock

import (
	"sync"
)

type RwMutex struct {
	mu *sync.RWMutex
}

func NewRwMutex(mu *sync.RWMutex) *RwMutex {
	return &RwMutex{
		mu: mu,
	}
}

func (a *RwMutex) RLock() *RLocked {
	a.mu.RLock()
	return newRLocked(a.mu)
}

func (a *RwMutex) Lock() *Locked {
	a.mu.Lock()
	return newLocked(a.mu)
}

type RLocked struct {
	mu *sync.RWMutex
}

func newRLocked(mu *sync.RWMutex) *RLocked {
	return &RLocked{
		mu: mu,
	}
}

func (a *RLocked) Unlock() {
	a.mu.RUnlock()
}

type Locked struct {
	mu *sync.RWMutex
}

func newLocked(mu *sync.RWMutex) *Locked {
	return &Locked{
		mu: mu,
	}
}

func (a *Locked) Unlock() {
	a.mu.Unlock()
}

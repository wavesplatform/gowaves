package lock

import (
	"errors"
	"sync"
	"time"
)

type Mutex struct {
	mu *sync.Mutex
}

func NewMutex() *Mutex {
	return &Mutex{
		mu: &sync.Mutex{},
	}
}

func (a *Mutex) Lock(timeout time.Duration) error {
	ch := make(chan struct{}, 1)
	locked := make(chan struct{}, 1)
	go func() {
		a.mu.Lock()
		locked <- struct{}{}
		ch <- struct{}{}
	}()
	select {
	case <-time.After(timeout):
		go func() {
			select {
			case <-locked:
				a.mu.Unlock()
			}
		}()
		return errors.New("lock timeout")
	case <-ch:
		return nil
	}
}

func (a *Mutex) Unlock() {
	a.mu.Unlock()
}

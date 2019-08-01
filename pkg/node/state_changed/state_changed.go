package state_changed

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/types"
)

type handlers []types.Handler

type StateChanged struct {
	mu       sync.Mutex
	handlers handlers
}

func NewStateChanged() *StateChanged {
	return &StateChanged{}
}

func (a *StateChanged) AddHandler(h types.Handler) {
	a.mu.Lock()
	a.handlers = append(a.handlers, h)
	a.mu.Unlock()
}

func (a *StateChanged) Handle() {
	a.mu.Lock()
	for _, h := range a.handlers {
		go h.Handle()
	}
	a.mu.Unlock()
}

func (a *StateChanged) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.handlers)
}

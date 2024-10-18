package runner

import (
	"maps"
	"sync"
)

type LogRunner interface {
	Named(name string, f func()) (done <-chan struct{})
	Running() map[string]int
}

type log struct {
	mu      sync.Mutex
	r       Runner
	running map[string]int
}

func NewLogRunner(r Runner) *log {
	return &log{
		r:       r,
		running: make(map[string]int),
	}
}

func (a *log) addNamed(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running[name] += 1
}

func (a *log) removeNamed(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running[name] -= 1
	if a.running[name] == 0 {
		delete(a.running, name)
	}
}

func (a *log) Named(name string, f func()) <-chan struct{} {
	done := make(chan struct{})
	a.r.Go(func() {
		defer close(done)
		a.addNamed(name)
		defer a.removeNamed(name)
		f()
	})
	return done
}

func (a *log) Running() map[string]int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return maps.Clone(a.running)
}

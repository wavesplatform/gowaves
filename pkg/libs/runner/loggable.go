package runner

import "sync"

type LogRunner interface {
	Named(name string, f func())
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
	a.running[name] += 1
	a.mu.Unlock()
}

func (a *log) removeNamed(name string) {
	a.mu.Lock()
	a.running[name] -= 1
	if a.running[name] == 0 {
		delete(a.running, name)
	}
	a.mu.Unlock()
}

func (a *log) Named(name string, f func()) {
	a.r.Go(func() {
		a.addNamed(name)
		defer a.removeNamed(name)
		f()
	})
}

func (a *log) Running() map[string]int {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[string]int, len(a.running))
	for k, v := range a.running {
		out[k] = v
	}
	return out
}

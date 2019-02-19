package utils

import "sync"

type SpawnedPeers struct {
	addrs map[string]struct{}
	mu    sync.Mutex
}

func NewSpawnedPeers() *SpawnedPeers {
	return &SpawnedPeers{
		addrs: make(map[string]struct{}),
	}
}

func (a *SpawnedPeers) Add(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.addrs[addr] = struct{}{}
}

func (a *SpawnedPeers) Exists(addr string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.addrs[addr]
	return ok
}

func (a *SpawnedPeers) Delete(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.addrs, addr)
}

func (a *SpawnedPeers) GetAll() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for k := range a.addrs {
		out = append(out, k)
	}
	return out
}

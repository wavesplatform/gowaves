package utils

import (
	"sort"
	"sync"
	"time"
)

type Response struct {
	Time  string
	Value Count
}

type ByTime []Response

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time < a[j].Time }

type Count struct {
	UniqueTransaction uint64
	TransactionsSend  uint64
}

type Counter struct {
	mu                     sync.Mutex
	resendTransactionCount map[string]Count
	interrupt              chan struct{}
}

func NewCounter() *Counter {
	c := &Counter{
		resendTransactionCount: make(map[string]Count),
		interrupt:              make(chan struct{}),
	}
	go c.clearBackground(c.interrupt, 1*time.Hour)
	return c
}

// collect how many transaction we send (or tried to send) in an hour
func (a *Counter) IncEachTransaction() {
	a.mu.Lock()
	t := time.Now().Format("2006-01-02T15")
	cnt := a.resendTransactionCount[t]
	cnt.TransactionsSend += 1
	a.resendTransactionCount[t] = cnt
	a.mu.Unlock()
}

// collect how many unique transaction we received
func (a *Counter) IncUniqueTransaction() {
	a.mu.Lock()
	t := time.Now().Format("2006-01-02T15")
	cnt := a.resendTransactionCount[t]
	cnt.UniqueTransaction += 1
	a.resendTransactionCount[t] = cnt
	a.mu.Unlock()
}

func (a *Counter) Get() []Response {
	out := make([]Response, 0, len(a.resendTransactionCount))
	a.mu.Lock()
	for k, v := range a.resendTransactionCount {
		out = append(out, Response{
			Time:  k,
			Value: v,
		})
	}
	a.mu.Unlock()
	sort.Sort(ByTime(out))
	return out
}

func (a *Counter) clearBackground(interrupt chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.clear(100)
		case <-interrupt:
			return
		}
	}
}

func (a *Counter) clear(count int) {
	responses := a.Get()
	if len(responses) > count {
		last := responses[len(responses)-1]
		a.mu.Lock()
		delete(a.resendTransactionCount, last.Time)
		a.mu.Unlock()
	}
}

func (a *Counter) Stop() {
	close(a.interrupt)
}

package utils

import (
	"context"
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
}

func NewCounter(ctx context.Context) *Counter {
	c := &Counter{
		resendTransactionCount: make(map[string]Count),
	}
	go c.clearBackground(ctx, time.Hour)
	return c
}

// collect how many transaction we send (or tried to send) in an hour
func (a *Counter) IncEachTransaction() {
	a.mu.Lock()
	cnt := a.resendTransactionCount[time.Now().Format("2006-01-02T15")]
	cnt.TransactionsSend += 1
	a.resendTransactionCount[time.Now().Format("2006-01-02T15")] = cnt
	a.mu.Unlock()
}

// collect how many unique transaction we received
func (a *Counter) IncUniqueTransaction() {
	a.mu.Lock()
	cnt := a.resendTransactionCount[time.Now().Format("2006-01-02T15")]
	cnt.UniqueTransaction += 1
	a.resendTransactionCount[time.Now().Format("2006-01-02T15")] = cnt
	a.mu.Unlock()
}

func (a *Counter) Get() []Response {
	var out []Response
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

func (a *Counter) clearBackground(ctx context.Context, duration time.Duration) {
	for {
		select {
		case <-time.After(duration):
			a.clear(100)
		case <-ctx.Done():
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

package utils

import (
	"sync"
	"time"
)

type Count struct {
	UniqueTransaction uint64
	TransactionsSend  uint64
}

type Counter struct {
	mu                       sync.Mutex
	resendedTransactionCount map[string]Count
}

func NewCounter() *Counter {
	return &Counter{
		resendedTransactionCount: make(map[string]Count),
	}
}

// collect how many transaction we send (or tried to send) in an hour
func (a *Counter) IncEachTransaction() {
	a.mu.Lock()
	cnt := a.resendedTransactionCount[time.Now().Format("2006-01-02T15")]
	cnt.TransactionsSend += 1
	a.resendedTransactionCount[time.Now().Format("2006-01-02T15")] = cnt
	a.mu.Unlock()
}

// collect how many unique transaction we received
func (a *Counter) IncUniqueTransaction() {
	a.mu.Lock()
	cnt := a.resendedTransactionCount[time.Now().Format("2006-01-02T15")]
	cnt.UniqueTransaction += 1
	a.resendedTransactionCount[time.Now().Format("2006-01-02T15")] = cnt
	a.mu.Unlock()
}

func (a *Counter) Get() map[string]Count {
	out := make(map[string]Count)
	a.mu.Lock()
	for k, v := range a.resendedTransactionCount {
		out[k] = v
	}
	a.mu.Unlock()
	return out
}

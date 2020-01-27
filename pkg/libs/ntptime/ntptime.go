package ntptime

import (
	"context"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

type inner interface {
	Query(addr string) (*ntp.Response, error)
}

type ntpInner struct {
}

func (a ntpInner) Query(addr string) (*ntp.Response, error) {
	return ntp.Query(addr)
}

type ntpTimeImpl struct {
	mu     sync.RWMutex
	err    error
	offset time.Duration
	addr   string
	inner  inner
}

func New(addr string) *ntpTimeImpl {
	return new(addr, ntpInner{})
}

func new(addr string, inner inner) *ntpTimeImpl {
	a := &ntpTimeImpl{
		mu:    sync.RWMutex{},
		addr:  addr,
		inner: inner,
	}
	tm, err := inner.Query(addr)
	if err != nil {
		a.err = err
	} else {
		a.offset = tm.ClockOffset
		a.err = nil
	}
	return a
}

func (a *ntpTimeImpl) Run(ctx context.Context, duration time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(duration):
			a.mu.Lock()
			tm, err := a.inner.Query(a.addr)
			if err != nil {
				a.err = err
			} else {
				a.offset = tm.ClockOffset
				a.err = nil
			}
			a.mu.Unlock()
		}
	}
}

func (a *ntpTimeImpl) Now() (time.Time, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return time.Now().Add(a.offset), a.err
}

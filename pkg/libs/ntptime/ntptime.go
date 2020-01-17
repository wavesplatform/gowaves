package ntptime

import (
	"context"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

type ntpTimeImpl struct {
	mu     sync.RWMutex
	err    error
	offset time.Duration
	addr   string
}

func New(addr string) *ntpTimeImpl {
	a := &ntpTimeImpl{
		mu:   sync.RWMutex{},
		addr: addr,
	}
	tm, err := ntp.Query(addr)
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
			tm, err := ntp.Query("0.ru.pool.ntp.org")
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

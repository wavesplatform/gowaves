package ntptime

import (
	"context"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"go.uber.org/zap"
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
	offset time.Duration
	addr   string
	inner  inner
}

func New(addr string) (*ntpTimeImpl, error) {
	return new(addr, ntpInner{})
}

func TryNew(addr string, tries uint) (*ntpTimeImpl, error) {
	return tryNew(addr, tries, ntpInner{})
}

func tryNew(addr string, tries uint, inner inner) (*ntpTimeImpl, error) {
	if tries == 0 {
		return new(addr, inner)
	}
	rs, err := new(addr, inner)
	if err != nil {
		return tryNew(addr, tries-1, inner)
	}
	return rs, nil
}

func new(addr string, inner inner) (*ntpTimeImpl, error) {
	a := &ntpTimeImpl{
		mu:    sync.RWMutex{},
		addr:  addr,
		inner: inner,
	}
	tm, err := inner.Query(addr)
	if err != nil {
		return nil, err
	}
	a.offset = tm.ClockOffset
	return a, nil
}

func (a *ntpTimeImpl) Run(ctx context.Context, duration time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(duration):
			tm, err := a.inner.Query(a.addr)
			if err != nil {
				zap.S().Debug("ntpTimeImpl Run: ", err)
				continue
			}
			a.mu.Lock()
			a.offset = tm.ClockOffset
			a.mu.Unlock()
		}
	}
}

func (a *ntpTimeImpl) Now() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return time.Now().Add(a.offset)
}

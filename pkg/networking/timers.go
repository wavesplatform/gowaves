package networking

import (
	"sync"
	"time"
)

type timerPool struct {
	p *sync.Pool
}

func newTimerPool() *timerPool {
	const initialTimerInterval = time.Hour * 1e6
	return &timerPool{
		p: &sync.Pool{
			New: func() any {
				timer := time.NewTimer(initialTimerInterval)
				timer.Stop()
				return timer
			},
		},
	}
}

func (p *timerPool) Get() *time.Timer {
	t, ok := p.p.Get().(*time.Timer)
	if !ok {
		panic("invalid type of item in TimerPool")
	}
	return t
}

func (p *timerPool) Put(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	p.p.Put(t)
}

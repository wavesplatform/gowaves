package cancellable

import (
	"context"
	"sync/atomic"
	"time"
)

func After(duration time.Duration, callback func()) context.CancelFunc {
	return after(time.After(duration), callback)
}

func after(ch <-chan time.Time, callback func()) context.CancelFunc {
	cancelCh := make(chan struct{})
	flag := uint32(0)

	go func() {
		select {
		case <-cancelCh:
			return
		case <-ch:
			if atomic.LoadUint32(&flag) == 0 {
				callback()
			}
		}
	}()

	return func() {
		atomic.StoreUint32(&flag, 1)
		close(cancelCh)
	}
}

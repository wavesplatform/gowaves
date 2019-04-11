package cancellable

import (
	"context"
	"time"
)

func After(duration time.Duration, callback func(c context.Context)) context.CancelFunc {
	return after(time.After(duration), callback)
}

func after(ch <-chan time.Time, callback func(c context.Context)) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			callback(ctx)
		}
	}()

	return cancel
}

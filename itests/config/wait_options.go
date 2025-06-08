package config

import (
	"context"
	"time"

	"github.com/ccoveille/go-safecast"
)

const defaultBlockGenerationTimeout = 30 * time.Second

type WaitParams struct {
	Ctx     context.Context
	Timeout time.Duration
}

func DefaultWaitParams() *WaitParams {
	return &WaitParams{
		Ctx:     context.Background(),
		Timeout: defaultBlockGenerationTimeout,
	}
}

func NewWaitParams(opts ...WaitOption) *WaitParams {
	params := DefaultWaitParams()
	for _, opt := range opts {
		opt(params)
	}
	return params
}

// WaitOption is a functional option type that allows to set additional parameters of waiting operations.
type WaitOption func(*WaitParams)

// WaitWithContext sets the context for waiting operations.
func WaitWithContext(ctx context.Context) WaitOption {
	return func(params *WaitParams) {
		params.Ctx = ctx
	}
}

// WaitWithTimeout sets the timeout for waiting operations.
func WaitWithTimeout(timeout time.Duration) WaitOption {
	return func(params *WaitParams) {
		if timeout <= 0 {
			timeout = defaultBlockGenerationTimeout
		} else {
			params.Timeout = timeout
		}
	}
}

// WaitWithTimeoutInBlocks sets the timeout for waiting operations based on the number of blocks to wait for.
func WaitWithTimeoutInBlocks(blocks uint64) WaitOption {
	return func(params *WaitParams) {
		n, err := safecast.ToInt64(blocks)
		if err != nil {
			panic("invalid number of blocks: " + err.Error())
		}
		if n <= 0 {
			params.Timeout = defaultBlockGenerationTimeout
		} else {
			params.Timeout = time.Duration(n) * defaultBlockGenerationTimeout
		}
	}
}

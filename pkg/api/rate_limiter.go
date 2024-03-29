package api

import (
	"github.com/pkg/errors"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

func createRateLimiter(opts *RateLimiterOptions) (throttled.HTTPRateLimiterCtx, error) {
	store, err := memstore.New(opts.MemoryCacheSize)
	if err != nil {
		return throttled.HTTPRateLimiterCtx{},
			errors.Wrapf(
				err,
				"createRateLimiter: failed to create memstore with capacity %d",
				opts.MemoryCacheSize,
			)
	}

	quota := throttled.RateQuota{
		MaxRate:  throttled.PerSec(opts.MaxRequestsPerSecond),
		MaxBurst: opts.MaxBurst,
	}

	rateLimiter, err := throttled.NewGCRARateLimiterCtx(throttled.WrapStoreWithContext(store), quota)
	if err != nil {
		return throttled.HTTPRateLimiterCtx{},
			errors.Wrap(err, "createRateLimiter: can't create rate limiter")
	}

	httpRateLimiter := throttled.HTTPRateLimiterCtx{
		RateLimiter: rateLimiter,
		VaryBy: &throttled.VaryBy{
			RemoteAddr: true,
		},
	}

	return httpRateLimiter, nil
}

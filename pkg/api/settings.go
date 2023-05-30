package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	DefaultMaxConnections       = 128
	DefaultRateLimiterCacheSize = 64 * 1024 // 64 KB
	DefaultRateLimiterRPS       = 1
	DefaultRateLimiterBurst     = 1
)

const (
	cacheSizeKey = "cache"
	rpsKey       = "rps"
	burstKey     = "burst"
)

type RunOptions struct {
	RateLimiterOpts      *RateLimiterOptions
	LogHttpRequestOpts   bool
	CollectMetrics       bool
	UseRealIPMiddleware  bool
	RequestIDMiddleware  bool
	EnableHeartbeatRoute bool
	RouteNotFoundHandler func(w http.ResponseWriter, r *http.Request)
	MaxConnections       int
	EnableMetaMaskAPI    bool
	EnableMetaMaskAPILog bool
}

type RateLimiterOptions struct {
	MemoryCacheSize      int
	MaxRequestsPerSecond int
	MaxBurst             int
}

func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		RateLimiterOpts: &RateLimiterOptions{
			MemoryCacheSize:      DefaultRateLimiterCacheSize,
			MaxRequestsPerSecond: DefaultRateLimiterRPS,
			MaxBurst:             DefaultRateLimiterBurst,
		},
		LogHttpRequestOpts:   false,
		EnableHeartbeatRoute: true,
		UseRealIPMiddleware:  true,
		RequestIDMiddleware:  true,
		CollectMetrics:       true,
		RouteNotFoundHandler: func(w http.ResponseWriter, r *http.Request) {
			zap.S().Debugf("NodeApi not found %+v, %s", r, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		},
		MaxConnections:       DefaultMaxConnections,
		EnableMetaMaskAPI:    false,
		EnableMetaMaskAPILog: false,
	}
}

func NewRateLimiterOptionsFromString(s string) (*RateLimiterOptions, error) {
	opt := &RateLimiterOptions{
		MemoryCacheSize:      DefaultRateLimiterCacheSize,
		MaxRequestsPerSecond: DefaultRateLimiterRPS,
		MaxBurst:             DefaultRateLimiterBurst,
	}
	query, err := url.ParseQuery(strings.TrimSpace(s))
	if err != nil {
		return nil, errors.Wrap(err, "invalid rate limiter options")
	}
	cacheSize, ok, err := extractFirstIntValue(query, cacheSizeKey)
	if ok {
		if err != nil {
			return nil, errors.Wrap(err, "invalid rate limiter options")
		}
		opt.MemoryCacheSize = cacheSize
	}
	rps, ok, err := extractFirstIntValue(query, rpsKey)
	if ok {
		if err != nil {
			return nil, errors.Wrap(err, "invalid rate limiter options")
		}
		opt.MaxRequestsPerSecond = rps
	}
	burst, ok, err := extractFirstIntValue(query, burstKey)
	if ok {
		if err != nil {
			return nil, errors.Wrap(err, "invalid rate limiter options")
		}
		opt.MaxBurst = burst
	}
	return opt, nil
}

func extractFirstIntValue(query url.Values, key string) (int, bool, error) {
	values, ok := query[key]
	if !ok {
		return 0, false, nil
	}
	if len(values) < 1 {
		return 0, true, errors.Errorf("no value for key '%s'", key)
	}
	v, err := strconv.ParseInt(strings.TrimSpace(values[0]), 10, 32)
	if err != nil {
		return 0, true, errors.Wrapf(err, "invalid value for key '%s'", key)
	}
	return int(v), true, nil
}

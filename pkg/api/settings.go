package api

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
		RateLimiterOpts:      DefaultRateLimiterOptions(),
		LogHttpRequestOpts:   false,
		EnableHeartbeatRoute: true,
		UseRealIPMiddleware:  true,
		RequestIDMiddleware:  true,
		CollectMetrics:       true,
		RouteNotFoundHandler: func(w http.ResponseWriter, r *http.Request) {
			slog.Debug("NodeApi not found", "request", r, "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		},
		MaxConnections:       DefaultMaxConnections,
		EnableMetaMaskAPI:    false,
		EnableMetaMaskAPILog: false,
	}
}

func DefaultRateLimiterOptions() *RateLimiterOptions {
	return &RateLimiterOptions{
		MemoryCacheSize:      DefaultRateLimiterCacheSize,
		MaxRequestsPerSecond: DefaultRateLimiterRPS,
		MaxBurst:             DefaultRateLimiterBurst,
	}
}

func NewRateLimiterOptionsFromString(s string) (*RateLimiterOptions, error) {
	opt := DefaultRateLimiterOptions()
	query, err := url.ParseQuery(strings.TrimSpace(s))
	if err != nil {
		return nil, errors.Wrap(err, "invalid rate limiter options")
	}
	cacheSize, err := extractFirstIntValue(query, cacheSizeKey, DefaultRateLimiterCacheSize)
	if err != nil {
		return nil, errors.Wrap(err, "invalid rate limiter options")
	}
	opt.MemoryCacheSize = cacheSize
	rps, err := extractFirstIntValue(query, rpsKey, DefaultRateLimiterRPS)
	if err != nil {
		return nil, errors.Wrap(err, "invalid rate limiter options")
	}
	opt.MaxRequestsPerSecond = rps
	burst, err := extractFirstIntValue(query, burstKey, DefaultRateLimiterBurst)
	if err != nil {
		return nil, errors.Wrap(err, "invalid rate limiter options")
	}
	opt.MaxBurst = burst
	return opt, nil
}

func extractFirstIntValue(query url.Values, key string, dft int) (int, error) {
	values, ok := query[key]
	if !ok {
		return dft, nil
	}
	if len(values) < 1 {
		return 0, errors.Errorf("no value for key '%s'", key)
	}
	v, err := strconv.ParseInt(strings.TrimSpace(values[0]), 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for key '%s'", key)
	}
	return int(v), nil
}

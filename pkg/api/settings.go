package api

import (
	"net/http"

	"go.uber.org/zap"
)

const (
	DefaultMaxConnections         = 128
	DefaultRateLimiterStorageSize = 64 * 1024 // 64 KB
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
			MemoryCacheSize:      DefaultRateLimiterStorageSize,
			MaxRequestsPerSecond: 1,
			MaxBurst:             1,
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

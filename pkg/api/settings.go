package api

import (
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

//const (
//	rateLimiterMemoryCacheSize      = 65_536
//	rateLimiterMaxRequestsPerSecond = 100
//	rateLimiterMaxBurst             = 5
//)

type RunOptions struct {
	RateLimiterOpts      *RateLimiterOptions
	LogHttpRequestOpts   bool
	CollectMetrics       bool
	UseRealIPMiddleware  bool
	EnableHeartbeatRoute bool
	RouteNotFoundHandler func(w http.ResponseWriter, r *http.Request)
}

type RateLimiterOptions struct {
	MemoryCacheSize      int
	MaxRequestsPerSecond int
	MaxBurst             int
}

func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		RateLimiterOpts:      nil,
		LogHttpRequestOpts:   false,
		EnableHeartbeatRoute: true,
		UseRealIPMiddleware:  true,
		CollectMetrics:       true,
		RouteNotFoundHandler: func(w http.ResponseWriter, r *http.Request) {
			zap.S().Debugf("NodeApi not found %+v, %s", r, r.URL.Path)
			if r.Method == http.MethodPost {
				// TODO(nickeskov): it looks vulnerable (memory overflow)
				rs, err := ioutil.ReadAll(r.Body)
				zap.S().Debugf("NodeApi not found post body: %s %+v", string(rs), err)
			}
			w.WriteHeader(http.StatusNotFound)
		},
	}
}

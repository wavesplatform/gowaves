package api

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"go.uber.org/zap"
	"net/http"
)

func createRateLimiter(opts *RateLimiterOptions) (throttled.HTTPRateLimiter, error) {
	store, err := memstore.New(opts.MemoryCacheSize)
	if err != nil {
		return throttled.HTTPRateLimiter{},
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

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return throttled.HTTPRateLimiter{},
			errors.Wrap(err, "createRateLimiter: can't create rate limiter")
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy: &throttled.VaryBy{
			RemoteAddr: true,
		},
	}

	return httpRateLimiter, nil
}

func (a *NodeApi) routes(opts *RunOptions) (chi.Router, error) {
	r := chi.NewRouter()

	// TODO(nickskov): it's correct middleware apply order?
	if opts.UseRealIPMiddleware {
		// nickeskov: for nginx/haproxy specific headers
		r.Use(middleware.RealIP)
	}
	if opts.CollectMetrics {
		r.Use(chiHttpApiGeneralMetricsMiddleware)
	}
	if opts.RateLimiterOpts != nil {
		rateLimiter, err := createRateLimiter(opts.RateLimiterOpts)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		r.Use(rateLimiter.RateLimit)
	}
	if opts.LogHttpRequestOpts != nil {
		r.Use(middleware.RequestID, LoggerMiddleware(zap.L()))
	}
	if opts.RouteNotFoundHandler != nil {
		r.NotFound(opts.RouteNotFoundHandler)
	}

	if opts.EnableHeartbeatRoute {
		r.Get("/debug/health", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK")); err != nil {
				zap.S().Errorf("Can't write 'OK' to ResponseWriter: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
	}

	// nickeskov: json api
	r.Group(func(r chi.Router) {
		r.Use(jsonContentTypeMiddleware)

		r.Get("/addresses", a.Addresses)

		r.Get("/blocks/last", a.BlocksLast)
		r.Get("/blocks/height", a.BlockHeight)
		r.Get("/blocks/first", a.BlocksFirst)
		r.Get("/blocks/at/{height:\\d+}", a.BlockAt)
		r.Get("/blocks/score/at/{id:\\d+}", a.BlockScoreAt)
		r.Get("/blocks/id/{id}", a.BlockIDAt)
		r.Get("/blocks/generators", a.BlocksGenerators)
		r.Post("/blocks/rollback", RollbackToHeight(a.app))
		r.Get("/pool/transactions", a.poolTransactions)
		r.Get("/transactions/unconfirmed/size", a.unconfirmedSize)
		r.Route("/peers", func(r chi.Router) {
			r.Get("/known", a.PeersAll)
			r.Get("/connected", a.PeersConnected)
			r.Post("/connect", a.PeersConnect)
			r.Get("/suspended", a.PeersSuspended)
			r.Get("/spawned", a.PeersSpawned)
		})
		r.Get("/miner/info", a.MinerInfo)
		r.Post("/transactions/broadcast", a.TransactionsBroadcast)

		r.Post("/wallet/load", WalletLoadKeys(a.app))
		r.Get("/wallet/accounts", a.WalletAccounts)

		r.Get("/node/processes", a.nodeProcesses)
		r.Get("/debug/stateHash/{height:\\d+}", a.stateHash)

		// enable or disable history sync
		//r.Get("/debug/sync/{enabled:\\d+}", a.DebugSyncEnabled)
	})

	return r, nil
}

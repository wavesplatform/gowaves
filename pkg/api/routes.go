package api

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/semrush/zenrpc/v2"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/api/metamask"
)

type HandleErrorFunc func(w http.ResponseWriter, r *http.Request, err error)
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func toHTTPHandlerFunc(handler HandlerFunc, errorHandler HandleErrorFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		err := handler(writer, request)
		if err != nil {
			errorHandler(writer, request, err)
		}
	}
}

func (a *NodeApi) routes(opts *RunOptions) (chi.Router, error) {
	r := chi.NewRouter()

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
	if opts.RequestIDMiddleware {
		r.Use(middleware.RequestID)
	}
	if opts.LogHttpRequestOpts {
		r.Use(createLoggerMiddleware(zap.L()))
	}
	if opts.RouteNotFoundHandler != nil {
		r.NotFound(opts.RouteNotFoundHandler)
	}

	// nickeskov: middlewares and custom handlers
	errHandler := NewErrorHandler(zap.L())
	checkAuthMiddleware := createCheckAuthMiddleware(a.app, errHandler.Handle)

	wrapper := func(handlerFunc HandlerFunc) http.HandlerFunc {
		return toHTTPHandlerFunc(handlerFunc, errHandler.Handle)
	}

	if opts.EnableHeartbeatRoute {
		r.Get("/go/node/healthz", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK")); err != nil {
				zap.S().Errorf("Can't write 'OK' to ResponseWriter: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
	}

	// nickeskov: go node routes
	r.Route("/go", func(r chi.Router) {
		r.Route("/blocks", func(r chi.Router) {
			r.Get("/score/at/{id:\\d+}", wrapper(a.BlockScoreAt))
			r.Get("/id/{id}", wrapper(a.BlockIDAt))
			r.Get("/generators", wrapper(a.BlocksGenerators))
			r.Get("/first", wrapper(a.BlocksFirst))
			r.Get("/snapshot/at/{height:\\d+}", wrapper(a.BlocksSnapshotAt))
		})

		r.Route("/peers", func(r chi.Router) {
			r.Get("/known", wrapper(a.PeersKnown))
			r.Get("/spawned", wrapper(a.PeersSpawned))
		})

		r.Route("/wallet", func(r chi.Router) {
			r.Get("/accounts", wrapper(a.WalletAccounts))

			rAuth := r.With(checkAuthMiddleware)

			rAuth.Post("/load", wrapper(WalletLoadKeys(a.app)))
		})
		r.Route("/debug", func(r chi.Router) {
			r.Get("/snapshotStateHash/{height:\\d+}", wrapper(a.snapshotStateHash))
		})

		r.Get("/miner/info", wrapper(a.GoMinerInfo))
		r.Get("/pool/transactions", wrapper(a.poolTransactions))
	})

	// nickeskov: json api
	r.Group(func(r chi.Router) {
		r.Route("/blocks", func(r chi.Router) {
			r.Get("/last", wrapper(a.BlocksLast))
			r.Get("/height", wrapper(a.BlockHeight))
			r.Get("/height/{id}", wrapper(a.BlockHeightByID))
			r.Get("/at/{height}", wrapper(a.BlockAt))
			r.Get("/{id}", wrapper(a.BlockIDAt))

			r.Route("/headers", func(r chi.Router) {
				r.Get("/last", wrapper(a.BlocksHeadersLast))
				r.Get("/at/{height:\\d+}", wrapper(a.BlocksHeadersAt))
				r.Get("/{id}", wrapper(a.BlockHeadersID))
				r.Get("/seq/{from:\\d+}/{to:\\d+}", wrapper(a.BlocksHeadersSeqFromTo))
			})
		})

		r.Route("/assets", func(r chi.Router) {
			r.Get("/details/{id}", wrapper(a.AssetsDetailsByID))
			r.Get("/details", wrapper(a.AssetsDetailsByIDsGet))
			r.Post("/details", wrapper(a.AssetsDetailsByIDsPost))
		})

		r.Route("/addresses", func(r chi.Router) {
			r.Get("/", wrapper(a.Addresses))
			r.Get("/balance/{address}", wrapper(a.WavesRegularBalanceByAddress))
		})

		r.Route("/alias", func(r chi.Router) {
			r.Get("/by-alias/{alias}", wrapper(a.AddrByAlias))
			r.Get("/by-address/{address}", wrapper(a.AliasesByAddr))
		})

		r.Route("/transactions", func(r chi.Router) {
			r.Get("/unconfirmed/size", wrapper(a.unconfirmedSize))
			r.Get("/info/{id}", wrapper(a.TransactionInfo))
			r.Post("/broadcast", wrapper(a.TransactionsBroadcast))
		})

		r.Route("/peers", func(r chi.Router) {
			r.Get("/all", wrapper(a.PeersAll))
			r.Get("/connected", wrapper(a.PeersConnected))
			r.Get("/suspended", wrapper(a.PeersSuspended))
			r.Get("/blacklisted", wrapper(a.PeersBlackListed))

			rAuth := r.With(checkAuthMiddleware)

			rAuth.Post("/connect", wrapper(a.PeersConnect))
			rAuth.Post("/clearblacklist", wrapper(a.PeersClearBlackList))
		})

		r.Route("/debug", func(r chi.Router) {
			r.Get("/stateHash/{height:\\d+}", wrapper(a.stateHash))
			r.Get("/stateHash/last", wrapper(a.stateHashLast))

			rAuth := r.With(checkAuthMiddleware)

			rAuth.Post("/print", wrapper(a.debugPrint))
			rAuth.Post("/rollback", wrapper(a.RollbackToHeight))
			rAuth.Post("/rollback-to/{id}", wrapper(a.RollbackTo))
		})
		r.Route("/node", func(r chi.Router) {
			r.Get("/version", wrapper(a.version))
			r.Get("/status", wrapper(a.NodeStatus))
		})

		r.Route("/wallet", func(r chi.Router) {
			rAuth := r.With(checkAuthMiddleware)

			rAuth.Get("/seed", wrapper(a.walletSeed))
		})

		r.Route("/eth", func(r chi.Router) {
			r.Get("/abi/{address}", wrapper(a.EthereumDAppABI))
			if opts.EnableMetaMaskAPI {
				service := metamask.NewRPCService(&a.app.services)
				rpc := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true, AllowCORS: true})
				if opts.EnableMetaMaskAPILog {
					rpc.Use(metamask.APILogMiddleware)
				}
				rpc.Register("", service)
				r.Handle("/", rpc)
			}
		})

		r.Route("/blockchain", func(r chi.Router) {
			r.Get("/rewards", wrapper(a.blockchainRewards))
			r.Get("/rewards/{height}", wrapper(a.blockchainRewardsAtHeight))
		})

		// enable or disable history sync
		//r.Get("/debug/sync/{enabled:\\d+}", a.DebugSyncEnabled)
	})

	return r, nil
}

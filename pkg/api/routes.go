package api

import (
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func (a *NodeApi) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(
		chiHttpApiGeneralMetricsMiddleware,
		panicMiddleware,
		jsonContentTypeMiddleware,
	)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		zap.S().Debugf("NodeApi not found %+v, %s", r, r.URL.Path)
		if r.Method == http.MethodPost {
			rs, err := ioutil.ReadAll(r.Body)
			zap.S().Debugf("NodeApi not found post body: %s %+v", string(rs), err)
		}
		// TODO(nickeskov): send json response
		w.WriteHeader(http.StatusNotFound)
	})

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

	r.Get("/debug/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			zap.S().Errorf("Can't write 'OK' to ResponseWriter: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	return r
}

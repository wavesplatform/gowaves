package api

import "github.com/go-chi/chi"

func (a *NodeApi) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/blocks/last", a.BlocksLast)
	r.Get("/blocks/height", a.BlockHeight)
	r.Get("/blocks/first", a.BlocksFirst)
	r.Get("/blocks/at/{height:\\d+}", a.BlockAt)
	r.Get("/blocks/score/at/{id:\\d+}", a.BlockScoreAt)
	r.Get("/blocks/signature/{signature}", a.BlockSignatureAt)
	r.Get("/blocks/generators", a.BlocksGenerators)
	r.Post("/blocks/rollback", RollbackToHeight(a.app))
	r.Get("/pool/transactions", a.poolTransactions)
	r.Route("/peers", func(r chi.Router) {
		r.Get("/known", a.PeersAll)
		r.Get("/connected", a.PeersConnected)
		r.Post("/connect", a.PeersConnect)
		r.Get("/suspended", a.PeersSuspended)
		r.Get("/spawned", a.PeersSpawned)
	})
	r.Get("/miner/info", a.Minerinfo)
	r.Post("/transactions/broadcast", a.TransactionsBroadcast)

	r.Post("/wallet/load", WalletLoadKeys(a.app))

	r.Get("/node/processes", a.nodeProcesses)
	// enable or disable history sync
	//r.Get("/debug/sync/{enabled:\\d+}", a.DebugSyncEnabled)

	return r
}

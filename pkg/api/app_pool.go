package api

func (a *App) PoolTransactions() int {
	return a.utx.Count()
}

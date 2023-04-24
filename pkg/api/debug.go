package api

func (a *App) DebugSyncEnabled(enabled bool) {
	a.sync.SetEnabled(enabled)
}

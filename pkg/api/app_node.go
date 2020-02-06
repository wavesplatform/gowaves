package api

func (a *App) NodeProcesses() map[string]int {
	return a.services.LoggableRunner.Running()
}

package api

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/versioning"
)

type nodeVersion struct {
	Version string `json:"version"`
}

func (a *App) version() nodeVersion {
	return nodeVersion{Version: fmt.Sprintf("Gowaves %s", versioning.Version)}
}

func (a *App) NodeProcesses() map[string]int {
	return a.services.LoggableRunner.Running()
}

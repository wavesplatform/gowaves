package api

import "github.com/wavesplatform/gowaves/pkg/proto"

func (a *App) RollbackToHeight(apiKey string, height proto.Height) error {
	err := a.checkAuth(apiKey)
	if err != nil {
		return err
	}
	defer a.state.Mutex().Lock().Unlock()
	return a.state.RollbackToHeight(height)
}

package api

import "github.com/wavesplatform/gowaves/pkg/proto"

func (a *App) RollbackToHeight(height proto.Height) error {
	defer a.state.Mutex().Lock().Unlock()
	return a.state.RollbackToHeight(height)
}

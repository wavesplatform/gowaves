package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// TODO Here should be internal message with rollback action
func (a *App) RollbackToHeight(apiKey string, height proto.Height) error {
	return errors.New("api method disabled")
}

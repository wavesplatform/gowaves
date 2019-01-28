package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
)

type handlerParams struct {
	ctx                       context.Context
	address                   string
	connection                conn.Connection
	remote                    remote
	receiveFromRemoteCallback ReceiveFromRemoteCallback
	parent                    Parent
	pool                      conn.Pool
}

func handle(params handlerParams) {
	for {
		select {
		case <-params.ctx.Done():
			_ = params.connection.Close()
			return

		case bts := <-params.remote.fromCh:
			params.receiveFromRemoteCallback(bts, params.address, params.parent.ResendToParentCh, params.pool)

		case err := <-params.remote.errCh:
			out := InfoMessage{
				ID:    params.address,
				Value: err,
			}
			params.parent.ParentInfoChan <- out
		}
	}
}

package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
)

type handlerParams struct {
	ctx                       context.Context
	id                        string
	connection                conn.Connection
	remote                    remote
	receiveFromRemoteCallback ReceiveFromRemoteCallback
	parent                    Parent
	pool                      conn.Pool
}

// for handle doesn't matter outgoing or incoming connection, it just send and receive messages
func handle(params handlerParams) {
	for {
		select {
		case <-params.ctx.Done():
			_ = params.connection.Close()
			return

		case bts := <-params.remote.fromCh:
			params.receiveFromRemoteCallback(bts, params.id, params.parent.MessageCh, params.pool)

		case err := <-params.remote.errCh:
			out := InfoMessage{
				ID:    params.id,
				Value: err,
			}
			params.parent.InfoCh <- out
		}
	}
}

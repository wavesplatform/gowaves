package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"

	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

func bytesToMessage(b []byte, id string, resendTo chan ProtoMessage, pool bytespool.Pool) {
	defer func() {
		pool.Put(b)
	}()

	m, err := proto.UnmarshalMessage(b)
	if err != nil {
		zap.L().Error("can't unmarshal network message", zap.Error(err))
		return
	}

	mess := ProtoMessage{
		ID:      id,
		Message: m,
	}

	select {
	case resendTo <- mess:
	default:
		zap.L().Warn("failed to resend to parent, channel is full", zap.String("id", id))
	}
}

type handlerParams struct {
	ctx        context.Context
	id         string
	connection conn.Connection
	remote     remote
	parent     Parent
	pool       bytespool.Pool
}

// for handle doesn't matter outgoing or incoming connection, it just send and receive messages
func handle(params handlerParams) {
	for {
		select {
		case <-params.ctx.Done():
			_ = params.connection.Close()
			return

		case bts := <-params.remote.fromCh:
			bytesToMessage(bts, params.id, params.parent.MessageCh, params.pool)

		case err := <-params.remote.errCh:
			out := InfoMessage{
				ID:    params.id,
				Value: err,
			}
			params.parent.InfoCh <- out
		}
	}
}

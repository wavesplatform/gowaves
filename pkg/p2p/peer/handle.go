package peer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"

	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
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
		zap.L().Warn("failed to resend to Parent, channel is full", zap.String("ID", id))
	}
}

type HandlerParams struct {
	Ctx        context.Context
	ID         string
	Connection conn.Connection
	Remote     Remote
	Parent     Parent
	Pool       bytespool.Pool
}

// for Handle doesn't matter outgoing or incoming Connection, it just send and receive messages
func Handle(params HandlerParams) error {
	for {
		select {
		case <-params.Ctx.Done():
			_ = params.Connection.Close()
			return params.Ctx.Err()

		case bts := <-params.Remote.FromCh:
			bytesToMessage(bts, params.ID, params.Parent.MessageCh, params.Pool)

		case err := <-params.Remote.ErrCh:
			out := InfoMessage{
				ID:    params.ID,
				Value: err,
			}
			params.Parent.InfoCh <- out
		}
	}
}

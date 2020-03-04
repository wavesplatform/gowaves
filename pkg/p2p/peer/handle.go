package peer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

func bytesToMessage(b []byte, id string, resendTo chan ProtoMessage, pool bytespool.Pool, p Peer) error {
	defer func() {
		pool.Put(b)
	}()

	m, err := proto.UnmarshalMessage(b)
	if err != nil {
		return err
	}

	mess := ProtoMessage{
		ID:      p,
		Message: m,
	}

	select {
	case resendTo <- mess:
	default:
		zap.S().Debugf("failed to resend to Parent, channel is full: %s, %T", id, m)
	}
	return nil
}

type HandlerParams struct {
	Ctx        context.Context
	ID         string
	Connection conn.Connection
	Remote     Remote
	Parent     Parent
	Pool       bytespool.Pool
	Peer       Peer
}

// for Handle doesn't matter outgoing or incoming Connection, it just send and receive messages
func Handle(params HandlerParams) error {
	for {
		select {
		case <-params.Ctx.Done():
			_ = params.Connection.Close()
			return errors.Wrap(params.Ctx.Err(), "Handle")

		case bts := <-params.Remote.FromCh:
			err := bytesToMessage(bts, params.ID, params.Parent.MessageCh, params.Pool, params.Peer)
			if err != nil {
				out := InfoMessage{
					Peer:  params.Peer,
					Value: err,
				}
				select {
				case params.Parent.InfoCh <- out:
				default:
				}
			}

		case err := <-params.Remote.ErrCh:
			out := InfoMessage{
				Peer:  params.Peer,
				Value: err,
			}
			params.Parent.InfoCh <- out
		}
	}
}

package peer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type DuplicateChecker interface {
	Add([]byte) (isNew bool)
}

func bytesToMessage(b []byte, d DuplicateChecker, resendTo chan ProtoMessage, pool bytespool.Pool, p Peer) error {
	defer func() {
		pool.Put(b)
	}()

	if d != nil {
		isNew := d.Add(b)
		if !isNew {
			return nil
		}
	}

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
		zap.S().Debugf("Failed to resend to Parent, channel is full: %s, %T", m, m)
	}
	return nil
}

type HandlerParams struct {
	Ctx              context.Context
	ID               string
	Connection       conn.Connection
	Remote           Remote
	Parent           Parent
	Pool             bytespool.Pool
	Peer             Peer
	DuplicateChecker DuplicateChecker
}

// Handle sends and receives messages no matter outgoing or incoming connection.
func Handle(params HandlerParams) error {
	for {
		select {
		case <-params.Ctx.Done():
			_ = params.Connection.Close()
			//TODO: On Done() Err() contains only Canceled or DeadlineExceeded.
			// Actually, those errors are only logged in different places and not used to alter behavior.
			// Consider removing wrapping. For now, if context was canceled no error is passed by.
			if errors.Is(params.Ctx.Err(), context.Canceled) {
				return nil
			}
			return errors.Wrap(params.Ctx.Err(), "Handle")

		case bts := <-params.Remote.FromCh:
			err := bytesToMessage(bts, params.DuplicateChecker, params.Parent.MessageCh, params.Pool, params.Peer)
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

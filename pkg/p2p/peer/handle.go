package peer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/p2p/conn"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type DuplicateChecker interface {
	Add([]byte) (isNew bool)
}

func bytesToMessage(data []byte, d DuplicateChecker, resendTo chan ProtoMessage, p Peer) error {
	if d != nil {
		isNew := d.Add(data)
		if !isNew {
			return nil
		}
	}

	m, err := proto.UnmarshalMessage(data)
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
		zap.S().Debugf("[%s] Failed to resend message of type '%T' because upstream channel is full", p.ID(), m)
	}
	return nil
}

type HandlerParams struct {
	Ctx              context.Context
	ID               string
	Connection       conn.Connection
	Remote           Remote
	Parent           Parent
	Peer             Peer
	DuplicateChecker DuplicateChecker
}

// Handle sends and receives messages no matter outgoing or incoming connection.
// TODO: caller should be responsible for closing network connection
func Handle(params HandlerParams) error {
	var errSentToParent bool // if errSentToParent is true then we need to wait params.Ctx cancellation
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

		case bb := <-params.Remote.FromCh:
			if !errSentToParent {
				err := bytesToMessage(bb.Bytes(), params.DuplicateChecker, params.Parent.MessageCh, params.Peer)
				if err != nil {
					out := InfoMessage{Peer: params.Peer, Value: &InternalErr{Err: err}}
					params.Parent.InfoCh <- out
					errSentToParent = true
				}
			}
			bytebufferpool.Put(bb) // bytes buffer should be returned to the pool in any execution branch

		case err := <-params.Remote.ErrCh:
			if !errSentToParent {
				out := InfoMessage{Peer: params.Peer, Value: &InternalErr{Err: err}}
				params.Parent.InfoCh <- out
				errSentToParent = true
			}
		}
	}
}

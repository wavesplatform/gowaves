package peer

import (
	"context"
	"log/slog"
	"sync"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func bytesToMessage(data []byte, resendTo chan ProtoMessage, p Peer, logger *slog.Logger) error {
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
		logger.Debug("Failed to resend message because upstream channel is full", slog.Any("peer", p.ID()),
			logging.Type(m))
	}
	return nil
}

type peerOnceCloser struct {
	Peer
	once       sync.Once
	errOnClose error
}

func newPeerOnceCloser(p Peer) *peerOnceCloser {
	return &peerOnceCloser{Peer: p}
}

func (p *peerOnceCloser) Close() error {
	p.once.Do(func() {
		p.errOnClose = p.Peer.Close()
	})
	return p.errOnClose
}

// Handle sends and receives messages no matter outgoing or incoming connection.
// Handle consumes provided peer parameter and closes it when the function ends.
func Handle(ctx context.Context, peer Peer, parent Parent, remote Remote, logger, dl *slog.Logger) error {
	peer = newPeerOnceCloser(peer) // wrap peer in order to prevent multiple peer.Close() calls
	defer func(p Peer) {
		if err := p.Close(); err != nil {
			slog.Error("Failed to close peer", slog.Any("direction", p.Direction()),
				slog.Any("peer", p.ID()), logging.Error(err))
		}
	}(peer)
	connectedMsg := InfoMessage{Peer: peer, Value: &Connected{Peer: peer}}
	parent.InfoCh <- connectedMsg // notify parent about new connection

	var errSentToParent bool // if errSentToParent is true then we need to wait ctx cancellation
	for {
		select {
		case <-ctx.Done(): // context is unique for each peer, so when passed 'peer' arg is closed, context is canceled
			//TODO: On Done() Err() contains only Canceled or DeadlineExceeded.
			// Actually, those errors are only logged in different places and not used to alter behavior.
			// Consider removing wrapping. For now, if context was canceled no error is passed by.
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return errors.Wrap(ctx.Err(), "Handle")

		case bb := <-remote.FromCh:
			if !errSentToParent {
				dl.Debug("Receiving from network", "peer", peer.ID(), "data", proto.B64Bytes(bb.Bytes()))
				err := bytesToMessage(bb.Bytes(), parent.MessageCh, peer, logger)
				if err != nil {
					out := InfoMessage{Peer: peer, Value: &InternalErr{Err: err}}
					parent.InfoCh <- out
					errSentToParent = true
				}
			}
			bytebufferpool.Put(bb) // bytes buffer should be returned to the pool in any execution branch

		case err := <-remote.ErrCh:
			if !errSentToParent {
				out := InfoMessage{Peer: peer, Value: &InternalErr{Err: err}}
				parent.InfoCh <- out
				errSentToParent = true
			}
		}
	}
}

package peer

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
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
func Handle(ctx context.Context, peer Peer, parent Parent, remote Remote, duplicateChecker DuplicateChecker) error {
	peer = newPeerOnceCloser(peer) // wrap peer in order to prevent multiple peer.Close() calls
	defer func(p Peer) {
		if err := p.Close(); err != nil {
			zap.S().Errorf("Failed to close '%s' peer '%s': %v", p.Direction(), p.ID(), err)
		}
	}(peer)
	connectedMsg := InfoMessage{Peer: peer, Value: &Connected{Peer: peer}}
	parent.InfoCh <- connectedMsg // notify parent about new connection

	var errSentToParent bool // if errSentToParent is true then we need to wait ctx cancellation
	for {
		select {
		case <-ctx.Done():
			//TODO: On Done() Err() contains only Canceled or DeadlineExceeded.
			// Actually, those errors are only logged in different places and not used to alter behavior.
			// Consider removing wrapping. For now, if context was canceled no error is passed by.
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return errors.Wrap(ctx.Err(), "Handle")

		case bb := <-remote.FromCh:
			if !errSentToParent {
				err := bytesToMessage(bb.Bytes(), duplicateChecker, parent.MessageCh, peer)
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

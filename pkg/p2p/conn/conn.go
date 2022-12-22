package conn

import (
	"context"
	"io"
	"net"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const maxMessageSize = 2 << (10 * 2)

type Dialer func(network string, addr string) (net.Conn, error)

type Connection interface {
	io.Closer
	Conn() net.Conn
	SendClosed() bool
	ReceiveClosed() bool
}

// send to remote
func sendToRemote(conn io.Writer, ctx context.Context, toRemoteCh chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case bts := <-toRemoteCh:
			_, err := conn.Write(bts)
			if err != nil {
				return err
			}
		}
	}
}

// SkipFilter indicates that the network message should be skipped.
type SkipFilter func(proto.Header) bool

func receiveFromRemote(conn io.Reader, fromRemoteCh chan *bytebufferpool.ByteBuffer, skip SkipFilter, addr string) error {
	for {
		header := proto.Header{}
		if _, err := header.ReadFrom(conn); err != nil {
			return errors.Wrap(err, "failed to read header")
		}

		if skip(header) {
			if _, err := io.CopyN(io.Discard, conn, int64(header.PayloadLength)); err != nil {
				return errors.Wrap(err, "failed to skip payload")
			}
			continue
		}
		// received too big message, probably it's an error
		if l := int(header.HeaderLength() + header.PayloadLength); l > maxMessageSize {
			return errors.Errorf("received too long message, size=%d > max=%d", l, maxMessageSize)
		}
		b := bytebufferpool.Get()
		// put header before payload
		if _, err := header.WriteTo(b); err != nil {
			bytebufferpool.Put(b)
			return errors.Wrap(err, "failed to write header into buff")
		}
		// then read all message to remaining buffer
		if _, err := io.CopyN(b, conn, int64(header.PayloadLength)); err != nil {
			bytebufferpool.Put(b)
			return errors.Wrap(err, "failed to read payload into buffer")
		}
		select {
		case fromRemoteCh <- b:
		default:
			bytebufferpool.Put(b)
			zap.S().Debugf("[%s] Failed to send bytes from network to upstream channel because it's full", addr)
		}
	}
}

type ConnectionImpl struct {
	sendClosed    *atomic.Bool
	receiveClosed *atomic.Bool
	conn          net.Conn
	cancel        context.CancelFunc
}

func (a *ConnectionImpl) Close() error {
	a.cancel()
	return a.conn.Close()
}

func (a *ConnectionImpl) Conn() net.Conn {
	return a.conn
}

func (a *ConnectionImpl) SendClosed() bool {
	return a.sendClosed.Load()
}

func (a *ConnectionImpl) ReceiveClosed() bool {
	return a.receiveClosed.Load()
}

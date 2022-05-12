package conn

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"

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

func handleErr(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default:
		zap.L().Warn("can't send error, chan is full", zap.Error(err))
	}
}

// send to remote
func sendToRemote(closed *atomic.Bool, conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error) {
	defer closed.Store(true)
	for {
		select {
		case <-ctx.Done():
			return
		case bts := <-toRemoteCh:
			_, err := conn.Write(bts)
			if err != nil {
				handleErr(err, errCh)
			}
		}
	}
}

// nonRecoverableError returns `true` if we can't recover from such error.
// On non-recoverable errors we should close connection and exit.
func nonRecoverableError(err error) bool {
	if err == nil {
		return false
	}
	// for more details with net.ErrClosed see https://github.com/golang/go/issues/4373
	return errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed)
}

// SkipFilter indicates that the network message should be skipped.
type SkipFilter func(proto.Header) bool

func receiveFromRemote(stopped *atomic.Bool, conn io.Reader, fromRemoteCh chan *bytebufferpool.ByteBuffer, errCh chan error, skip SkipFilter, addr string) {
	defer stopped.Store(true)
	for {
		header := proto.Header{}
		_, err := header.ReadFrom(conn)
		if err != nil {
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			continue
		}

		if skip(header) {
			_, err = io.CopyN(ioutil.Discard, conn, int64(header.PayloadLength))
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			continue
		}
		// received too long message than we expected, probably it is error, discard
		if int(header.HeaderLength()+header.PayloadLength) > maxMessageSize {
			_, err = io.CopyN(ioutil.Discard, conn, int64(header.PayloadLength))
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			continue
		}
		b := bytebufferpool.Get()
		// put header before payload
		if _, err := header.WriteTo(b); err != nil {
			bytebufferpool.Put(b)
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			handleErr(err, errCh)
			continue
		}
		// then read all message to remaining buffer
		//hl := header.HeaderLength()
		pl := int64(header.PayloadLength)
		_, err = io.CopyN(b, conn, pl)
		//_, err = proto.ReadPayload(b.B[hl:hl+pl], conn)
		if err != nil {
			bytebufferpool.Put(b)
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			handleErr(err, errCh)
			continue
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

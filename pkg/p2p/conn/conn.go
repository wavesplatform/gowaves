package conn

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

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
// we should close connection and exit
func nonRecoverableError(err error) bool {
	if err != nil {
		if err == io.EOF {
			return true
		}
		if strings.Contains(err.Error(), "use of closed network connection") {
			return true
		}
	}
	return false
}

// if returned type is `true`, then network message will be skipped.
type SkipFilter func(proto.Header) bool

func recvFromRemote(stopped *atomic.Bool, pool bytespool.Pool, conn io.Reader, fromRemoteCh chan []byte, errCh chan error, skip SkipFilter) {
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
		messageIsTooLong := int(header.HeaderLength()+header.PayloadLength) > pool.BytesLen()
		if messageIsTooLong {
			_, err = io.CopyN(ioutil.Discard, conn, int64(header.PayloadLength))
			if nonRecoverableError(err) {
				handleErr(err, errCh)
				return
			}
			continue
		}
		b := pool.Get()
		// put header before payload
		header.Copy(b)
		// then read all message to remaining buffer
		hl := header.HeaderLength()
		pl := header.PayloadLength
		_, err = proto.ReadPayload(b[hl:hl+pl], conn)
		if err != nil {
			pool.Put(b)
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
			pool.Put(b)
			zap.L().Warn("recvFromRemote send bytes failed, chan is full")
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

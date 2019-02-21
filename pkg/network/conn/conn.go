package conn

import (
	"context"
	"io"
	"net"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type Pool interface {
	Get() []byte
	Put([]byte)
	BytesLen() int
}

type Dialer func(network string, addr string) (net.Conn, error)

type Connection interface {
	io.Closer
	Conn() net.Conn
}

func handleErr(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default:
		zap.S().Warnf("can't send error, chan is full, error is %s", err)
	}
}

// send to remote
func sendToRemote(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error) {
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

func recvFromRemote(pool Pool, conn io.Reader, fromRemoteCh chan []byte, errCh chan error) {
	for {
		b := pool.Get()
		n, err := proto.ReadPacket(b, conn)
		// we got message, that may be greater than out max network message
		// better log this
		if n == int64(pool.BytesLen()) {
			zap.S().Warnf("incoming message(%d bytes) may be greater than expected (%d bytes) %s", n, pool.BytesLen())
		}

		if err != nil {
			if err == io.EOF {
				pool.Put(b)
				return
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				pool.Put(b)
				return
			}
			handleErr(err, errCh)
			pool.Put(b)
			continue
		}
		select {
		case fromRemoteCh <- b:
		default:
			pool.Put(b)
			zap.S().Warnf("recvFromRemote send bytes failed, chan is full")
		}
	}
}

type ConnectionImpl struct {
	conn   net.Conn
	cancel context.CancelFunc
}

func (a *ConnectionImpl) Close() error {
	a.cancel()
	return a.conn.Close()
}

func (a *ConnectionImpl) Conn() net.Conn {
	return a.conn
}

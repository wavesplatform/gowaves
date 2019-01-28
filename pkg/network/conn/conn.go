package conn

import (
	"context"
	"io"
	"net"
	"strings"

	"go.uber.org/zap"
)

type Pool interface {
	Get() []byte
	Put([]byte)
}

// size of incoming message, 2 megabytes
const size = 1024 * 1024 * 2

type Dialer func(network string, addr string) (net.Conn, error)

type Connection interface {
	io.Closer
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

func recvFromRemote(pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error) {

	for {
		b := pool.Get()
		n, err := reader.Read(b)
		// we got message, that may be greater than out max network message
		// better log this
		if n == size {
			zap.S().Warnf("incoming message(%d bytes) may be greater than expected (%d bytes)", n, size)
		}

		if err != nil {
			if err == io.EOF {
				return
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			handleErr(err, errCh)
			continue
		}
		select {
		case fromRemoteCh <- b:
		default:
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

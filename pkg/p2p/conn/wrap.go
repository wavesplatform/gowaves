package conn

import (
	"bufio"
	"context"
	"io"
	"net"

	. "github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"go.uber.org/atomic"
)

func WrapConnection(conn net.Conn, pool Pool, toRemoteCh chan []byte, fromRemoteCh chan []byte, errCh chan error, skip SkipFilter) Connection {
	return wrapConnection(wrapParams{
		conn:         conn,
		pool:         pool,
		toRemoteCh:   toRemoteCh,
		fromRemoteCh: fromRemoteCh,
		errCh:        errCh,
		sendFunc:     sendToRemote,
		receiveFunc:  receiveFromRemote,
		skip:         skip,
	})
}

type wrapParams struct {
	conn         net.Conn
	pool         Pool
	toRemoteCh   chan []byte
	fromRemoteCh chan []byte
	errCh        chan error
	sendFunc     func(closed *atomic.Bool, conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error)
	receiveFunc  func(closed *atomic.Bool, pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error, skip SkipFilter, addr string)
	skip         SkipFilter
}

func wrapConnection(params wrapParams) *ConnectionImpl {
	ctx, cancel := context.WithCancel(context.Background())

	impl := &ConnectionImpl{
		cancel:        cancel,
		conn:          params.conn,
		receiveClosed: atomic.NewBool(false),
		sendClosed:    atomic.NewBool(false),
	}

	bufReader := bufio.NewReader(params.conn)

	go params.receiveFunc(impl.receiveClosed, params.pool, bufReader, params.fromRemoteCh, params.errCh, params.skip, params.conn.RemoteAddr().String())
	go params.sendFunc(impl.sendClosed, params.conn, ctx, params.toRemoteCh, params.errCh)

	return impl
}

package conn

import (
	"bufio"
	"context"
	. "github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"io"
	"net"
)

func WrapConnection(conn net.Conn, pool Pool, toRemoteCh chan []byte, fromRemoteCh chan []byte, errCh chan error, skip SkipFilter) Connection {
	return wrapConnection(wrapParams{
		conn:         conn,
		pool:         pool,
		toRemoteCh:   toRemoteCh,
		fromRemoteCh: fromRemoteCh,
		errCh:        errCh,
		sendFunc:     sendToRemote,
		recvFunc:     recvFromRemote,
		skip:         skip,
	})
}

type wrapParams struct {
	conn         net.Conn
	pool         Pool
	toRemoteCh   chan []byte
	fromRemoteCh chan []byte
	errCh        chan error
	sendFunc     func(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error)
	recvFunc     func(pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error, skip SkipFilter)
	skip         SkipFilter
}

func wrapConnection(params wrapParams) Connection {
	ctx, cancel := context.WithCancel(context.Background())

	impl := &ConnectionImpl{
		cancel: cancel,
		conn:   params.conn,
	}

	bufReader := bufio.NewReader(params.conn)

	go params.recvFunc(params.pool, bufReader, params.fromRemoteCh, params.errCh, params.skip)
	go params.sendFunc(params.conn, ctx, params.toRemoteCh, params.errCh)

	return impl
}

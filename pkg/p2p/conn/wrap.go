package conn

import (
	"bufio"
	"context"
	"io"
	"net"

	"github.com/valyala/bytebufferpool"
	"go.uber.org/atomic"
)

func WrapConnection(ctx context.Context, conn net.Conn, toRemoteCh chan []byte, fromRemoteCh chan *bytebufferpool.ByteBuffer, errCh chan error, skip SkipFilter) Connection {
	return wrapConnection(ctx, wrapParams{
		conn:         conn,
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
	toRemoteCh   chan []byte
	fromRemoteCh chan *bytebufferpool.ByteBuffer
	errCh        chan error
	sendFunc     func(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error)
	receiveFunc  func(reader io.Reader, fromRemoteCh chan *bytebufferpool.ByteBuffer, errCh chan error, skip SkipFilter, addr string)
	skip         SkipFilter
}

func wrapConnection(ctx context.Context, params wrapParams) *ConnectionImpl {
	ctx, cancel := context.WithCancel(ctx)

	receiveClosed := atomic.NewBool(false)
	go func() {
		defer receiveClosed.Store(true)
		bufReader := bufio.NewReader(params.conn)
		params.receiveFunc(bufReader, params.fromRemoteCh, params.errCh, params.skip, params.conn.RemoteAddr().String())
	}()

	sendClosed := atomic.NewBool(false)
	go func() {
		defer sendClosed.Store(true)
		params.sendFunc(params.conn, ctx, params.toRemoteCh, params.errCh)
	}()

	return &ConnectionImpl{
		cancel:        cancel,
		conn:          params.conn,
		receiveClosed: receiveClosed,
		sendClosed:    sendClosed,
	}
}

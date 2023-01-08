package conn

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
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
	sendFunc     func(ctx context.Context, conn deadlineWriter, toRemoteCh chan []byte, now func() time.Time) error
	receiveFunc  func(reader deadlineReader, fromRemoteCh chan *bytebufferpool.ByteBuffer, skip SkipFilter, addr string, now func() time.Time) error
	skip         SkipFilter
}

func wrapConnection(ctx context.Context, params wrapParams) *ConnectionImpl {
	ctx, cancel := context.WithCancel(ctx)
	var (
		once             = new(sync.Once)
		notifyAboutError = func(err error) {
			if err == nil {
				return
			}
			once.Do(func() { // error handler should be notified exactly once
				select {
				case <-ctx.Done():
					return // nothing to do, context canceled, connection was closed manually
				default:
					// some error happened in receiveFunc or sendFunc
				}
				params.errCh <- err // notify error handler that connection should be closed
				cancel()            // cancel connection context manually (need for select inside sendFunc)
			})
		}
	)

	now := func() time.Time { return time.Now() }

	receiveClosed := atomic.NewBool(false)
	go func() {
		defer receiveClosed.Store(true)
		defer cancel() // ensure cleanup (mostly in case if the parent context has been canceled)
		bufReader := &struct {
			io.Reader
			readDeadlineSetter
		}{bufio.NewReader(params.conn), params.conn}
		remoteAddr := params.conn.RemoteAddr().String()
		err := params.receiveFunc(bufReader, params.fromRemoteCh, params.skip, remoteAddr, now)
		if err != nil {
			notifyAboutError(errors.Wrapf(err, "receiveFunc failed with addr %q", remoteAddr))
		}
	}()

	sendClosed := atomic.NewBool(false)
	go func() {
		defer sendClosed.Store(true)
		defer cancel() // ensure cleanup (mostly in case if the parent context has been canceled)
		err := params.sendFunc(ctx, params.conn, params.toRemoteCh, now)
		if err != nil {
			remoteAddr := params.conn.RemoteAddr().String()
			notifyAboutError(errors.Wrapf(err, "sendFunc failed with addr %q", remoteAddr))
		}
	}()

	return &ConnectionImpl{
		cancel:        cancel,
		conn:          params.conn,
		receiveClosed: receiveClosed,
		sendClosed:    sendClosed,
	}
}

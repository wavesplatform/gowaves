package conn

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"go.uber.org/zap"
)

func sendfunc(i *uint64) func(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error) {
	return func(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error) {
		defer atomic.AddUint64(i, 1)
		sendToRemote(conn, ctx, toRemoteCh, errCh)
	}
}

func recvfunc(i *uint64) func(pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error) {
	return func(pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error) {
		defer atomic.AddUint64(i, 1)
		recvFromRemote(pool, reader, fromRemoteCh, errCh)
	}
}

// check on calling close method spawned goroutines exited
func TestConnectionImpl_Close(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := listener.Accept()
			require.NoError(t, err)
			_ = conn.Close()
		}
	}()

	c, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	pool := bytespool.NewBytesPool(32, 2*1024*1024)

	counter := uint64(0)

	params := wrapParams{

		conn:         c,
		pool:         pool,
		toRemoteCh:   nil,
		fromRemoteCh: make(chan []byte, 2),
		errCh:        nil,
		sendFunc:     sendfunc(&counter),
		recvFunc:     recvfunc(&counter),
	}

	conn := wrapConnection(params)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	<-time.After(10 * time.Millisecond)
	assert.EqualValues(t, 2, atomic.LoadUint64(&counter))
}

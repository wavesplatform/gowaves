package conn

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

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

	params := wrapParams{
		conn:         c,
		pool:         pool,
		toRemoteCh:   nil,
		fromRemoteCh: make(chan []byte, 2),
		errCh:        make(chan error, 1),
		sendFunc:     sendToRemote,
		recvFunc:     recvFromRemote,
	}

	conn := wrapConnection(params)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	<-time.After(10 * time.Millisecond)
	assert.True(t, conn.sendClosed.Load())
	assert.True(t, conn.receiveClosed.Load())
}

func TestRecvFromRemote_Transaction(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	messBytes := byte_helpers.TransferWithSig.MessageBytes
	pool := bytespool.NewNoOpBytesPool(len(messBytes))
	fromRemoteCh := make(chan []byte, 2)

	recvFromRemote(atomic.NewBool(false), pool, bytes.NewReader(messBytes), fromRemoteCh, make(chan error, 1), func(headerBytes proto.Header) bool {
		return false
	})

	retBytes := <-fromRemoteCh
	assert.Equal(t, messBytes, retBytes)
}

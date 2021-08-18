package conn

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"
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
	params := wrapParams{
		conn:         c,
		toRemoteCh:   nil,
		fromRemoteCh: make(chan *bytebufferpool.ByteBuffer, 2),
		errCh:        make(chan error, 1),
		sendFunc:     sendToRemote,
		receiveFunc:  receiveFromRemote,
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
	fromRemoteCh := make(chan *bytebufferpool.ByteBuffer, 2)

	receiveFromRemote(atomic.NewBool(false), bytes.NewReader(messBytes), fromRemoteCh, make(chan error, 1), func(headerBytes proto.Header) bool {
		return false
	}, "test")

	bb := <-fromRemoteCh
	assert.Equal(t, messBytes, bb.Bytes())
}

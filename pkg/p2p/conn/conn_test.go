package conn

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"go.uber.org/zap"
)

// check on calling close method spawned goroutines exited
func TestConnectionImpl_Close(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	sendData := []byte{42, 21}

	go func() {
		for {
			conn, err := listener.Accept()
			require.NoError(t, err)
			out := make([]byte, len(sendData))
			_, err = conn.Read(out)
			require.NoError(t, err)
			require.Equal(t, sendData, out)
			_ = conn.Close()
		}
	}()

	c, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	params := wrapParams{
		conn:         c,
		toRemoteCh:   make(chan []byte, 1),
		fromRemoteCh: make(chan *bytebufferpool.ByteBuffer, 1),
		errCh:        make(chan error, 2),
		sendFunc:     sendToRemote,
		receiveFunc:  receiveFromRemote,
	}

	conn := wrapConnection(context.Background(), params)
	require.NoError(t, err)
	params.toRemoteCh <- sendData
	<-time.After(10 * time.Millisecond)
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

	now := time.Now()
	nowFn := func() time.Time { return now }
	filter := func(headerBytes proto.Header) bool { return false }

	var rdr *mockDeadlineReader
	rdr = &mockDeadlineReader{
		ReadFunc: bytes.NewReader(messBytes).Read,
		SetReadDeadlineFunc: func(tm time.Time) error {
			if len(rdr.SetReadDeadlineCalls())%2 == 1 {
				assert.Equal(t, now.Add(MaxConnIdleIODuration), tm)
			} else {
				assert.Equal(t, now.Add(maxConnIODurationPerMessage), tm)
			}
			return nil
		},
	}

	err := receiveFromRemote(rdr, fromRemoteCh, filter, "test", nowFn)
	require.ErrorIs(t, err, io.EOF)
	assert.Len(t, rdr.SetReadDeadlineCalls(), 3)

	bb := <-fromRemoteCh
	assert.Equal(t, messBytes, bb.Bytes())
}

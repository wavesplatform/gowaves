package conn

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

// test that we are receiving bytes
func TestWrapConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := listener.Accept()
			require.NoError(t, err)
			_, _ = conn.Write(byte_helpers.TransferWithSig.MessageBytes)
			_ = conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	ch := make(chan *bytebufferpool.ByteBuffer, 1)
	wrapped := WrapConnection(context.Background(), conn, nil, ch, nil, func(bytes proto.Header) bool {
		return false
	}, slog.New(slog.DiscardHandler))

	select {
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("no value arrived in 10ms")
	case m := <-ch:
		assert.Equal(t, byte_helpers.TransferWithSig.MessageBytes, m.Bytes())
		require.NoError(t, wrapped.Close())
	}
}

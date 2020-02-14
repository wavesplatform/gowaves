package conn

import (
	"net"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
	"go.uber.org/zap"
)

//test that we receiving bytes
func TestWrapConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

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

	pool := bytespool.NewBytesPool(1, len(byte_helpers.TransferWithSig.MessageBytes))
	ch := make(chan []byte, 1)
	wrapped := WrapConnection(conn, pool, nil, ch, nil, func(bytes proto.Header) bool {
		return false
	})

	select {
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("no value arrived in 10ms")
	case m := <-ch:
		assert.Equal(t, byte_helpers.TransferWithSig.MessageBytes, m)
		require.NoError(t, wrapped.Close())
	}
}

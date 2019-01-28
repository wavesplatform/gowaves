package conn

import (
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"go.uber.org/zap"
	"net"
	"testing"
	"time"
)

func TestWrapConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := listener.Accept()
			require.NoError(t, err)
			_, _ = conn.Write([]byte("aaaaa"))
			_ = conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	pool := bytespool.NewBytesPool(1, 1024)
	ch := make(chan []byte, 1)
	wrapped := WrapConnection(conn, pool, nil, ch, nil)

	<-time.After(100 * time.Millisecond)
	assert.Equal(t, []byte("aaaaa"), (<-ch)[:5])
	require.NoError(t, wrapped.Close())
}

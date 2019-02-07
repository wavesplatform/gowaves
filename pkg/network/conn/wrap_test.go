package conn

import (
	"encoding/binary"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"go.uber.org/zap"
	"net"
	"testing"
	"time"
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
			out := make([]byte, 4)
			binary.BigEndian.PutUint32(out, 0)
			_, _ = conn.Write(out)
			_ = conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	pool := bytespool.NewBytesPool(1, 1024)
	ch := make(chan []byte, 1)
	wrapped := WrapConnection(conn, pool, nil, ch, nil)

	<-time.After(100 * time.Millisecond)
	assert.Equal(t, []byte{0, 0, 0, 0}, (<-ch)[:4])
	require.NoError(t, wrapped.Close())
}

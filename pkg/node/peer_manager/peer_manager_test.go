package peer_manager

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSuspended_Block(t *testing.T) {
	addr := proto.NewTCPAddr(net.IPv4(8, 8, 8, 8), 80).ToIpPort()

	t.Run("check with same port", func(t *testing.T) {
		b := suspended{}
		require.False(t, b.Blocked(addr, time.Now()))

		b.Block(addr, 5*time.Minute)

		require.True(t, b.Blocked(addr, time.Now()))
		require.False(t, b.Blocked(addr, time.Now().Add(10*time.Minute)))

		require.Equal(t, 1, b.Len())

		b.clear(time.Now().Add(10 * time.Minute))
		require.Equal(t, 0, b.Len())
	})

	t.Run("check with different ports", func(t *testing.T) {
		b := suspended{}
		addr2 := proto.NewTCPAddr(net.IPv4(8, 8, 8, 8), 180).ToIpPort()

		b.Block(addr, 5*time.Minute)

		require.True(t, b.Blocked(addr2, time.Now()), "should be suspended, ignore port")
	})
}

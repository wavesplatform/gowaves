package state

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
)

func TestKnownPeer(t *testing.T) {
	p := proto.NewTCPAddr(net.IPv4(127, 0, 0, 1), 65535)
	k := intoBytes(p)
	p2, _ := fromBytes(k)
	require.Equal(t, p, p2)
}

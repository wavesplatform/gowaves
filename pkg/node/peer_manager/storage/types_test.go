package storage

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
	"time"
)

func TestFromUnixMillis(t *testing.T) {
	ts := time.Now().Truncate(time.Millisecond)
	tsMillis := ts.UnixNano() / 1_000_000

	require.Equal(t, ts.String(), fromUnixMillis(tsMillis).String())
}

func TestIpFromIpPort(t *testing.T) {
	ip := IpFromIpPort(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.3.4.1:2345")))
	require.Equal(t, "13.3.4.1", ip.String())
}

func TestIPFromString(t *testing.T) {
	ip := IPFromString("13.3.4.1")
	require.Equal(t, "13.3.4.1", ip.String())
}

func TestKnownPeerIP(t *testing.T) {
	k := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.3.4.1:2345")))
	ip := k.IP()
	require.Equal(t, "13.3.4.1", ip.String())
}

func TestKnownPeer_IpPort(t *testing.T) {
	k := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("13.3.4.1:2345")))
	ipPort := k.IpPort()
	require.Equal(t, ipPort.String(), k.String())
}

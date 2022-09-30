package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

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

func TestKnownOldestFirst(t *testing.T) {
	p1 := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("1.2.3.4:1")))
	p2 := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("1.2.3.4:2")))
	p3 := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("1.2.3.4:3")))
	p4 := KnownPeer(proto.NewIpPortFromTcpAddr(proto.NewTCPAddrFromString("1.2.3.4:4")))
	ps := knownPeers{}
	ps[p1] = 3
	ps[p2] = 2
	ps[p3] = 1
	ps[p4] = 0

	r := ps.OldestFirst(10)
	expected := []KnownPeer{p4, p3, p2, p1}
	assert.Equal(t, expected, r)
}

package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
)

func TestKnownPeer(t *testing.T) {
	p := KnownPeer(proto.NewNodeAddr(net.IPv4(127, 0, 0, 1), 6868))
	//{
	//	IP:   net.IPv4(127, 0, 0, 1),
	//	Port: 6868,
	//}
	p2 := KnownPeer{}
	p2.FromKey(p.key())

	assert.Equal(t, p, p2)
}

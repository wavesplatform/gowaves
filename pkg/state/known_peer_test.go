package state

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestKnownPeer(t *testing.T) {
	p := KnownPeer{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 6868,
	}
	p2 := KnownPeer{}
	p2.FromKey(p.key())

	assert.Equal(t, p, p2)
}

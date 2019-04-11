package state

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestKnownPeer(t *testing.T) {
	p := NewKnownPeer(net.IPv4(127, 0, 0, 1), 65535)
	assert.Equal(t, net.IPv4(127, 0, 0, 1), p.Addr())
	assert.Equal(t, 65535, p.Port())
	k := p.key()
	p2 := NewKnownPeerFromKey(k)
	assert.Equal(t, p, p2)
}

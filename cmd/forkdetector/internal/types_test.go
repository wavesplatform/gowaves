package internal

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestPeerDesignationString(t *testing.T) {
	pd := PeerDesignation{Address: net.IPv4(1, 2, 3, 4), Nonce: 1234567890}
	assert.Equal(t, "1.2.3.4-1234567890", pd.String())
	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
	assert.Equal(t, "255.255.255.255-0", pd.String())
}

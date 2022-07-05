package peer

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {
	addr, _ := net.ResolveTCPAddr("", "127.0.0.1:6868")
	assert.Equal(t, "127.0.0.1-100500", peerImplID{addr: addr, nonce: 100500}.String())
}

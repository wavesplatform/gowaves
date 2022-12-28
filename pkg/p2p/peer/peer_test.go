package peer

import (
	"fmt"
	"net/netip"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type netAddr struct{ net, addr string }

func (n netAddr) Network() string { return n.net }

func (n netAddr) String() string { return n.addr }

func TestPeerImplID(t *testing.T) {
	tests := []struct {
		net, addr string
		nonce     uint64
		errorStr  string
	}{
		{net: "tcp", addr: "127.0.0.1:100", nonce: 100501},
		{net: "tcp4", addr: "127.0.0.1:100", nonce: 100502},
		{net: "", addr: "127.0.0.1:100", nonce: 100504},
		{net: "tcp", addr: "[2001:db8::1]:8080", nonce: 80},
		{net: "tcp6", addr: "[2001:db8::1]:8080", nonce: 82},
		{
			net: "tcp6", addr: "127.0.0.1:100", nonce: 100503,
			errorStr: "failed to resolve 'tcp6' addr from '127.0.0.1:100': address 127.0.0.1: no suitable address found",
		},
		{
			net: "tcp4", addr: "[2001:db8::1]:8080", nonce: 81,
			errorStr: "failed to resolve 'tcp4' addr from '[2001:db8::1]:8080': address 2001:db8::1: no suitable address found",
		},
		{
			net: "udp", addr: "[2001:db8::1]:8080", nonce: 80,
			errorStr: "failed to resolve 'udp' addr from '[2001:db8::1]:8080': unknown network udp",
		},
		{
			net: "tcp", addr: "127.0.0.01", nonce: 90,
			errorStr: "failed to resolve 'tcp' addr from '127.0.0.01': address 127.0.0.01: missing port in address",
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			id, err := newPeerImplID(netAddr{net: test.net, addr: test.addr}, test.nonce)
			if test.errorStr != "" {
				assert.EqualError(t, err, test.errorStr)
			} else {
				addrP, err := netip.ParseAddrPort(test.addr)
				require.NoError(t, err)
				expectedAddr := addrP.Addr()
				assert.Equal(t, expectedAddr.As16(), id.addr16)
				assert.Equal(t, test.nonce, id.nonce)
				expectedString := fmt.Sprintf("%s-%d", expectedAddr, test.nonce)
				assert.Equal(t, expectedString, id.String())
			}
		})
	}
}

func TestPeerImplId_InMap(t *testing.T) {
	const (
		net  = "tcp"
		addr = "127.0.0.1:8080"
	)
	type noncePair struct{ first, second uint64 }
	for i, np := range []noncePair{{100, 500}, {100, 100}} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			first, err := newPeerImplID(netAddr{net: net, addr: addr}, np.first)
			require.NoError(t, err)
			second, err := newPeerImplID(netAddr{net: net, addr: addr}, np.second)
			require.NoError(t, err)

			m := map[ID]struct{}{first: {}}
			_, ok := m[second]
			if unique := np.first != np.second; unique {
				assert.False(t, ok)
			} else {
				assert.True(t, ok)
			}
		})
	}
}

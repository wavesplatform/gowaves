package internal

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestPeerDesignationString(t *testing.T) {
	pd := PeerDesignation{Address: net.IPv4(1, 2, 3, 4), Nonce: 1234567890}
	assert.Equal(t, "1.2.3.4-1234567890", pd.String())
	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
	assert.Equal(t, "255.255.255.255-0", pd.String())
}

func TestPeerDesignationMarshalJSON(t *testing.T) {
	pd := PeerDesignation{Address:net.IPv4(1, 2, 3, 4).To4(), Nonce: 567890}
	js, err := json.Marshal(pd)
	require.NoError(t, err)
	assert.Equal(t, "\"1.2.3.4-567890\"", string(js))
	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
	js, err = json.Marshal(pd)
	require.NoError(t, err)
	assert.Equal(t, "\"255.255.255.255-0\"", string(js))
}

func TestParsePeerAddr(t *testing.T) {
	pa1, err := ParsePeerAddr("127.0.0.1:1234")
	require.NoError(t, err)
	assert.Equal(t, net.IPv4(127, 0, 0, 1), pa1.IP)
	assert.Equal(t, 1234, pa1.Port)

	pa2, err := ParsePeerAddr("8.8.8.8:12345")
	require.NoError(t, err)
	assert.Equal(t, net.IPv4(8, 8, 8, 8), pa2.IP)
	assert.Equal(t, 12345, pa2.Port)

	pa3, err := ParsePeerAddr("[::1]:1234")
	require.NoError(t, err)
	assert.Equal(t, net.IPv6loopback, pa3.IP)
	assert.Equal(t, 1234, pa3.Port)

	pa4, err := ParsePeerAddr("[FE80:0000:0000:0000:0202:B3FF:FE1E:8329]:65535")
	require.NoError(t, err)
	assert.Equal(t, net.ParseIP("FE80:0000:0000:0000:0202:B3FF:FE1E:8329"), pa4.IP)
	assert.Equal(t, 65535, pa4.Port)


	_, err = ParsePeerAddr("localhost:1234")
	assert.Errorf(t, err, "failed to create PeerAddr from string")

	_, err = ParsePeerAddr("8.8.8.8")
	assert.Errorf(t, err, "failed to create PeerAddr from string")

	_, err = ParsePeerAddr("8.8.8.8:")
	assert.Errorf(t, err, "failed to create PeerAddr from string")

	_, err = ParsePeerAddr("8.8.8.8:port")
	assert.Errorf(t, err, "failed to create PeerAddr from string")

	_, err = ParsePeerAddr("host:port")
	assert.Errorf(t, err, "failed to create PeerAddr from string")

	_, err = ParsePeerAddr("host:12345")
	assert.Errorf(t, err, "failed to create PeerAddr from string")
}

func TestPeerAddrHash(t *testing.T) {
	pa1, err := ParsePeerAddr("127.0.0.1:1234")
	require.NoError(t, err)
	pa2, err := ParsePeerAddr("127.0.0.1:1235")
	require.NoError(t, err)
	pa3, err := ParsePeerAddr("127.0.0.1:1234")
	require.NoError(t, err)
	pa4, err := ParsePeerAddr("[::1]:1235")
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, pa1.Hash(), pa3.Hash())
	assert.NotEqual(t, pa1.Hash(), pa2.Hash())
	assert.NotEqual(t, pa2.Hash(), pa4.Hash())
}
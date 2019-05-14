package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
	"time"
)

func TestPeerNodeBinaryRoundTrip(t *testing.T) {
	ts := time.Now()
	tests := []struct {
		address  net.IP
		port     uint16
		nonce    uint64
		name     string
		version  proto.Version
		attempts int
		next     time.Time
		state    NodeState
	}{
		{net.IPv4(1, 2, 3, 4), 1234, 1234567890, "wavesT", proto.Version{Major: 0, Minor: 16}, 0, ts, NodeUnknown},
		{net.IPv6zero, 5678, 9876543210, "wavesW", proto.Version{Major: 0, Minor: 15}, 12345, ts, NodeHostile},
		{net.IPv6loopback, 6666, 0, "", proto.Version{}, 1, ts, NodeResponding},
		{net.IPv4zero, 0, 0, "", proto.Version{}, 0, ts, NodeDiscarded},
	}
	for _, tc := range tests {
		pn := PeerNode{
			Address:     tc.address,
			Port:        tc.port,
			Nonce:       tc.nonce,
			Name:        tc.name,
			Version:     tc.version,
			Attempts:    tc.attempts,
			NextAttempt: tc.next,
			State:       tc.state,
		}
		if b, err := pn.MarshalBinary(); assert.NoError(t, err) {
			var a PeerNode
			if err := a.UnmarshalBinary(b); assert.NoError(t, err) {
				assert.Equal(t, pn.Address, a.Address)
				assert.Equal(t, pn.Port, a.Port)
				assert.Equal(t, pn.Nonce, a.Nonce)
				assert.Equal(t, pn.Name, a.Name)
				assert.Equal(t, pn.Version, a.Version)
				assert.Equal(t, pn.Attempts, a.Attempts)
				assert.True(t, pn.NextAttempt.Equal(a.NextAttempt))
				assert.Equal(t, pn.State, a.State)
			}
		}
	}
}

func TestPeerNodeString(t *testing.T) {
	ts := time.Time{}
	tests := []struct {
		address  net.IP
		port     uint16
		nonce    uint64
		name     string
		version  proto.Version
		attempts int
		next     time.Time
		state    NodeState
		exp      string
	}{
		{net.IPv4(1, 2, 3, 4), 1234, 1234567890, "wavesT", proto.Version{Major: 0, Minor: 16}, 0, ts, NodeUnknown, "1.2.3.4:1234|1234567890|wavesT|UNKNOWN|v0.16.0|0|0001-01-01T00:00:00Z"},
		{net.IPv6zero, 5678, 9876543210, "wavesW", proto.Version{Major: 0, Minor: 15}, 12345, ts, NodeHostile, "[::]:5678|9876543210|wavesW|HOSTILE|v0.15.0|12345|0001-01-01T00:00:00Z"},
		{net.IPv6loopback, 6666, 0, "", proto.Version{}, 1, ts, NodeResponding, "[::1]:6666|0||RESPONDING|v0.0.0|1|0001-01-01T00:00:00Z"},
		{net.IPv4zero, 0, 0, "", proto.Version{}, 0, ts, NodeDiscarded, "0.0.0.0:0|0||DISCARDED|v0.0.0|0|0001-01-01T00:00:00Z"},
	}
	for _, tc := range tests {
		pn := PeerNode{
			Address:     tc.address,
			Port:        tc.port,
			Nonce:       tc.nonce,
			Name:        tc.name,
			Version:     tc.version,
			Attempts:    tc.attempts,
			NextAttempt: tc.next,
			State:       tc.state,
		}
		assert.Equal(t, tc.exp, pn.String())
	}
}

//func TestPeerDesignationString(t *testing.T) {
//	pd := PeerDesignation{Address: net.IPv4(1, 2, 3, 4), Nonce: 1234567890}
//	assert.Equal(t, "1.2.3.4-1234567890", pd.String())
//	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
//	assert.Equal(t, "255.255.255.255-0", pd.String())
//}
//
//func TestPeerDesignationMarshalJSON(t *testing.T) {
//	pd := PeerDesignation{Address:net.IPv4(1, 2, 3, 4).To4(), Nonce: 567890}
//	js, err := json.Marshal(pd)
//	require.NoError(t, err)
//	assert.Equal(t, "\"1.2.3.4-567890\"", string(js))
//	pd = PeerDesignation{Address:net.IPv4bcast, Nonce:0}
//	js, err = json.Marshal(pd)
//	require.NoError(t, err)
//	assert.Equal(t, "\"255.255.255.255-0\"", string(js))
//}
//
//func TestParsePeerAddr(t *testing.T) {
//	pa1, err := ParsePeerAddr("127.0.0.1:1234")
//	require.NoError(t, err)
//	assert.Equal(t, net.IPv4(127, 0, 0, 1), pa1.IP)
//	assert.Equal(t, 1234, pa1.Port)
//
//	pa2, err := ParsePeerAddr("8.8.8.8:12345")
//	require.NoError(t, err)
//	assert.Equal(t, net.IPv4(8, 8, 8, 8), pa2.IP)
//	assert.Equal(t, 12345, pa2.Port)
//
//	pa3, err := ParsePeerAddr("[::1]:1234")
//	require.NoError(t, err)
//	assert.Equal(t, net.IPv6loopback, pa3.IP)
//	assert.Equal(t, 1234, pa3.Port)
//
//	pa4, err := ParsePeerAddr("[FE80:0000:0000:0000:0202:B3FF:FE1E:8329]:65535")
//	require.NoError(t, err)
//	assert.Equal(t, net.ParseIP("FE80:0000:0000:0000:0202:B3FF:FE1E:8329"), pa4.IP)
//	assert.Equal(t, 65535, pa4.Port)
//
//
//	_, err = ParsePeerAddr("localhost:1234")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//
//	_, err = ParsePeerAddr("8.8.8.8")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//
//	_, err = ParsePeerAddr("8.8.8.8:")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//
//	_, err = ParsePeerAddr("8.8.8.8:port")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//
//	_, err = ParsePeerAddr("host:port")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//
//	_, err = ParsePeerAddr("host:12345")
//	assert.Errorf(t, err, "failed to create PeerAddr from string")
//}
//
//func TestPeerAddrHash(t *testing.T) {
//	pa1, err := ParsePeerAddr("127.0.0.1:1234")
//	require.NoError(t, err)
//	pa2, err := ParsePeerAddr("127.0.0.1:1235")
//	require.NoError(t, err)
//	pa3, err := ParsePeerAddr("127.0.0.1:1234")
//	require.NoError(t, err)
//	pa4, err := ParsePeerAddr("[::1]:1235")
//	require.NoError(t, err)
//
//	require.NoError(t, err)
//	assert.Equal(t, pa1.Hash(), pa3.Hash())
//	assert.NotEqual(t, pa1.Hash(), pa2.Hash())
//	assert.NotEqual(t, pa2.Hash(), pa4.Hash())
//}

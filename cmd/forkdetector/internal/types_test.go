package internal

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
		{net.IPv4(1, 2, 3, 4), 1234, 1234567890, "wavesT", proto.NewVersion(0, 16, 0), 0, ts, NodeUnknown},
		{net.IPv6zero, 5678, 9876543210, "wavesW", proto.NewVersion(0, 15, 0), 12345, ts, NodeHostile},
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
		{net.IPv4(1, 2, 3, 4), 1234, 1234567890, "wavesT", proto.NewVersion(0, 16, 0), 0, ts, NodeUnknown, "1.2.3.4:1234|1234567890|wavesT|UNKNOWN|v0.16.0|0|0001-01-01T00:00:00Z"},
		{net.IPv6zero, 5678, 9876543210, "wavesW", proto.NewVersion(0, 15, 0), 12345, ts, NodeHostile, "[::]:5678|9876543210|wavesW|HOSTILE|v0.15.0|12345|0001-01-01T00:00:00Z"},
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

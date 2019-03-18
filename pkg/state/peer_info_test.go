package state

//
//import (
//	"bytes"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"net"
//	"testing"
//	"time"
//)
//
//func TestPeerInfo_Marshalling(t *testing.T) {
//	p := PeerInfo{
//		IP:            net.IPv4(127, 0, 0, 1),
//		Nonce:         255,
//		Port:          6868,
//		Name:          "first",
//		LastConnected: time.Date(2019, 3, 8, 23, 59, 14, 0, time.UTC),
//	}
//
//	buf := new(bytes.Buffer)
//	n, err := p.WriteTo(buf)
//
//	require.NoError(t, err)
//	assert.Equal(t, 28, n)
//
//	var p2 PeerInfo
//	require.NoError(t, p2.ReadFrom(buf))
//	assert.True(t, p.IP.Equal(p2.IP))
//	assert.Equal(t, p.Nonce, p2.Nonce)
//	assert.Equal(t, p.Port, p2.Port)
//	assert.Equal(t, p.Name, p2.Name)
//	assert.True(t, p.LastConnected.Equal(p2.LastConnected))
//}
//
//func TestPeerInfo_key(t *testing.T) {
//	p := PeerInfo{
//		IP:    net.IPv4(127, 0, 0, 1),
//		Nonce: 255,
//	}
//	assert.Equal(t, [13]byte{knownPeersPrefix, 127, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 255}, [13]byte(p.key()))
//}

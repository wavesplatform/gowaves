package client

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewPeers(t *testing.T) {
	assert.NotNil(t, NewPeers(defaultOptions))
}

var peersAllJson = `
{
  "peers": [
    {
      "address": "/127.0.0.1:6868",
      "lastSeen": 1540383498486
    }
  ]
}
`

func TestPeers_All(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(peersAllJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.All(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "127.0.0.1", body[0].Address.Addr.String())
	assert.Equal(t, uint16(6868), body[0].Address.Port)
	assert.Equal(t, uint64(1540383498486), body[0].LastSeen)
}

var peersConnectedJson = `
{
  "peers": [
    {
      "address": "/127.0.0.1:6863",
      "declaredAddress": "/127.0.0.1:6863",
      "peerName": "official-7",
      "peerNonce": 204829,
      "applicationName": "wavesT",
      "applicationVersion": "0.15.0"
    },
    {
      "address": "/8.8.8.8:60938",
      "declaredAddress": "N/A",
      "peerName": "My TESTNET node",
      "peerNonce": 1828593,
      "applicationName": "wavesT",
      "applicationVersion": "0.15.0"
    }
  ]
}
`

func TestPeers_Connected(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(peersConnectedJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.Connected(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 2, len(body))
	assert.Equal(t, &PeersConnectedRow{
		Address: proto.PeerInfo{
			Addr: net.ParseIP("127.0.0.1"),
			Port: 6863,
		},
		DeclaredAddress: proto.PeerInfo{
			Addr: net.ParseIP("127.0.0.1"),
			Port: 6863,
		},
		PeerName:           "official-7",
		PeerNonce:          204829,
		ApplicationName:    "wavesT",
		ApplicationVersion: "0.15.0",
	}, body[0])

	assert.Nil(t, body[1].DeclaredAddress.Addr)
	assert.Equal(t, uint16(0), body[1].DeclaredAddress.Port)
}

var peersBlacklistedJson = `
[
  {
    "hostname": "127.0.0.1",
    "timestamp": 1540460047473,
    "reason": "Timeout expired while waiting for handshake"
  }
]
`

func TestPeers_Blacklisted(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(peersBlacklistedJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.Blacklisted(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "127.0.0.1", body[0].Hostname.Addr.String())
	assert.Equal(t, uint64(1540460047473), body[0].Timestamp)
	assert.Equal(t, "Timeout expired while waiting for handshake", body[0].Reason)
}

var peersSuspendedJson = `
[
  {
    "hostname": "/127.0.0.1",
    "timestamp": 1540461977116
  }
]
`

func TestPeers_Suspended(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(peersSuspendedJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.Suspended(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "127.0.0.1", body[0].Hostname.Addr.String())
	assert.Equal(t, uint64(1540461977116), body[0].Timestamp)
}

var peersConnectJson = `
{
  "hostname": "localhost",
  "status": "Trying to connect"
}
`

func TestPeers_Connect(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(peersConnectJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.Connect(context.Background(), "localhost", 6868)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "localhost", body.Hostname)
	assert.Equal(t, "Trying to connect", body.Status)
}

func TestPeers_ClearBlacklist(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(`{"result": "blacklist cleared"}`, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Peers.ClearBlacklist(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "blacklist cleared", body)
}

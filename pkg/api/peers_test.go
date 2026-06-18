package api

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestApp_PeersKnown(t *testing.T) {
	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{
			GenerationPeriod: 0,
		},
	}
	peerManager := peers.NewMockPeerManager(t)
	addr := proto.NewTCPAddr(net.ParseIP("127.0.0.1"), 6868).ToIpPort()
	peerManager.EXPECT().KnownPeers().Return([]storage.KnownPeer{storage.KnownPeer(addr)})

	app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
	require.NoError(t, err)

	rs2, err := app.PeersKnown()
	require.NoError(t, err)
	require.Len(t, rs2.Peers, 1)
}

func TestApp_PeersBlackListed(t *testing.T) {
	peerManager := peers.NewMockPeerManager(t)

	now := time.Now()

	ips := []string{"13.3.4.1", "5.3.6.7"}
	testData := []storage.BlackListedPeer{
		{
			IP:                      storage.IPFromString(ips[0]),
			RestrictTimestampMillis: now.Add(time.Minute).UnixMilli(),
			RestrictDuration:        time.Minute,
			Reason:                  "some reason #1",
		},
		{
			IP:                      storage.IPFromString(ips[1]),
			RestrictTimestampMillis: now.Add(2 * time.Minute).UnixMilli(),
			RestrictDuration:        time.Minute,
			Reason:                  "some reason #2",
		},
	}

	peerManager.EXPECT().BlackList().Return(testData)
	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{
			GenerationPeriod: 0,
		},
	}
	app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
	require.NoError(t, err)

	blackList := app.PeersBlackListed()

	for i, actual := range blackList {
		p := testData[i]
		expected := RestrictedPeerInfo{
			Hostname:  "/" + ips[i],
			Timestamp: p.RestrictTimestampMillis,
			Reason:    p.Reason,
		}
		assert.Equal(t, expected, actual)
	}
}

func TestApp_PeersAll(t *testing.T) {
	peerManager := peers.NewMockPeerManager(t)

	const blackListedIPStr = "13.3.4.1"
	blackListedIP := storage.IPFromString(blackListedIPStr)
	peerManager.EXPECT().BlackList().Return([]storage.BlackListedPeer{{
		IP:                      blackListedIP,
		RestrictTimestampMillis: time.Now().UnixMilli(),
		RestrictDuration:        time.Minute,
		Reason:                  "some reason",
	}})

	allowedAddr := proto.NewTCPAddr(net.ParseIP("5.3.6.7"), 6868).ToIpPort()
	blackListedAddr := proto.NewTCPAddr(net.ParseIP(blackListedIPStr), 6868).ToIpPort()
	peerManager.EXPECT().KnownPeers().Return([]storage.KnownPeer{
		storage.KnownPeer(allowedAddr),
		storage.KnownPeer(blackListedAddr),
	})

	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{
			GenerationPeriod: 0,
		},
	}

	app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
	require.NoError(t, err)

	out, err := app.PeersAll()
	require.NoError(t, err)
	require.Len(t, out.Peers, 1)
	addresses := []string{out.Peers[0].Address}
	assert.ElementsMatch(t, []string{"/5.3.6.7:6868"}, addresses)
	assert.NotZero(t, out.Peers[0].LastSeen)
}

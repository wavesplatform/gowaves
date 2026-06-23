package api

import (
	stderrs "errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/api/errors"
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

func TestApp_PeersBlackList(t *testing.T) {
	const (
		requestID = "request-123"
		clientIP  = "192.168.1.1"
	)
	t.Run("ip:port", func(t *testing.T) {
		const (
			blacklistedIP   = "5.3.6.7"
			blacklistedPort = "6868"
			blacklistedAddr = blacklistedIP + ":" + blacklistedPort
		)
		peerManager := peers.NewMockPeerManager(t)
		peerManager.EXPECT().AddToBlackListByIP(
			mock.MatchedBy(func(addr proto.TCPAddr) bool {
				return addr.String() == blacklistedIP+":0" // because port is ignored in the blacklist
			}),
			mock.MatchedBy(func(t time.Time) bool {
				return !t.IsZero()
			}),
			mock.MatchedBy(func(reason string) bool {
				return reason != ""
			}),
		).Return()

		cfg := &settings.BlockchainSettings{
			FunctionalitySettings: settings.FunctionalitySettings{
				GenerationPeriod: 0,
			},
		}

		app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
		require.NoError(t, err)

		err = app.PeersBlackList(blacklistedAddr, requestID, clientIP)
		require.NoError(t, err)
	})
	t.Run("ip:bad_port", func(t *testing.T) {
		doTest := func(t *testing.T, port string) {
			const (
				blacklistedIP = "5.3.6.7"
			)
			peerManager := peers.NewMockPeerManager(t)
			cfg := &settings.BlockchainSettings{
				FunctionalitySettings: settings.FunctionalitySettings{
					GenerationPeriod: 0,
				},
			}

			app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
			require.NoError(t, err)

			blacklistedAddr := blacklistedIP + ":" + port
			err = app.PeersBlackList(blacklistedAddr, requestID, clientIP)
			require.Error(t, err)
			uErr := stderrs.Unwrap(err)
			require.Error(t, uErr)
			require.EqualError(t, uErr, fmt.Sprintf(
				"failed to resolve blacklisted host '5.3.6.7:%s': invalid port '%s'", port, port,
			))
			require.Equal(t, errors.NewBadRequestError(uErr), err)
		}
		for i, v := range []string{"bad", "0"} {
			t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
				doTest(t, v)
			})
		}
	})
	t.Run("ip", func(t *testing.T) {
		const blacklistedAddr = "5.3.6.7"
		peerManager := peers.NewMockPeerManager(t)
		peerManager.EXPECT().AddToBlackListByIP(
			mock.MatchedBy(func(addr proto.TCPAddr) bool {
				return addr.String() == blacklistedAddr+":0"
			}),
			mock.MatchedBy(func(t time.Time) bool {
				return !t.IsZero()
			}),
			mock.MatchedBy(func(reason string) bool {
				return reason != ""
			}),
		).Return()

		cfg := &settings.BlockchainSettings{
			FunctionalitySettings: settings.FunctionalitySettings{
				GenerationPeriod: 0,
			},
		}

		app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
		require.NoError(t, err)

		err = app.PeersBlackList(blacklistedAddr, requestID, clientIP)
		require.NoError(t, err)
	})
	t.Run("unspecified_ip", func(t *testing.T) {
		doTest := func(t *testing.T, addr string) {
			peerManager := peers.NewMockPeerManager(t)
			cfg := &settings.BlockchainSettings{
				FunctionalitySettings: settings.FunctionalitySettings{
					GenerationPeriod: 0,
				},
			}

			app, err := NewApp("key", nil, services.Services{Peers: peerManager}, cfg)
			require.NoError(t, err)

			err = app.PeersBlackList(addr, requestID, clientIP)
			require.Error(t, err)
			uErr := stderrs.Unwrap(err)
			require.Error(t, uErr)
			require.EqualError(t, uErr, fmt.Sprintf("no valid IPs found for blacklisted host '%s'", addr))
			require.Equal(t, errors.NewBadRequestError(uErr), err)
		}
		for i, v := range []string{"", ":21", "0.0.0.0", "0.0.0.0:42"} {
			t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
				doTest(t, v)
			})
		}
	})
}

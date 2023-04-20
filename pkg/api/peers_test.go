package api

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestApp_PeersKnown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	peerManager := mock.NewMockPeerManager(ctrl)
	addr := proto.NewTCPAddr(net.ParseIP("127.0.0.1"), 6868).ToIpPort()
	peerManager.EXPECT().KnownPeers().Return([]storage.KnownPeer{storage.KnownPeer(addr)})

	app, err := NewApp("key", nil, services.Services{Peers: peerManager})
	require.NoError(t, err)

	rs2, err := app.PeersKnown()
	require.NoError(t, err)
	require.Len(t, rs2.Peers, 1)
}

func TestApp_PeersSuspended(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	peerManager := mock.NewMockPeerManager(ctrl)

	now := time.Now()

	ips := []string{"13.3.4.1", "5.3.6.7"}
	testData := []storage.SuspendedPeer{
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

	peerManager.EXPECT().Suspended().Return(testData)

	app, err := NewApp("key", nil, services.Services{Peers: peerManager})
	require.NoError(t, err)

	suspended := app.PeersSuspended()

	for i, actual := range suspended {
		p := testData[i]
		expected := RestrictedPeerInfo{
			Hostname:  "/" + ips[i],
			Timestamp: p.RestrictTimestampMillis,
			Reason:    p.Reason,
		}
		assert.Equal(t, expected, actual)
	}
}

func TestApp_PeersBlackList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	peerManager := mock.NewMockPeerManager(ctrl)

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

	app, err := NewApp("key", nil, services.Services{Peers: peerManager})
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

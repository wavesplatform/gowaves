package api

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestApp_PeersKnown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	peerManager := mock.NewMockPeerManager(ctrl)
	peerManager.EXPECT().KnownPeers().Return([]proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")}, nil)

	//s := mock.NewMockState(ctrl)
	//s.EXPECT().Peers().Return([]proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")}, nil)

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
	testData := []peer_manager.SuspendedInfo{
		{
			IP:              peer_manager.IPFromString(ips[0]),
			SuspendTime:     now.Add(time.Minute),
			SuspendDuration: time.Minute,
			Reason:          "some reason #1",
		},
		{
			IP:              peer_manager.IPFromString(ips[1]),
			SuspendTime:     now.Add(2 * time.Minute),
			SuspendDuration: time.Minute,
			Reason:          "some reason #2",
		},
	}

	peerManager.EXPECT().Suspended().Return(testData)

	app, err := NewApp("key", nil, services.Services{Peers: peerManager})
	require.NoError(t, err)

	suspended := app.PeersSuspended()

	for i, actual := range suspended {
		p := testData[i]
		expected := SuspendedPeerInfo{
			Hostname:  "/" + ips[i],
			Timestamp: unixMillis(p.SuspendTime),
			Reason:    p.Reason,
		}
		assert.Equal(t, expected, actual)
	}
}

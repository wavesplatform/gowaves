package api

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"

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

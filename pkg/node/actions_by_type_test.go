package node

import (
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestPeersAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock.NewMockPeerManager(ctrl)
	m.EXPECT().KnownPeers().Return([]proto.TCPAddr{}, nil)
	m.EXPECT().UpdateKnownPeers([]proto.TCPAddr{{IP: net.ParseIP("127.0.0.1"), Port: 6868}})

	_, _, err := PeersAction(services.Services{
		Peers: m,
	}, peer.ProtoMessage{
		Message: &proto.PeersMessage{
			Peers: []proto.PeerInfo{
				{
					Addr: net.ParseIP("127.0.0.1"),
					Port: 6868,
				},
			},
		},
	}, nil)
	require.NoError(t, err)
}

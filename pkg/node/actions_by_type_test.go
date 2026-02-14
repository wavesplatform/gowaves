package node

import (
	"log/slog"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestPeersAction(t *testing.T) {
	m := peers.NewMockPeerManager(t)
	m.EXPECT().KnownPeers().Return([]storage.KnownPeer{})
	addr := proto.NewTCPAddr(net.ParseIP("127.0.0.1"), 6868).ToIpPort()
	m.EXPECT().UpdateKnownPeers([]storage.KnownPeer{storage.KnownPeer(addr)}).Return(nil)

	_, err := PeersAction(services.Services{
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
	}, nil, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
}

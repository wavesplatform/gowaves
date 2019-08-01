package state_changed

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type eachConnectedImpl struct {
	peers []Peer
}

func (a eachConnectedImpl) EachConnected(f func(Peer, *proto.Score)) {
	for _, p := range a.peers {
		f(p, nil)
	}
}

// test that score sender sends right
func TestScoreSender_Handle(t *testing.T) {
	g := &proto.Block{
		BlockHeader: proto.BlockHeader{
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100500,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"),
		},
	}
	peer := mock.NewPeer()
	peers := eachConnectedImpl{
		peers: []Peer{peer},
	}
	s := node.NewMockStateManager(g)
	sender := NewScoreSender(peers, s)

	require.Equal(t, 0, len(peer.SendMessageCalledWith))
	sender.Handle()
	require.Equal(t, 1, len(peer.SendMessageCalledWith))
}

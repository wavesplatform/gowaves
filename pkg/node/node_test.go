package node

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestNode_HandleProtoMessage_GetBlockBySignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	peers := mock.NewMockPeerManager(ctrl)
	peer := mock.NewMockPeer(ctrl)
	s := newMockStateWithGenesis()

	peer.EXPECT().SendMessage(gomock.Any())

	n := NewNode(services.Services{State: s, Peers: peers}, proto.TCPAddr{}, proto.TCPAddr{}, nil, nil)
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	n.handleBlockBySignatureMessage(peer, sig)
}

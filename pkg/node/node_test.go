package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestNode_HandleProtoMessage_GetBlockBySignature(t *testing.T) {
	s := newMockStateWithGenesis()
	peers, peer := NewMockPeerManagerWithDefaultPeer()
	n := NewNode(services.Services{State: s, Peers: peers}, proto.TCPAddr{}, nil, nil, nil)
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	n.handleBlockBySignatureMessage(peer, sig)
	assert.Equal(t, 1, len(peer.SendMessageCalledWith))
}

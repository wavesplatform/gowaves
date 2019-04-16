package node

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestNode_HandleProtoMessage_GetBlockBySignature(t *testing.T) {
	s := newMockStateWithGenesis()
	peers, pName, peer := NewMockPeerManagerWithDefaultPeer()
	n := NewNode(s, peers, proto.TCPAddr{})
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	n.handleBlockBySignatureMessage(pName, sig)
	assert.Equal(t, 1, len(peer.SendMessageCalledWith))
}

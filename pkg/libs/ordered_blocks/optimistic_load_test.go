package ordered_blocks_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var sig1 = crypto.MustSignatureFromBase58("31jt6L3pDU2mkow3kDK7kUZjQbqJsMnE5gC6As7Cz27xjqAaZpiNqopf6NJWbtwrV9VcjShKFfhgLmjpr8Ybuv41")
var sig2 = crypto.MustSignatureFromBase58("53wmPSc2n5DwpDcJUfNCV7j2wCuc227M9onwrs72orKuyKy5iPkcvKzE4a1Bikr2ixTG8N6GRrM8grn8sQ7qaC8w")

func makeBlock(sig crypto.Signature) *proto.Block {
	return &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: sig,
		},
	}
}

func TestOrderedBlocks(t *testing.T) {
	o := state_fsm.NewOrderedBlocks()
	o.Add(sig1)
	require.Len(t, o.PopAll(), 0)

	o.Add(sig2)
	require.Len(t, o.PopAll(), 0)

	// second block arrived first, no sequence right now
	o.SetBlock(makeBlock(sig2))
	require.Len(t, o.PopAll(), 0)

	// finally arrived first block, so seq contains 2 blocks
	o.SetBlock(makeBlock(sig1))
	require.Len(t, o.PopAll(), 2)
}

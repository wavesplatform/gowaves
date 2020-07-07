package ordered_blocks_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/ordered_blocks"
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
	o := ordered_blocks.NewOrderedBlocks()
	o.Add(proto.NewBlockIDFromSignature(sig1))
	require.Len(t, o.PopAll(), 0)

	o.Add(proto.NewBlockIDFromSignature(sig2))
	require.Len(t, o.PopAll(), 0)

	// second block arrived first, no sequence right now
	o.SetBlock(makeBlock(sig2))
	require.Len(t, o.PopAll(), 0)
	//require.Equal(t, 0, o.ReceivedCount())

	// finally arrived first block, so seq contains 2 blocks
	o.SetBlock(makeBlock(sig1))
	//require.Equal(t, 2, o.ReceivedCount())
	require.Len(t, o.PopAll(), 2)
}

func TestOrderedBlocks_AvailableCount(t *testing.T) {
	o := ordered_blocks.NewOrderedBlocks()
	o.Add(proto.NewBlockIDFromSignature(sig1))
	o.Add(proto.NewBlockIDFromSignature(sig2))
	require.Equal(t, 0, o.ReceivedCount())

	o.SetBlock(makeBlock(sig1))
	require.Equal(t, 1, o.ReceivedCount())

	o.SetBlock(makeBlock(sig2))
	require.Equal(t, 2, o.ReceivedCount())

	o.PopAll()
	require.Equal(t, 0, o.ReceivedCount())
}

package ng

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var sig1 = proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("5djHxGcrueh8tP2aA1UUgcvGQ1Up5e9u65wTvap2P4igbUo1oFn5cV6rSNGzo15mxhEMY7JPnA3M645jLPES93ZC"))
var sig2 = proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("Me1vXm5e7jKoHQ165vHKvZLwU5xL7Y6PVKRmbKrD7T15hLnLvpkGukVUTVHsNzrLL9H7kqsrR38dVUA98V6qUdB"))
var sig3 = proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("3yAvQJGneoczXz6aQPHKFE3z3DEuWp7MoSVqxz8y57huufanv3RJPncmaZWZN3hDc7can994ET2UxSbLNesKV29H"))
var sig4 = proto.NewBlockIDFromSignature(crypto.MustSignatureFromBase58("3k9YkeXviV4edXuBizn5pP8vRbHeyGHaRQwrVzTX6pHTCvUymRj4n3Ye1hWG1JwCypc4P34uHvag1uCTH9HRazUw"))
var emptySig = crypto.Signature{}

func newBlock(sig proto.BlockID, parent proto.BlockID) *proto.Block {
	return &proto.Block{
		BlockHeader: proto.BlockHeader{
			ID:     sig,
			Parent: parent,
		},
		Transactions: proto.Transactions(nil),
	}
}

func newMicro(sig crypto.Signature, parent proto.BlockID) *proto.MicroBlock {
	return &proto.MicroBlock{
		TotalResBlockSigField: sig,
		Reference:             parent,
	}
}

func TestBlockSequence_NoSideEffects(t *testing.T) {
	rs1 := NewBlocksFromBlock(newBlock(sig1, proto.NewBlockIDFromSignature(emptySig)))
	require.Equal(t, 1, rs1.Len())

	rs2, err := rs1.AddMicro(newMicro(sig2, sig1))
	require.NoError(t, err)

	require.Equal(t, 1, rs1.Len())
	require.Equal(t, 2, rs2.Len())
}

func TestBlockSequence_Row(t *testing.T) {
	block1 := NewBlocksFromBlock(newBlock(sig1, emptySig))
	micro1, err := block1.AddMicro(newMicro(sig2, sig1))
	require.NoError(t, err)

	row1 := micro1.Row()
	require.Equal(t, sig1, row1.KeyBlock.BlockSignature)
	require.Equal(t, 1, len(row1.MicroBlocks))

	block2, err := micro1.AddBlock(newBlock(sig3, sig2))
	require.NoError(t, err)
	micro2, err := block2.AddMicro(newMicro(sig4, sig3))
	require.NoError(t, err)

	row2 := micro2.Row()
	require.Equal(t, sig3, row2.KeyBlock.BlockSignature)
	require.Equal(t, 1, len(row2.MicroBlocks))

	/// check previous row
	prevRow, _ := micro2.PreviousRow()
	require.Equal(t, row1, prevRow)
}

// we have [Block1, Micro1, Block2, Micro2]
// after cur expect to be [Block2, Micro2]
func TestBlockSequence_CutFirst(t *testing.T) {
	block1 := NewBlocksFromBlock(newBlock(sig1, emptySig))
	micro1, _ := block1.AddMicro(newMicro(sig2, sig1))

	block2, _ := micro1.AddBlock(newBlock(sig3, sig2))
	micro2, _ := block2.AddMicro(newMicro(sig4, sig3))

	rs3 := micro2.CutFirstRow()
	require.Equal(t, 2, rs3.Len())
	require.Equal(t, sig3, rs3.First().BlockSignature)
}

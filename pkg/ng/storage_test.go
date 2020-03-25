package ng

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var sig1 = crypto.MustSignatureFromBase58("5djHxGcrueh8tP2aA1UUgcvGQ1Up5e9u65wTvap2P4igbUo1oFn5cV6rSNGzo15mxhEMY7JPnA3M645jLPES93ZC")
var id1 = proto.NewBlockIDFromSignature(sig1)
var sig2 = crypto.MustSignatureFromBase58("Me1vXm5e7jKoHQ165vHKvZLwU5xL7Y6PVKRmbKrD7T15hLnLvpkGukVUTVHsNzrLL9H7kqsrR38dVUA98V6qUdB")
var sig3 = crypto.MustSignatureFromBase58("3yAvQJGneoczXz6aQPHKFE3z3DEuWp7MoSVqxz8y57huufanv3RJPncmaZWZN3hDc7can994ET2UxSbLNesKV29H")
var sig4 = crypto.MustSignatureFromBase58("3k9YkeXviV4edXuBizn5pP8vRbHeyGHaRQwrVzTX6pHTCvUymRj4n3Ye1hWG1JwCypc4P34uHvag1uCTH9HRazUw")
var emptySig = crypto.Signature{}
var emptyId = proto.NewBlockIDFromSignature(emptySig)

func newBlock(sig crypto.Signature, parent crypto.Signature) *proto.Block {
	return &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: sig,
			Parent:         proto.NewBlockIDFromSignature(parent),
		},
		Transactions: proto.Transactions(nil),
	}
}

func newMicro(sig crypto.Signature, parent crypto.Signature) *proto.MicroBlock {
	return &proto.MicroBlock{
		TotalResBlockSigField: sig,
		TotalBlockID:          proto.NewBlockIDFromSignature(sig),
		Reference:             proto.NewBlockIDFromSignature(parent),
	}
}

func newInv(sig crypto.Signature) *proto.MicroBlockInv {
	id := proto.NewBlockIDFromSignature(sig)
	return &proto.MicroBlockInv{
		TotalBlockID: id,
	}
}

/* TODO: unused code, need to write tests if it is needed or otherwise remove it.
type noOpValidator struct {
}

func (a *noOpValidator) validateKeyBlock(block *proto.Block) error {
	return nil
}

func (a *noOpValidator) validateMicroBlock(m *proto.MicroBlock) error {
	return nil
}
*/

func TestBlockSequence(t *testing.T) {
	b := newBlocks()
	require.Empty(t, b.Len())

	rs1, err := b.AddBlock(newBlock(sig1, emptySig))
	require.NoError(t, err)
	require.Equal(t, 1, rs1.Len())
	require.Equal(t, 0, b.Len())

	rs2, err := rs1.AddMicro(newMicro(sig2, sig1))
	require.NoError(t, err)
	require.Equal(t, 2, rs2.Len())

	rs3, err := rs2.AddMicro(newMicro(sig3, sig2))
	require.NoError(t, err)
	require.Equal(t, 3, rs3.Len())
	require.Equal(t, 1, rs1.Len())

	rs4, err := rs3.AddBlock(newBlock(sig4, sig1))
	require.NoError(t, err)
	require.Equal(t, 2, rs4.Len())
	require.Equal(t, 3, rs3.Len())

	row1, err := rs3.Row()
	require.NoError(t, err)
	require.Equal(t, sig1, row1.KeyBlock.BlockSignature)
	require.Equal(t, 2, len(row1.MicroBlocks))

	row2, err := rs4.Row()
	require.NoError(t, err)
	require.Equal(t, sig4, row2.KeyBlock.BlockSignature)
	require.Equal(t, 0, len(row2.MicroBlocks))
}

func TestBlocks_PreviousRow(t *testing.T) {
	b := newBlocks()
	rs1, err := b.AddBlock(newBlock(sig1, emptySig))
	require.NoError(t, err)

	rs2, err := rs1.AddMicro(newMicro(sig2, sig1))
	require.NoError(t, err)

	rs3, err := rs2.AddBlock(newBlock(sig3, sig2))
	require.NoError(t, err)

	row, err := rs3.PreviousRow()
	require.NoError(t, err)
	require.Equal(t, 1, len(row.MicroBlocks))
}

func TestStorage(t *testing.T) {
	s := newStorage(proto.TestNetScheme)

	require.NoError(t, s.PushBlock(newBlock(sig1, emptySig)))
	require.NoError(t, s.PushBlock(newBlock(sig2, sig1)))
	require.NoError(t, s.PushMicro(newMicro(sig3, sig2)))

	block1, err := s.Block()
	require.NoError(t, err)
	require.Equal(t, block1.BlockSignature, sig3)

	s.Pop()

	block2, err := s.Block()
	require.NoError(t, err)
	require.Equal(t, block2.BlockSignature, sig2)
}

func TestShrink(t *testing.T) {

	arr := make([]interface{}, 0, 150)
	for i := 0; i < 103; i++ {
		arr = append(arr, i)
	}
	resized := shrink(arr)
	require.Equal(t, 100, len(resized))
	require.Equal(t, 3, resized[0])
	require.Equal(t, 102, resized[99])
}

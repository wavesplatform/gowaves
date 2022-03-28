package blocks_applier

import (
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var genesisSign = crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY")
var genesisId = proto.NewBlockIDFromSignature(genesisSign)
var genesis = &proto.Block{
	BlockHeader: proto.BlockHeader{
		Timestamp:      1558613307877,
		BlockSignature: genesisSign,
		NxtConsensus: proto.NxtConsensus{
			BaseTarget:   153722867,
			GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
		},
	},
}

func TestApply_ValidBlockWithRollback(t *testing.T) {
	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisId,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}
	block2 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisId,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 50,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"),
		},
	}

	mockState, err := NewMockStateManager(genesis, block1)
	require.NoError(t, err)

	ba := innerBlocksApplier{}
	height, err := ba.apply(mockState, []*proto.Block{block2})
	require.NoError(t, err)
	require.EqualValues(t, 2, height)
	newBlock, _ := mockState.BlockByHeight(2)
	require.Equal(t, crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"), newBlock.BlockSignature)
}

// in this test we check that block rollback previous deleted block, when try to add new block.
// Emulate new blocks have error, so we can't accept them, and rollbacked blocks apply again
func TestApply_InvalidBlockWithRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisId,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}
	block2 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisId,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 50,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"),
		},
	}

	stateMock := mock.NewMockState(ctrl)
	stateMock.EXPECT().Block(block2.BlockID()).Return(nil, proto.ErrNotFound)
	stateMock.EXPECT().Height().Return(proto.Height(2), nil)
	// this returns current height
	stateMock.EXPECT().ScoreAtHeight(proto.Height(2)).Return(big.NewInt(2), nil)
	// returns parent height for block we insert, it will be genesis height, 1
	stateMock.EXPECT().BlockIDToHeight(genesisId).Return(proto.Height(1), nil)
	// returns score for genesis block, it will be 1
	stateMock.EXPECT().ScoreAtHeight(proto.Height(1)).Return(big.NewInt(1), nil)
	// now we save block for rollback
	stateMock.EXPECT().BlockByHeight(proto.Height(2)).Return(block1, nil)
	// rollback to first(genesis) block
	stateMock.EXPECT().RollbackToHeight(proto.Height(1)).Return(nil)
	// adding new blocks, and have error on applying
	stateMock.EXPECT().AddDeserializedBlocks([]*proto.Block{block2}).Return(nil, errors.New("error message"))
	// return blocks
	stateMock.EXPECT().AddDeserializedBlocks([]*proto.Block{block1}).Return(nil, nil)

	ba := innerBlocksApplier{}
	_, err := ba.apply(stateMock, []*proto.Block{block2})
	require.NotNil(t, err)
	require.Equal(t, "failed add deserialized blocks, first block id sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW: error message", err.Error())
}

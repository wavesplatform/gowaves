package node

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

var genesisSign = crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY")
var genesis = &proto.Block{
	BlockHeader: proto.BlockHeader{
		Timestamp:      1558613307877,
		BlockSignature: genesisSign,
		NxtConsensus: proto.NxtConsensus{
			BaseTarget:   153722867,
			GenSignature: crypto.MustDigestFromBase58("11111111111111111111111111111111"),
		},
	},
}
var genesisScore int64 = 120000000219

func TestApply_NewBlock(t *testing.T) {
	block := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
		},
	}

	mockState := NewMockStateManager(genesis)
	ba := innerBlockApplier{mockState}
	block, height, err := ba.apply(BlockWithBytes{Block: block})
	require.NoError(t, err)
	require.EqualValues(t, 2, height)
}

func TestApply_ValidBlockWithRollback(t *testing.T) {
	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}
	block2 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 50,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"),
		},
	}
	block2Bytes, err := block2.MarshalBinary()
	require.NoError(t, err)

	mockState := NewMockStateManager(genesis, block1)

	ba := innerBlockApplier{mockState}
	_, height, err := ba.apply(BlockWithBytes{Block: block2, Bytes: block2Bytes})
	require.NoError(t, err)
	require.EqualValues(t, 2, height)
	newBlock, _ := mockState.BlockByHeight(2)
	require.Equal(t, crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"), newBlock.BlockSignature)
}

func TestApply_InvalidBlockWithRollback(t *testing.T) {
	errorMessage := "error message"
	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}
	block2 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 50,
			},
			TransactionBlockLength: 4,
			BlockSignature:         crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"),
		},
	}
	block2Bytes, err := block2.MarshalBinary()
	require.NoError(t, err)

	// make second block invalid
	_addBlockFunc := func(a *MockStateManager, block []byte) (*proto.Block, error) {
		sig, err := proto.BlockGetSignature(block)
		if err != nil {
			return nil, err
		}
		if sig == crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW") {
			return nil, errors.New(errorMessage)
		}
		return DefaultAddBlockFunc(a, block)
	}
	mockState := NewMockStateManagerWithAddBlock(_addBlockFunc, genesis, block1)

	ba := innerBlockApplier{mockState}
	_, _, err = ba.apply(BlockWithBytes{Block: block2, Bytes: block2Bytes})
	require.EqualError(t, err, errorMessage)
	// check new block was not added
	newBlock, err := mockState.BlockByHeight(2)
	require.NoError(t, err)
	require.Equal(t, crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"), newBlock.BlockSignature)
}

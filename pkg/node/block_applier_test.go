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
			GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
		},
	},
}

func TestApply_NewBlock(t *testing.T) {
	block := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}

	mockState, err := NewMockStateManager(genesis)
	require.NoError(t, err)
	ba := innerBlockApplier{mockState}
	_, height, err := ba.apply(block)
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

	mockState, err := NewMockStateManager(genesis, block1)
	require.NoError(t, err)

	ba := innerBlockApplier{mockState}
	_, height, err := ba.apply(block2)
	require.NoError(t, err)
	require.EqualValues(t, 2, height)
	newBlock, _ := mockState.BlockByHeight(2)
	require.Equal(t, crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW"), newBlock.BlockSignature)
}

var errorMessage = "error message"

type checkErrMock struct {
	*MockStateManager
}

func (a *checkErrMock) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	if block.BlockSignature == crypto.MustSignatureFromBase58("sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW") {
		return nil, errors.New(errorMessage)
	}
	return a.MockStateManager.AddDeserializedBlock(block)
}

func TestApply_InvalidBlockWithRollback(t *testing.T) {

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

	sm, err := NewMockStateManager(genesis, block1)
	require.NoError(t, err)
	mockState := &checkErrMock{sm}

	ba := innerBlockApplier{mockState}
	_, _, err = ba.apply(block2)
	require.Equal(t, "failed add deserialized block \"sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW\": error message", err.Error())
	// check new block was not added
	newBlock, err := mockState.BlockByHeight(2)
	require.NoError(t, err)
	require.Equal(t, crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"), newBlock.BlockSignature)
}

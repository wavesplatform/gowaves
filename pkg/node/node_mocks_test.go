package node

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math/big"
	"testing"
)

func TestMockStateManager_ScoreAtHeight(t *testing.T) {
	genesis := &proto.Block{
		BlockHeader: proto.BlockHeader{
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 153722867,
			},
			BlockSignature: crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY"),
		},
	}

	m := NewMockStateManager(genesis)
	score, _ := m.ScoreAtHeight(1)
	require.Equal(t, big.NewInt(120000000219), score)
}

func TestMockStateManager_RollbackToHeight(t *testing.T) {
	genesis := &proto.Block{
		BlockHeader: proto.BlockHeader{
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 153722867,
			},
			BlockSignature: crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY"),
		},
	}
	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}

	m := NewMockStateManager(genesis, block1)
	actualHeight, _ := m.Height()
	require.EqualValues(t, 2, actualHeight)
	m.RollbackToHeight(1)
	actualHeight, _ = m.Height()
	require.EqualValues(t, 1, actualHeight)
}

func TestMockStateManager_BlockByHeight(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY")
	genesis := &proto.Block{
		BlockHeader: proto.BlockHeader{
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 153722867,
			},
			BlockSignature: sig,
		},
	}

	m := NewMockStateManager(genesis)
	block, _ := m.BlockByHeight(1)
	require.Equal(t, sig, block.BlockSignature)
}

func TestMockStateManager_DuplicateBlock(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY")
	genesis := &proto.Block{
		BlockHeader: proto.BlockHeader{
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 153722867,
			},
			BlockSignature: sig,
		},
	}

	require.Panics(t, func() {
		NewMockStateManager(genesis, genesis)
	})
}

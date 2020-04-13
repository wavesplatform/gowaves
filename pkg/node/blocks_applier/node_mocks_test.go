package blocks_applier

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

	m, err := NewMockStateManager(genesis)
	require.NoError(t, err)
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
			Parent: genesisId,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			BlockSignature: crypto.MustSignatureFromBase58("5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F"),
		},
	}

	m, err := NewMockStateManager(genesis, block1)
	require.NoError(t, err)
	actualHeight, _ := m.Height()
	require.EqualValues(t, 2, actualHeight)
	err = m.RollbackToHeight(1)
	require.NoError(t, err)
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

	m, err := NewMockStateManager(genesis)
	require.NoError(t, err)
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
		_, err := NewMockStateManager(genesis, genesis)
		t.Fatalf("Error: %v\n", err)
	})
}

package node

import (
	"math/big"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type item struct {
	block  *proto.Block
	height uint64
}

func blocksMap() (map[proto.BlockID]item, *proto.Block, *proto.Block) {
	genesisSign := crypto.MustSignatureFromBase58(
		"31oSQjtBqNjyj37qmrkocHvoazMtycbaw1shbznXoN66d3nfwczqTr4FKdGmqvaVGyxtrpiKdF6RGiZWNa9rEEkY",
	)
	genesisID := proto.NewBlockIDFromSignature(genesisSign)
	genesis := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Timestamp:      1558613307877,
			BlockSignature: genesisSign,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget:   153722867,
				GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
			},
		},
	}
	block1 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisID,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 100,
			},
			TransactionBlockLength: 4,
			BlockSignature: crypto.MustSignatureFromBase58(
				"5z4Ny16o9ED9PG8z4LDnAmPBaQcmDztAeU3Lbz1YBM6q4971BzN71aLX5hYdxK19fpCPkA4NAPcwjyWWD68SWb1F",
			),
		},
	}
	block2 := &proto.Block{
		BlockHeader: proto.BlockHeader{
			Parent: genesisID,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget: 50,
			},
			TransactionBlockLength: 4,
			BlockSignature: crypto.MustSignatureFromBase58(
				"sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW",
			),
		},
	}
	bm := map[proto.BlockID]item{genesisID: {block: genesis, height: 1}}
	return bm, block1, block2
}

func maxHeight(bm map[proto.BlockID]item) uint64 {
	var mh uint64
	for _, it := range bm {
		if it.height > mh {
			mh = it.height
		}
	}
	return mh
}

func TestApply_ValidBlockWithRollback(t *testing.T) {
	bm, _, block2 := blocksMap()
	ms := &MockState{
		BlockFunc: func(blockID proto.BlockID) (*proto.Block, error) {
			if it, ok := bm[blockID]; ok {
				return it.block, nil
			}
			return nil, keyvalue.ErrNotFound
		},
		AddDeserializedBlocksFunc: func(blocks []*proto.Block) (*proto.Block, error) {
			var last *proto.Block
			h := maxHeight(bm)
			for _, b := range blocks {
				h++
				bm[b.BlockID()] = item{block: b, height: h}
				last = b
			}
			return last, nil
		},
		HeightFunc: func() (uint64, error) {
			return maxHeight(bm), nil
		},
		ScoreAtHeightFunc: func(height uint64) (*big.Int, error) {
			if int(height) > len(bm) {
				return nil, errors.New("invalid test height")
			}
			i := big.NewInt(0).SetUint64(height * 10)
			return i, nil
		},
		BlockIDToHeightFunc: func(blockID proto.BlockID) (uint64, error) {
			for _, it := range bm {
				if it.block.BlockID() == blockID {
					return it.height, nil
				}
			}
			return 0, keyvalue.ErrNotFound
		},
		BlockByHeightFunc: func(height uint64) (*proto.Block, error) {
			for _, it := range bm {
				if it.height == height {
					return it.block, nil
				}
			}
			return nil, keyvalue.ErrNotFound
		},
	}

	a := NewApplier(ms)
	b, err := a.applyBlocks([]*proto.Block{block2})
	require.NoError(t, err)
	assert.Equal(t, block2, b)
	newBlock, err := ms.BlockByHeight(2)
	require.NoError(t, err)
	assert.Equal(t, "sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW",
		newBlock.BlockSignature.String())
}

// in this test we check that block rollback previous deleted block, when try to add new block.
// Emulate new blocks have error, so we can't accept them, and roll backed blocks apply again.
func TestApply_InvalidBlockWithRollback(t *testing.T) {
	bm, block1, block2 := blocksMap()
	ms := &MockState{
		BlockFunc: func(_ proto.BlockID) (*proto.Block, error) {
			return nil, proto.ErrNotFound
		},
		AddDeserializedBlocksFunc: func(blocks []*proto.Block) (*proto.Block, error) {
			var last *proto.Block
			for _, b := range blocks {
				last = b
				if b == block2 {
					return nil, errors.New("error message")
				}
			}
			return last, nil
		},
		HeightFunc: func() (uint64, error) {
			return 2, nil
		},
		ScoreAtHeightFunc: func(height uint64) (*big.Int, error) {
			if height > 2 {
				return nil, errors.New("invalid test height")
			}
			i := big.NewInt(0).SetUint64(height * 10)
			return i, nil
		},
		BlockIDToHeightFunc: func(blockID proto.BlockID) (uint64, error) {
			if it, ok := bm[blockID]; ok {
				return it.height, nil
			}
			return 0, proto.ErrNotFound
		},
		BlockByHeightFunc: func(height uint64) (*proto.Block, error) {
			if height == 2 {
				return block1, nil
			}
			return nil, proto.ErrNotFound
		},
		RollbackToHeightFunc: func(height uint64) error {
			if height == 1 {
				return nil
			}
			return errors.New("invalid height")
		},
	}
	a := NewApplier(ms)
	_, err := a.applyBlocks([]*proto.Block{block2})
	require.NotNil(t, err)
	require.Equal(t, "failed add deserialized blocks, first block id "+
		"sV8beveiVKCiUn9BGZRgZj7V5tRRWPMRj1V9WWzKWnigtfQyZ2eErVXHi7vyGXj5hPuaxF9sGxowZr5XuD4UAwW: error message",
		err.Error())
}

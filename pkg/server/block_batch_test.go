package server

import (
	"fmt"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type batchTest struct {
	blocks []proto.Block
	expErr error
}

var tests = []batchTest{
	{
		blocks: []proto.Block{
			proto.Block{
				BlockSignature: proto.BlockID{1},
				Parent:         proto.BlockID{0},
			},
			proto.Block{
				BlockSignature: proto.BlockID{2},
				Parent:         proto.BlockID{1},
			},
			proto.Block{
				BlockSignature: proto.BlockID{3},
				Parent:         proto.BlockID{2},
			},
		},
		expErr: nil,
	},
	{
		blocks: []proto.Block{
			proto.Block{
				BlockSignature: proto.BlockID{1},
				Parent:         proto.BlockID{0},
			},
			proto.Block{
				BlockSignature: proto.BlockID{3},
				Parent:         proto.BlockID{2},
			}, proto.Block{
				BlockSignature: proto.BlockID{2},
				Parent:         proto.BlockID{1},
			},
		},
		expErr: nil,
	},
	{
		blocks: []proto.Block{
			proto.Block{
				BlockSignature: proto.BlockID{1},
				Parent:         proto.BlockID{0},
			},
			proto.Block{
				BlockSignature: proto.BlockID{3},
				Parent:         proto.BlockID{2},
			},
		},
		expErr: batchIncomplete,
	},
}

func TestBatch(t *testing.T) {
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := func() error {
				ids := make([]proto.BlockID, 0, len(test.blocks))
				for _, block := range test.blocks {
					ids = append(ids, block.BlockSignature)
				}

				batch, err := NewBatch(ids)
				if err != nil {
					return err
				}

				for i := range test.blocks {
					if err := batch.addBlock(&test.blocks[i]); err != nil {
						t.Error(err)
					}
				}

				orderedBatch, err := batch.orderedBatch()
				if err != nil {
					return err
				}
				for i := 1; i < len(orderedBatch); i++ {
					parent := orderedBatch[i-1]
					child := orderedBatch[i]

					if child.Parent != parent.BlockSignature {
						t.Errorf("wrong parent for block %v: want %v, have %v",
							child.BlockSignature, child.Parent, parent.BlockSignature)
					}
				}
				return nil
			}()

			if err != test.expErr {
				t.Error("unexpected error, want ", test.expErr, " have ", err)
			}
		})
	}
}

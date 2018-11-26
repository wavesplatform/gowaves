package server

import (
	"fmt"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type batchTest struct {
	blocks []proto.Block
	expErr error
}

var tests = []batchTest{
	{
		blocks: []proto.Block{
			{
				BlockSignature: crypto.Signature{1},
				Parent:         crypto.Signature{0},
			},
			{
				BlockSignature: crypto.Signature{2},
				Parent:         crypto.Signature{1},
			},
			{
				BlockSignature: crypto.Signature{3},
				Parent:         crypto.Signature{2},
			},
		},
		expErr: nil,
	},
	{
		blocks: []proto.Block{
			{
				BlockSignature: crypto.Signature{1},
				Parent:         crypto.Signature{0},
			},
			{
				BlockSignature: crypto.Signature{3},
				Parent:         crypto.Signature{2},
			}, {
				BlockSignature: crypto.Signature{2},
				Parent:         crypto.Signature{1},
			},
		},
		expErr: nil,
	},
	{
		blocks: []proto.Block{
			{
				BlockSignature: crypto.Signature{1},
				Parent:         crypto.Signature{0},
			},
			{
				BlockSignature: crypto.Signature{3},
				Parent:         crypto.Signature{2},
			},
		},
		expErr: batchIncomplete,
	},
}

func TestBatch(t *testing.T) {
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := func() error {
				ids := make([]crypto.Signature, 0, len(test.blocks))
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

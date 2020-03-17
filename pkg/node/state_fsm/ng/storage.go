package ng

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Blocks []interface{}

func shrink(sequence Blocks) Blocks {
	if l := len(sequence); l > 100 {
		return sequence.CutFirstRow()
	}
	return sequence
}

func (a Blocks) AddBlock(block *proto.Block) (Blocks, error) {
	if a.Len() == 0 {
		panic("blocks should not contain 0 elements")
	}
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockSignature == block.Parent {
				return shrink(append(a[:i:i], t, block)), nil
			}
		case *proto.MicroBlock:
			if t.TotalResBlockSigField == block.Parent {
				return shrink(append(a[:i:i], t, block)), nil
			}
		default:
			panic(fmt.Sprintf("invalid type %T", t))
		}
	}
	return a, errors.Errorf("parent %s not found", block.Parent.String())
}

// always first elemnt block
func (a Blocks) First() *proto.Block {
	return a[0].(*proto.Block)
}

func (a Blocks) CutFirstRow() Blocks {
	if a.Len() == 0 {
		panic("blocks len should be never eq 0")
	}

	for i, b := range a {
		if i == 0 { // 0 expect always block
			continue
		}
		if _, ok := b.(*proto.Block); ok {
			return a[i:]
		}
	}
	return a
}

func (a Blocks) AddMicro(micro *proto.MicroBlock) (Blocks, error) {
	if a.Len() == 0 {
		return nil, nil
	}
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockSignature == micro.PrevResBlockSigField {
				return shrink(append(a[:i:i], t, micro)), nil
			}
		case *proto.MicroBlock:
			if t.TotalResBlockSigField == micro.PrevResBlockSigField {
				return shrink(append(a[:i:i], t, micro)), nil
			}
		default:
			panic(fmt.Sprintf("invalid type %T", t))
		}
	}
	return a, errors.New("parent not found")
}

func (a Blocks) ContainsSig(sig crypto.Signature) bool {
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockSignature == sig {
				return true
			}
		case *proto.MicroBlock:
			if t.TotalResBlockSigField == sig {
				return true
			}
		default:
			continue
		}
	}
	return false
}

func (a Blocks) Len() int {
	return len(a)
}

//
//func newBlocks() Signatures {
//	return Signatures{}
//}

func NewBlocksFromBlock(block *proto.Block) Blocks {
	return []interface{}{block}
}

// block should always contain at least 1 row
func (a Blocks) Row() types.MicroblockRow {
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			return types.MicroblockRow{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:])...)}
		default:
			continue
		}
	}
	panic("no buildable row")
}

func (a Blocks) PreviousRow() (types.MicroblockRow, error) {
	lastBlock := 0
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if lastBlock != 0 {
				return types.MicroblockRow{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:lastBlock])...)}, nil
			}
			lastBlock = i
		default:
			continue
		}
	}
	return types.MicroblockRow{}, errors.New("no buildable row")
}

func inf2micro(in []interface{}) []*proto.MicroBlock {
	out := make([]*proto.MicroBlock, 0, len(in))
	for _, row := range in {
		out = append(out, row.(*proto.MicroBlock))
	}
	return out
}

//
//type storage struct {
//	curState  Signatures
//	prevState Signatures
//	// TODO add validation
//	//validator validator
//}
//
//func newStorage() *storage {
//	return &storage{}
//}
//
//func (a *storage) PushBlock(block *proto.Block) error {
//	state, err := a.curState.AddBlock(block)
//	if err != nil {
//		return err
//	}
//	a.prevState = a.curState
//	a.curState = state
//	return nil
//}
//
//func (a *storage) PushMicro(m *proto.MicroBlock) error {
//	state, err := a.curState.AddMicro(m)
//	if err != nil {
//		return err
//	}
//	/* wait for better times
//	row, err := a.curState.Row()
//	if err != nil {
//		return err
//	}
//	err = a.validator.validateMicro(row)
//	if err != nil {
//		zap.S().Error(err)
//		return err
//	}
//	*/
//
//	a.prevState = a.curState
//	a.curState = state
//	return nil
//}
//
//func (a *storage) Block() (*proto.Block, error) {
//	row, err := a.curState.Row()
//	if err != nil {
//		return nil, err
//	}
//	return a.fromRow(row)
//}
//
//func (a *storage) PreviousBlock() (*proto.Block, error) {
//	row, err := a.curState.PreviousRow()
//	if err != nil {
//		return nil, err
//	}
//	return a.fromRow(row)
//}
//
//func (a *storage) ContainsSig(sig crypto.Signature) bool {
//	return a.curState.ContainsSig(sig)
//}
//
//func (a *storage) fromRow(seq Row) (*proto.Block, error) {
//	var err error
//
//	keyBlock := seq.KeyBlock
//	t := keyBlock.Transactions
//	BlockSignature := keyBlock.BlockSignature
//	for _, row := range seq.MicroBlocks {
//		t = t.Join(row.Transactions)
//		BlockSignature = row.TotalResBlockSigField
//	}
//
//	block, err := proto.CreateBlock(
//		t,
//		keyBlock.Timestamp,
//		keyBlock.Parent,
//		keyBlock.GenPublicKey,
//		keyBlock.NxtConsensus,
//		keyBlock.Version,
//		keyBlock.Features,
//		keyBlock.RewardVote)
//	if err != nil {
//		return nil, err
//	}
//	block.BlockSignature = BlockSignature
//	return block, nil
//}
//
//func (a *storage) newFromBlock(block *proto.Block) *storage {
//	return &storage{
//		curState: NewBlocksFromBlock(block),
//		//validator: a.validator,
//	}
//}
//
//func (a *storage) Pop() {
//	a.curState = a.prevState
//	a.prevState = newBlocks()
//}

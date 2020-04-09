package ng

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Row struct {
	KeyBlock    *proto.Block
	MicroBlocks []*proto.MicroBlock
}

type Blocks []interface{}

func shrink(sequence Blocks) Blocks {
	if l := len(sequence); l > 100 {
		return sequence[l-100:]
	}
	return sequence
}

func (a Blocks) AddBlock(block *proto.Block) (Blocks, error) {
	if a.Len() == 0 {
		return []interface{}{block}, nil
	}
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockID() == block.Parent {
				return shrink(append(a[:i:i], t, block)), nil
			}
		case *proto.MicroBlock:
			if t.TotalBlockID == block.Parent {
				return shrink(append(a[:i:i], t, block)), nil
			}
		default:
			panic(fmt.Sprintf("invalid type %T", t))
		}
	}
	return nil, errors.Errorf("parent %s not found", block.Parent.String())
}

func (a Blocks) AddMicro(micro *proto.MicroBlock) (Blocks, error) {
	if a.Len() == 0 {
		return nil, nil
	}
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockID() == micro.Reference {
				return shrink(append(a[:i:i], t, micro)), nil
			}
		case *proto.MicroBlock:
			if t.TotalBlockID == micro.Reference {
				return shrink(append(a[:i:i], t, micro)), nil
			}
		default:
			panic(fmt.Sprintf("invalid type %T", t))
		}
	}
	return nil, errors.New("parent not found")
}

func (a Blocks) ContainsID(blockID proto.BlockID) bool {
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if t.BlockID() == blockID {
				return true
			}
		case *proto.MicroBlock:
			if t.TotalBlockID == blockID {
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

func newBlocks() Blocks {
	return Blocks{}
}

func NewBlocksFromBlock(block *proto.Block) Blocks {
	return []interface{}{block}
}

func (a Blocks) Row() (Row, error) {
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			return Row{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:])...)}, nil
		default:
			continue
		}
	}
	return Row{}, errors.New("no buildable row")
}

func (a Blocks) PreviousRow() (Row, error) {
	lastBlock := 0
	for i := len(a) - 1; i >= 0; i-- {
		switch t := a[i].(type) {
		case *proto.Block:
			if lastBlock != 0 {
				return Row{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:lastBlock])...)}, nil
			}
			lastBlock = i
		default:
			continue
		}
	}
	return Row{}, errors.New("no buildable row")
}

func inf2micro(in []interface{}) []*proto.MicroBlock {
	out := make([]*proto.MicroBlock, 0, len(in))
	for _, row := range in {
		out = append(out, row.(*proto.MicroBlock))
	}
	return out
}

type storage struct {
	curState  Blocks
	prevState Blocks
	scheme    proto.Scheme
	// TODO add validation
	//validator validator
}

func newStorage(scheme proto.Scheme) *storage {
	return &storage{scheme: scheme}
}

func (a *storage) PushBlock(block *proto.Block) error {
	state, err := a.curState.AddBlock(block)
	if err != nil {
		return err
	}
	a.prevState = a.curState
	a.curState = state
	return nil
}

func (a *storage) PushMicro(m *proto.MicroBlock) error {
	state, err := a.curState.AddMicro(m)
	if err != nil {
		return err
	}
	/* wait for better times
	row, err := a.curState.Row()
	if err != nil {
		return err
	}
	err = a.validator.validateMicro(row)
	if err != nil {
		zap.S().Error(err)
		return err
	}
	*/

	a.prevState = a.curState
	a.curState = state
	return nil
}

func (a *storage) Block() (*proto.Block, error) {
	row, err := a.curState.Row()
	if err != nil {
		return nil, err
	}
	return a.fromRow(row)
}

func (a *storage) PreviousBlock() (*proto.Block, error) {
	row, err := a.curState.PreviousRow()
	if err != nil {
		return nil, err
	}
	return a.fromRow(row)
}

func (a *storage) ContainsID(id proto.BlockID) bool {
	return a.curState.ContainsID(id)
}

func (a *storage) fromRow(seq Row) (*proto.Block, error) {
	var err error

	keyBlock := seq.KeyBlock
	t := keyBlock.Transactions
	BlockSignature := keyBlock.BlockSignature
	for _, row := range seq.MicroBlocks {
		t = t.Join(row.Transactions)
		BlockSignature = row.TotalResBlockSigField
	}

	block, err := proto.CreateBlock(
		t,
		keyBlock.Timestamp,
		keyBlock.Parent,
		keyBlock.GenPublicKey,
		keyBlock.NxtConsensus,
		keyBlock.Version,
		keyBlock.Features,
		keyBlock.RewardVote,
		a.scheme)
	if err != nil {
		return nil, err
	}
	block.BlockSignature = BlockSignature
	return block, nil
}

func (a *storage) newFromBlock(block *proto.Block) *storage {
	return &storage{
		curState: NewBlocksFromBlock(block),
		scheme:   a.scheme,
		//validator: a.validator,
	}
}

func (a *storage) Pop() {
	a.curState = a.prevState
	a.prevState = newBlocks()
}

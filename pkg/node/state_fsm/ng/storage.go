package ng

//
//import (
//	"fmt"
//
//	"github.com/pkg/errors"
//	"github.com/wavesplatform/gowaves/pkg/crypto"
//	"github.com/wavesplatform/gowaves/pkg/proto"
//	"go.uber.org/zap"
//)
//
//type Blocks []interface{}
//
//func shrink(sequence Blocks) Blocks {
//	if l := len(sequence); l > 100 {
//		return sequence.CutFirstRow()
//	}
//	return sequence
//}
//
//func (a Blocks) AddBlock(block *proto.Block) (Blocks, error) {
//	if a.Len() == 0 {
//		panic("blocks should not contain 0 elements")
//	}
//	for i := len(a) - 1; i >= 0; i-- {
//		switch t := a[i].(type) {
//		case *proto.Block:
//			if t.BlockID() == block.Parent {
//				return shrink(append(a[:i:i], t, block)), nil
//			}
//		case *proto.MicroBlock:
//			if t.TotalResBlockSigField == block.Parent {
//				return shrink(append(a[:i:i], t, block)), nil
//			}
//		default:
//			panic(fmt.Sprintf("invalid type %T", t))
//		}
//	}
//	return a, errors.Errorf("parent %s not found", block.Parent.String())
//}
//
//func (a Blocks) ForceAddBlock(block *proto.Block) Blocks {
//	new, err := a.AddBlock(block)
//	if err != nil {
//		zap.S().Warnf("a block ForceAddBlock err %q", err)
//		return NewBlocksFromBlock(block)
//	}
//	return new
//}
//
//// always first elemnt block
//func (a Blocks) First() *proto.Block {
//	return a[0].(*proto.Block)
//}
//
//func (a Blocks) CutFirstRow() Blocks {
//	if a.Len() == 0 {
//		panic("blocks len should be never eq 0")
//	}
//
//	for i, b := range a {
//		if i == 0 { // 0 expect always block
//			continue
//		}
//		if _, ok := b.(*proto.Block); ok {
//			return a[i:]
//		}
//	}
//	return a
//}
//
//func (a Blocks) AddMicro(micro *proto.MicroBlock) (Blocks, error) {
//	if a.Len() == 0 {
//		return nil, nil
//	}
//	for i := len(a) - 1; i >= 0; i-- {
//		switch t := a[i].(type) {
//		case *proto.Block:
//			if t.BlockID() == micro.Reference {
//				return shrink(append(a[:i:i], t, micro)), nil
//			}
//		case *proto.MicroBlock:
//			if t.TotalBlockID == micro.Reference {
//				return shrink(append(a[:i:i], t, micro)), nil
//			}
//		default:
//			panic(fmt.Sprintf("invalid type %T", t))
//		}
//	}
//	return a, errors.New("parent not found")
//}
//
//func (a Blocks) ContainsSig(sig crypto.Signature) bool {
//	for i := len(a) - 1; i >= 0; i-- {
//		switch t := a[i].(type) {
//		case *proto.Block:
//			if t.BlockSignature == sig {
//				return true
//			}
//		case *proto.MicroBlock:
//			if t.TotalResBlockSigField == sig {
//				return true
//			}
//		default:
//			continue
//		}
//	}
//	return false
//}
//
//func (a Blocks) Len() int {
//	return len(a)
//}
//
////
////func newBlocks() Signatures {
////	return Signatures{}
////}
//
//func NewBlocksFromBlock(block *proto.Block) Blocks {
//	return []interface{}{block}
//}
//
//// block should always contain at least 1 row
//func (a Blocks) Row() proto.MicroblockRow {
//	for i := len(a) - 1; i >= 0; i-- {
//		switch t := a[i].(type) {
//		case *proto.Block:
//			return proto.MicroblockRow{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:])...)}
//		default:
//			continue
//		}
//	}
//	panic("no buildable row")
//}
//
//func (a Blocks) PreviousRow() (proto.MicroblockRow, error) {
//	lastBlock := 0
//	for i := len(a) - 1; i >= 0; i-- {
//		switch t := a[i].(type) {
//		case *proto.Block:
//			if lastBlock != 0 {
//				return proto.MicroblockRow{KeyBlock: t, MicroBlocks: append([]*proto.MicroBlock(nil), inf2micro(a[i+1:lastBlock])...)}, nil
//			}
//			lastBlock = i
//		default:
//			continue
//		}
//	}
//	return proto.MicroblockRow{}, errors.New("no buildable row")
//}
//
//func inf2micro(in []interface{}) []*proto.MicroBlock {
//	out := make([]*proto.MicroBlock, 0, len(in))
//	for _, row := range in {
//		out = append(out, row.(*proto.MicroBlock))
//	}
//	return out
//}

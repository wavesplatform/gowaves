package ng

//import "github.com/wavesplatform/gowaves/pkg/proto"

//type ng interface {
//	// ng state from last inserted block
//	AppliedBlock(block *proto.Block) ng
//	MicroBlock(*proto.MicroBlock) (ng, error)
//	KeyBlock(block *proto.Block) (ng, error)
//}
//
//type Initial struct {
//	blocks Blocks
//}
//
//func (a *Initial) AppliedBlock(block *proto.Block) ng {
//	return NewNgFromAppliedBlock(block)
//}
//
//func (a *Initial) MicroBlock(micro *proto.MicroBlock) (ng, error) {
//	blocks, err := a.blocks.AddMicro(micro)
//	if err != nil {
//		return a, err
//	}
//	a.blocks = blocks
//	return a, nil
//}
//
//func (a *Initial) KeyBlock(block *proto.Block) (ng, error) {
//
//}
//
//func NewNgFromAppliedBlock(b *proto.Block) ng {
//	return &Initial{
//		blocks: NewBlocksFromBlock(b),
//	}
//}

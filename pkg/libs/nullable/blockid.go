package nullable

import "github.com/wavesplatform/gowaves/pkg/proto"

type BlockID struct {
	id   proto.BlockID
	null bool
}

func NewNullBlockID() BlockID {
	return BlockID{null: true}
}

func NewBlockID(id proto.BlockID) BlockID {
	return BlockID{
		id: id,
	}
}

func (a BlockID) Null() bool {
	return a.null
}

func (a BlockID) ID() proto.BlockID {
	return a.id
}

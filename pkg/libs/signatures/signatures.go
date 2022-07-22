package signatures

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

// from small to big by height
type NaturalOrdering = BlockIDs
type ReverseOrdering = BlockIDs

type BlockIDs struct {
	signatures []proto.BlockID
	unique     map[proto.BlockID]struct{}
}

func (a *BlockIDs) BlockIDS() []proto.BlockID {
	return a.signatures
}

func NewSignatures(signatures ...proto.BlockID) *NaturalOrdering {
	unique := make(map[proto.BlockID]struct{})
	for _, v := range signatures {
		unique[v] = struct{}{}
	}

	return &BlockIDs{
		signatures: signatures,
		unique:     unique,
	}
}

func (a *BlockIDs) Exists(sig proto.BlockID) bool {
	_, ok := a.unique[sig]
	return ok
}

func (a *BlockIDs) Revert() *ReverseOrdering {
	out := make([]proto.BlockID, len(a.signatures))
	for k, v := range a.signatures {
		out[len(a.signatures)-1-k] = v
	}
	return NewSignatures(out...)
}

func (a *BlockIDs) Len() int {
	return len(a.signatures)
}

type LastSignatures interface {
	LastBlockIDs(state storage.State) (*ReverseOrdering, error)
}

type LastSignaturesImpl struct {
}

func (LastSignaturesImpl) LastBlockIDs(state storage.State) (*ReverseOrdering, error) {
	var signatures []proto.BlockID

	height, err := state.Height()
	if err != nil {
		zap.S().Errorf("LastSignaturesImpl: failed to get height from state: %v", err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, err := state.HeightToBlockID(height)
		if err != nil {
			zap.S().Errorf("LastSignaturesImpl: failed to get blockID for height %d: %v", height, err)
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}

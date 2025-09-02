package signatures

import (
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
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
		slog.Error("LastBlockIDs: Failed to get height from state", logging.Error(err))
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, hErr := state.HeightToBlockID(height)
		if hErr != nil {
			slog.Error("LastBlockIDs: Failed to get blockID for height", slog.Any("height", height),
				logging.Error(hErr))
			return nil, hErr
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}

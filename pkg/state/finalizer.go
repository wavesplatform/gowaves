package state

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type finalizer struct {
	generators *generators
}

func newFinalizer(generators *generators) *finalizer {
	return &finalizer{
		generators: generators,
	}
}

func (f *finalizer) processBlockFinalization(finalizationVoting *proto.FinalizationVoting, blockID proto.BlockID) error {
	// First of all process conflicting endorsements.
	for _, ce := range finalizationVoting.ConflictEndorsements {
		if err := f.generators.banGenerator(ce.EndorserIndex, blockID); err != nil {
			return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
		}
	}
	// Check that other endorsers are valid to endorse the parent block.
	for _, ei := range finalizationVoting.EndorserIndexes {
		g, err := f.generators.generator(ei)
		if err != nil {
			return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
		}
	}
	// Check aggregate signature.
}

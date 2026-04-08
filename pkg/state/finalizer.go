package state

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type finalizer struct {
	generators *generators
	finality   *finality
}

func newFinalizer(generators *generators, finality *finality) *finalizer {
	return &finalizer{
		generators: generators,
		finality:   finality,
	}
}

// processBlockFinalization performs state updates required for block finalization
// and finalizes the block if all requirements are met.
//
// This function does not perform basic validation of FinalizationVoting,
// as it is handled earlier in the call stack by Validator.validateFinalizationVoting.
func (f *finalizer) processBlockFinalization(
	finalizationVoting proto.FinalizationVoting,
	block *proto.BlockHeader,
	height proto.Height,
) error {
	blockID := block.BlockID()
	// First of all process conflicting endorsements and ban endorsers that produced them.
	for _, ce := range finalizationVoting.ConflictEndorsements {
		if err := f.generators.banGenerator(ce.EndorserIndex, blockID); err != nil {
			return fmt.Errorf("failed to ban generator of conflicting endorsement: %w", err)
		}
	}
	// Check that other endorsers are valid to endorse the parent block.
	var endorsersBalance uint64 = 0
	pks := make([]bls.PublicKey, 0, f.generators.size())
	for _, ei := range finalizationVoting.EndorserIndexes {
		g, err := f.generators.generator(ei)
		if err != nil {
			return fmt.Errorf("failed to get generator by index %d: %w", ei, err)
		}
		if g.ban {
			return fmt.Errorf("banned generator '%s' finalization voting found in block '%s'",
				g.Address(), blockID)
		}
		balance := g.GenerationBalance()
		if balance == 0 {
			return fmt.Errorf("generator '%s' with invalid generation balance %d "+
				"found in finalization voting of block '%s'",
				g.Address(), balance, blockID)
		}
		pks = append(pks, g.BLSPublicKey())
		endorsersBalance += g.GenerationBalance()
	}
	// Check aggregate signature.
	msg, err := f.finality.buildLocalEndorsementMessage(height, block.Parent)
	if err != nil {
		return err
	}
	mb, err := msg.Bytes()
	if err != nil {
		return fmt.Errorf("failed to serialize local endorsement crypto message: %w", err)
	}
	if !bls.VerifyAggregate(pks, mb, finalizationVoting.AggregatedEndorsementSignature) {
		return fmt.Errorf("invalid aggregated signature of finalization voting of block '%s'", blockID)
	}

	// Check that the block is finalized.
	totalGenerationBalance, err := f.generators.TotalGenerationBalance()
	if err != nil {
		return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
	}
	if 3*endorsersBalance >= 2*totalGenerationBalance {
		if fErr := f.finality.updatePendingFinalization(height-1, blockID); fErr != nil {
			return fmt.Errorf("failed to update pending finalization of block '%s': %w", blockID, fErr)
		}
	}
	return nil
}

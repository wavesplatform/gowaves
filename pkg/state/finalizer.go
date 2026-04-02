package state

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type finalizer struct {
	generators *generators
	message    proto.EndorsementCryptoMessage
}

func newFinalizer(generators *generators) *finalizer {
	return &finalizer{
		generators: generators,
	}
}

func (f *finalizer) processBlockFinalization(finalizationVoting *proto.FinalizationVoting, blockID proto.BlockID) error {
	// First of all process conflicting endorsements and ban endorsers that produced them.
	for _, ce := range finalizationVoting.ConflictEndorsements {
		if err := f.generators.banGenerator(ce.EndorserIndex, blockID); err != nil {
			return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
		}
	}
	// Check that other endorsers are valid to endorse the parent block.
	var endorsersBalance uint64 = 0
	pks := make([]bls.PublicKey, 0, f.generators.size())
	for _, ei := range finalizationVoting.EndorserIndexes {
		g, err := f.generators.generator(ei)
		if err != nil {
			return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
		}
		if g.ban {
			//TODO: The block validation must fail on banned generator endorsement.
			return fmt.Errorf("banned generator '%s' finalization voting found in block '%s'",
				g.Address(), blockID)
		}
		//TODO: Check the generation balance of endorser is valid.
		pks = append(pks, g.BLSPublicKey())
		endorsersBalance += g.GenerationBalance()
	}
	// Check aggregate signature.
	msg, err := f.message.Bytes()
	if err != nil {
		return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
	}
	if !bls.VerifyAggregate(pks, msg, finalizationVoting.AggregatedEndorsementSignature) {
		return fmt.Errorf("invalid aggregated signature of finalization voting of block '%s'", blockID)
	}

	// Check that the block is finalized.
	totalGenerationBalance, err := f.generators.TotalGenerationBalance()
	if err != nil {
		return fmt.Errorf("failed to process finalization voting of block '%s': %w", blockID, err)
	}
	if 3*endorsersBalance >= 2*totalGenerationBalance {
		// TODO: Finalize the block.
	}
	return nil
}

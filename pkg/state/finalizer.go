package state

import (
	"errors"
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

func (f *finalizer) checkBlockFinalization(voting proto.FinalizationVoting) error {
	if l := len(voting.ConflictEndorsements); l > f.generators.size() {
		return fmt.Errorf("conflicting endorsements count %d exceeds generator set size %d",
			l, f.generators.size())
	}
	if l := len(voting.EndorserIndexes); l > f.generators.size() {
		return fmt.Errorf("endorsements count %d exceeds generator set size %d",
			l, f.generators.size())
	}
	return nil
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
	if err := f.checkBlockFinalization(finalizationVoting); err != nil {
		return err
	}
	blockID := block.BlockID()
	if err := f.processConflictingEndorsements(finalizationVoting.ConflictEndorsements, blockID); err != nil {
		return err
	}
	// Check that other endorsers are valid to endorse the parent block.
	var endorsersBalance uint64 = 0
	bg, err := f.generators.blockGenerator()
	if err != nil {
		return fmt.Errorf("failed to get block generator: %w", err)
	}
	blockGeneratorIndex := bg.index
	pks := make([]bls.PublicKey, 0, f.generators.size())
	for _, ei := range finalizationVoting.EndorserIndexes {
		g, gErr := f.generators.generator(ei)
		if gErr != nil {
			return fmt.Errorf("failed to get endorser by index %d: %w", ei, gErr)
		}
		if g.ban {
			return fmt.Errorf("banned generator '%s' finalization voting found", g.Address())
		}
		if blockGeneratorIndex == ei {
			return fmt.Errorf("block generator '%s' found in finalization voting", bg.Address())
		}
		balance := g.GenerationBalance()
		if balance == 0 {
			return fmt.Errorf("generator '%s' with insufficient generation balance found in finalization voting",
				g.Address())
		}
		pks = append(pks, g.BLSPublicKey())
		endorsersBalance += balance
	}
	// Add block generator's balance to endorsers balance.
	endorsersBalance += bg.GenerationBalance() // Balance of block generator already checked.
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
		return errors.New("invalid aggregated signature of finalization voting")
	}

	// Check that the block is finalized.
	if 3*endorsersBalance >= 2*f.generators.totalGenerationBalance() {
		if fErr := f.finality.updatePendingFinalization(height-1, blockID); fErr != nil {
			return fmt.Errorf("failed to update pending finalization: %w", fErr)
		}
	}
	return nil
}

func (f *finalizer) processConflictingEndorsements(
	conflictingEndorsements []proto.BlockEndorsement, blockID proto.BlockID,
) error {
	for _, ce := range conflictingEndorsements {
		// Check the signature of conflicting endorsement.
		cmb, err := ce.CryptoMessage().Bytes()
		if err != nil {
			return fmt.Errorf("failed to check conflicting endorsement: %w", err)
		}
		gi, err := f.generators.generator(ce.EndorserIndex)
		if err != nil {
			return fmt.Errorf("failed to get generator of conflicting endorsement by index %d: %w",
				ce.EndorserIndex, err)
		}
		valid, err := bls.Verify(gi.BLSPublicKey(), cmb, ce.Signature)
		if err != nil {
			return fmt.Errorf("failed to verify conflicting endorsement signature: %w", err)
		}
		if !valid {
			return fmt.Errorf("conflicting endorsement signature is invalid")
		}
		// Ban generator of conflicting endorsement.
		if bErr := f.generators.banGenerator(ce.EndorserIndex, blockID); bErr != nil {
			return fmt.Errorf("failed to ban generator of conflicting endorsement: %w", bErr)
		}
	}
	return nil
}

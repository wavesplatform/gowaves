package state

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type finalizer struct {
	generators *generatorsStorage
	finality   *finality
}

func newFinalizer(generators *generatorsStorage, finality *finality) *finalizer {
	return &finalizer{
		generators: generators,
		finality:   finality,
	}
}

func (f *finalizer) checkBlockFinalization(voting proto.FinalizationVoting, height proto.Height) error {
	if voting.FinalizedBlockHeight >= height {
		return fmt.Errorf("invalid finalization voting: finalized block height %d should be less than block height %d",
			voting.FinalizedBlockHeight, height)
	}
	gs, err := f.generators.generators(height)
	if err != nil {
		if errors.Is(err, ErrNoGeneratorsSet) {
			return voting.CheckSizes(0)
		}
		return fmt.Errorf("failed to check block finalization: %w", err)
	}
	return voting.CheckSizes(gs.Size())
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
	if err := f.checkBlockFinalization(finalizationVoting, height); err != nil {
		return err
	}
	blockID := block.BlockID()
	if err := f.processConflictingEndorsements(finalizationVoting.ConflictEndorsements, height, blockID); err != nil {
		return err
	}
	// Check that other endorsers are valid to endorse the parent block.
	var endorsersBalance uint64 = 0
	bgi, bg, err := f.generators.blockGenerator(height)
	if err != nil {
		return fmt.Errorf("failed to get block generator: %w", err)
	}
	pks := make([]bls.PublicKey, 0, len(finalizationVoting.EndorserIndexes))
	for _, ei := range finalizationVoting.EndorserIndexes {
		g, gErr := f.generators.generator(ei, height)
		if gErr != nil {
			return fmt.Errorf("failed to get endorser by index %d: %w", ei, gErr)
		}
		if g.Ban {
			return fmt.Errorf("banned generator '%s' finalization voting found", g.Address.String())
		}
		if bgi == ei {
			return fmt.Errorf("block generator with index %d found in finalization voting", bgi)
		}
		balance := g.Balance
		if balance == 0 {
			return fmt.Errorf("generator '%s' with insufficient generation balance found in finalization voting",
				g.Address.String())
		}
		pks = append(pks, g.BLSPublicKey)
		endorsersBalance += balance
	}
	// Add block generator's balance to endorsers balance.
	endorsersBalance += bg.Balance // Balance of block generator already checked.
	// Check aggregate signature.
	msg, err := f.finality.buildRemoteEndorsementMessage(finalizationVoting.FinalizedBlockHeight, block.Parent)
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

	// A block is considered finalized if the total endorsers' balance is at least 2/3 of the committed
	// generators' total balance.
	totalGenerationBalance, err := f.generators.totalGenerationBalance(height)
	if err != nil {
		return err
	}
	if 3*endorsersBalance >= 2*totalGenerationBalance {
		finalizedHeight := height - 1
		slog.Debug("Block finalization achieved", slog.Uint64("finalizedHeight", finalizedHeight),
			slog.Uint64("blockHeight", height), slog.String("blockID", blockID.String()))
		if fErr := f.finality.updatePendingFinalization(finalizedHeight, blockID); fErr != nil {
			return fmt.Errorf("failed to update pending finalization: %w", fErr)
		}
	}
	return nil
}

func (f *finalizer) processConflictingEndorsements(
	conflictingEndorsements []proto.BlockEndorsement, blockHeight proto.Height, blockID proto.BlockID,
) error {
	for _, ce := range conflictingEndorsements {
		// Check the signature of conflicting endorsement.
		cmb, err := ce.CryptoMessage().Bytes()
		if err != nil {
			return fmt.Errorf("failed to build crypto message for conflicting endorsement with index %d: %w",
				ce.EndorserIndex, err)
		}
		gi, err := f.generators.generator(ce.EndorserIndex, blockHeight)
		if err != nil {
			return fmt.Errorf("failed to get generator for conflicting endorsement with index %d: %w",
				ce.EndorserIndex, err)
		}
		valid, err := bls.Verify(gi.BLSPublicKey, cmb, ce.Signature)
		if err != nil {
			return fmt.Errorf("failed to verify signature of conflicting endorsement with index %d: %w",
				ce.EndorserIndex, err)
		}
		if !valid {
			return fmt.Errorf("invalid signature of conflicting endorsement with index %d",
				ce.EndorserIndex)
		}
		// Ban generator of conflicting endorsement.
		if bErr := f.generators.banGenerator(ce.EndorserIndex, blockHeight, blockID); bErr != nil {
			return fmt.Errorf("failed to ban generator of conflicting endorsement with index %d: %w",
				ce.EndorserIndex, bErr)
		}
	}
	return nil
}

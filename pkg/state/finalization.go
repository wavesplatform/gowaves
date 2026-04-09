package state

import (
	"fmt"
	"log/slog"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNoFinalization = errors.New("no finalized block recorded")
var ErrNoFinalizationHistory = errors.New("no finalization in history")

// finalizationRecord stores finalized height and pending (pre-finalized) height.
type finalizationRecord struct {
	FinalizedBlockHeight proto.Height `cbor:"0,keyasint,omitempty"`
	PendingBlockHeight   proto.Height `cbor:"1,keyasint,omitempty"`
}

func (fr *finalizationRecord) marshalBinary() ([]byte, error) {
	return cbor.Marshal(fr)
}

func (fr *finalizationRecord) unmarshalBinary(data []byte) error {
	return cbor.Unmarshal(data, fr)
}

type finality struct {
	hs *historyStorage
	rw *blockReadWriter
}

func newFinality(hs *historyStorage, rw *blockReadWriter) *finality {
	return &finality{hs: hs, rw: rw}
}

func (f *finality) newestRecord() (*finalizationRecord, error) {
	data, err := f.hs.newestTopEntryData([]byte{finalizationKeyPrefix})
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, ErrNoFinalizationHistory
		}
		return nil, fmt.Errorf("failed to retrieve finalization record: %w", err)
	}
	var rec finalizationRecord
	if unmarshalErr := rec.unmarshalBinary(data); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal finalization record: %w", unmarshalErr)
	}
	return &rec, nil
}

func (f *finality) writeRecord(rec *finalizationRecord, currentBlockID proto.BlockID) error {
	newData, err := rec.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal finalization record: %w", err)
	}
	if addErr := f.hs.addNewEntry(finalization, []byte{finalizationKeyPrefix}, newData, currentBlockID); addErr != nil {
		return fmt.Errorf("failed to add finalization record: %w", addErr)
	}
	return nil
}

// updateFinalization promotes pending finalization value to regular if pending value is set (i.e. not zero).
// Must be executed without conditions before new block applying.
// TODO: what block ID should be provided: applying one or its parent?
func (f *finality) updateFinalization(applyingBlockID proto.BlockID) error {
	rec, err := f.newestRecord()
	if err != nil {
		if !errors.Is(err, ErrNoFinalization) && !errors.Is(err, ErrNoFinalizationHistory) {
			return fmt.Errorf("failed to retrieve finalization record for update: %w", err)
		}
		rec = &finalizationRecord{} // no record found, create an empty one
	}
	if rec.PendingBlockHeight == 0 {
		return nil // nothing to do if no pending value has been stored before
	}
	rec = &finalizationRecord{
		FinalizedBlockHeight: rec.PendingBlockHeight, // promote pending value to finalized
		PendingBlockHeight:   0,
	}
	return f.writeRecord(rec, applyingBlockID)
}

// updatePendingFinalization sets pending finalization value for the current block's parent height.
// Must be executed after new block applying ONLY if current block applying has finalized its parent.
func (f *finality) updatePendingFinalization(
	parentHeight proto.Height, // i.e. finalized block height
	applyingBlockID proto.BlockID,
) error {
	rec, err := f.newestRecord()
	if err != nil {
		if !errors.Is(err, ErrNoFinalization) && !errors.Is(err, ErrNoFinalizationHistory) {
			return fmt.Errorf("failed to retrieve finalization record for update pending: %w", err)
		}
		rec = &finalizationRecord{} // no record found, create an empty one
	}
	if prevP := rec.PendingBlockHeight; prevP != 0 { // sanity check
		return fmt.Errorf("pending finalization already exists with height %d", prevP)
	}
	rec = &finalizationRecord{
		FinalizedBlockHeight: rec.FinalizedBlockHeight, // finalization value still the same
		PendingBlockHeight:   parentHeight,             // only update pending
	}
	return f.writeRecord(rec, applyingBlockID)
}

// forceWrite writes finalization record with provided
// finalized block height and zero pending height, without any checks.
func (f *finality) forceWrite(finalizedBlockHeight proto.Height, currentBlockID proto.BlockID) error {
	rec := &finalizationRecord{
		FinalizedBlockHeight: finalizedBlockHeight,
		PendingBlockHeight:   0, // no pending
	}
	return f.writeRecord(rec, currentBlockID)
}

// newestHeight returns last finalized height value.
func (f *finality) newestHeight() (proto.Height, error) {
	rec, err := f.newestRecord()
	if err != nil {
		return 0, err
	}
	finH := rec.FinalizedBlockHeight
	if finH == 0 { // handle case when finalization record exists but no finalized block height has been stored yet
		return 0, ErrNoFinalization
	}
	return finH, nil
}

// lastFinalizedHeight returns stored or calculated height of last finalized block.
func (f *finality) lastFinalizedHeight(height proto.Height) (proto.Height, error) {
	calculated := calculateLastFinalizedHeight(height)
	stored, err := f.newestHeight()
	if err != nil {
		if errors.Is(err, ErrNoFinalization) || errors.Is(err, ErrNoFinalizationHistory) {
			return calculated, nil
		}
		return 0, fmt.Errorf("failed to retrieve last finalized height: %w", err)
	}
	if stored >= calculated {
		return stored, nil
	}
	return calculated, nil
}

func (f *finality) buildLocalEndorsementMessage(
	height proto.Height, parentID proto.BlockID,
) (proto.EndorsementCryptoMessage, error) {
	finalizedHeight, err := f.lastFinalizedHeight(height)
	if err != nil {
		return proto.EndorsementCryptoMessage{}, fmt.Errorf("failed to build local endorsement message: %w", err)
	}
	finalizedBlockID, err := f.rw.newestBlockIDByHeight(finalizedHeight)
	if err != nil {
		return proto.EndorsementCryptoMessage{}, fmt.Errorf("failed to build local endorsement message: %w", err)
	}
	slog.Debug("Local finalization state",
		slog.Uint64("finalizedHeight", finalizedHeight),
		slog.String("finalizedBlockID", finalizedBlockID.String()),
	)
	fh, err := safecast.Convert[uint32](finalizedHeight)
	if err != nil {
		return proto.EndorsementCryptoMessage{}, fmt.Errorf("failed to build local endorsement message: %w", err)
	}
	return proto.EndorsementCryptoMessage{
		FinalizedBlockID:     finalizedBlockID,
		FinalizedBlockHeight: fh,
		EndorsedBlockID:      parentID,
	}, nil
}

func calculateLastFinalizedHeight(height proto.Height) proto.Height {
	const genesisHeight uint64 = 1
	const maxRollbackDeltaHeight uint64 = 100
	if height <= maxRollbackDeltaHeight {
		return genesisHeight
	}
	return height - maxRollbackDeltaHeight
}

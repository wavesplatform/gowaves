package state

import (
	"fmt"

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

type finalizations struct {
	hs *historyStorage
}

func newFinalizations(hs *historyStorage) *finalizations {
	return &finalizations{hs: hs}
}

func (f *finalizations) newestRecord() (*finalizationRecord, error) {
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

func (f *finalizations) writeRecord(rec *finalizationRecord, currentBlockID proto.BlockID) error {
	newData, err := rec.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal finalization record: %w", err)
	}
	if addErr := f.hs.addNewEntry(finalization, []byte{finalizationKeyPrefix}, newData, currentBlockID); addErr != nil {
		return fmt.Errorf("failed to add finalization record: %w", addErr)
	}
	return nil
}

func (f *finalizations) newestHeightForProcessing(rec *finalizationRecord) (proto.Height, error) {
	if rec.FinalizedBlockHeight == 0 && rec.PendingBlockHeight == 0 {
		return 0, ErrNoFinalization
	}
	if rec.PendingBlockHeight > rec.FinalizedBlockHeight {
		return rec.PendingBlockHeight, nil
	}
	return rec.FinalizedBlockHeight, nil
}

func (f *finalizations) newestVisibleHeight(rec *finalizationRecord, currentHeight proto.Height) (proto.Height, error) {
	if rec.FinalizedBlockHeight == 0 && rec.PendingBlockHeight == 0 {
		return 0, ErrNoFinalization
	}
	if rec.PendingBlockHeight != 0 && currentHeight >= rec.PendingBlockHeight+2 {
		if rec.PendingBlockHeight > rec.FinalizedBlockHeight {
			return rec.PendingBlockHeight, nil
		}
	}
	if rec.FinalizedBlockHeight == 0 {
		return 0, ErrNoFinalization
	}
	return rec.FinalizedBlockHeight, nil
}

// store writes pending finalization for the current block and promotes matured pending value.
func (f *finalizations) store(
	finalizedBlockHeight proto.Height,
	currentHeight proto.Height,
	currentBlockID proto.BlockID,
) error {
	rec, err := f.newestRecord()
	if err != nil {
		if errors.Is(err, ErrNoFinalization) || errors.Is(err, ErrNoFinalizationHistory) {
			rec = &finalizationRecord{}
		} else {
			return err
		}
	}
	if rec.PendingBlockHeight != 0 && currentHeight >= rec.PendingBlockHeight+2 {
		if rec.PendingBlockHeight > rec.FinalizedBlockHeight {
			rec.FinalizedBlockHeight = rec.PendingBlockHeight
		}
	}
	if currentHeight >= finalizedBlockHeight+2 && finalizedBlockHeight > rec.FinalizedBlockHeight {
		rec.FinalizedBlockHeight = finalizedBlockHeight
	}
	rec.PendingBlockHeight = finalizedBlockHeight
	return f.writeRecord(rec, currentBlockID)
}

// newestForProcessing returns latest known finalization immediately, including pre-finalized value.
func (f *finalizations) newestForProcessing() (proto.Height, error) {
	rec, err := f.newestRecord()
	if err != nil {
		return 0, err
	}
	return f.newestHeightForProcessing(rec)
}

// newestVisible returns delayed finalization height which is exposed outside finalization processing.
func (f *finalizations) newestVisible(currentHeight proto.Height) (proto.Height, error) {
	rec, err := f.newestRecord()
	if err != nil {
		return 0, err
	}
	return f.newestVisibleHeight(rec, currentHeight)
}

// newest keeps backward-compatible semantics for internal callers:
// return latest known finalization immediately.
func (f *finalizations) newest() (proto.Height, error) {
	return f.newestForProcessing()
}

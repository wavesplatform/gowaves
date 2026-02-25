package state

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNoFinalization = errors.New("no finalized block recorded")
var ErrNoFinalizationHistory = errors.New("no finalization in history")

// finalizationRecord stores only last finalized height.
type finalizationRecord struct {
	FinalizedBlockHeight proto.Height `cbor:"0,keyasint,omitempty"`
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

// store replaces existing finalization with a new height.
func (f *finalizations) store(
	finalizedBlockHeight proto.Height,
	currentBlockID proto.BlockID,
) error {
	key := finalizationKey{}

	rec := finalizationRecord{
		FinalizedBlockHeight: finalizedBlockHeight,
	}

	newData, err := rec.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal finalization record: %w", err)
	}

	if addErr := f.hs.addNewEntry(finalization, key.bytes(), newData, currentBlockID); addErr != nil {
		return fmt.Errorf("failed to add finalization record: %w", addErr)
	}

	return nil
}

// newest returns the last finalized height.
func (f *finalizations) newest() (proto.Height, error) {
	key := finalizationKey{}
	data, err := f.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return 0, ErrNoFinalizationHistory
		}
		return 0, fmt.Errorf("failed to retrieve finalization record: %w", err)
	}

	var rec finalizationRecord
	if unmarshalErr := rec.unmarshalBinary(data); unmarshalErr != nil {
		return 0, fmt.Errorf("failed to unmarshal finalization record: %w", unmarshalErr)
	}

	if rec.FinalizedBlockHeight == 0 {
		return 0, ErrNoFinalization
	}

	return rec.FinalizedBlockHeight, nil
}

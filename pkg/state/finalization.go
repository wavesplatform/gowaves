package state

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const finalizationKey = "finalization"

var ErrNoFinalization = errors.New("no finalized blocks recorded")
var ErrNoFinalizationHistory = errors.New("no finalization in history")

type finalizationItem struct {
	Block                proto.Block  `cbor:"0,keyasint,omitempty"`
	FinalizedBlockHeight proto.Height `cbor:"1,keyasint,omitempty"`
}

type finalizationRecord struct {
	Records []finalizationItem `cbor:"0,keyasint,omitempty"`
}

func (fr *finalizationRecord) append(block proto.Block, finalizedBlockHeight proto.Height) {
	fr.Records = append(fr.Records, finalizationItem{
		Block:                block,
		FinalizedBlockHeight: finalizedBlockHeight,
	})
}

func (fr *finalizationRecord) marshalBinary() ([]byte, error)    { return cbor.Marshal(fr) }
func (fr *finalizationRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, fr) }

type finalizations struct {
	hs *historyStorage
}

func newFinalizations(hs *historyStorage) *finalizations {
	return &finalizations{hs: hs}
}

func (f *finalizations) store(block proto.Block, finalizedBlockHeight proto.Height,
	currentBlockID proto.BlockID) error {
	key := []byte(finalizationKey)
	data, err := f.hs.newestTopEntryData(key)
	if err != nil && !isNotFoundInHistoryOrDBErr(err) {
		return fmt.Errorf("failed to retrieve finalization record: %w", err)
	}
	var rec finalizationRecord
	if len(data) != 0 {
		if umErr := rec.unmarshalBinary(data); umErr != nil {
			return fmt.Errorf("failed to unmarshal finalization record: %w", umErr)
		}
	}
	rec.append(block, finalizedBlockHeight)
	newData, mErr := rec.marshalBinary()
	if mErr != nil {
		return fmt.Errorf("failed to marshal finalization record: %w", mErr)
	}
	if addErr := f.hs.addNewEntry(finalization, key, newData, currentBlockID); addErr != nil {
		return fmt.Errorf("failed to add finalization record: %w", addErr)
	}
	return nil
}

// newest returns the last finalized block (if exists).
func (f *finalizations) newest() (*finalizationItem, error) {
	key := []byte(finalizationKey)
	data, err := f.hs.newestTopEntryData(key)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, ErrNoFinalizationHistory
		}
		return nil, fmt.Errorf("failed to retrieve finalization record: %w", err)
	}
	var rec finalizationRecord
	if unmrshhlErr := rec.unmarshalBinary(data); unmrshhlErr != nil {
		return nil, fmt.Errorf("failed to unmarshal finalization record: %w", unmrshhlErr)
	}
	if len(rec.Records) == 0 {
		return nil, ErrNoFinalization
	}
	return &rec.Records[len(rec.Records)-1], nil
}

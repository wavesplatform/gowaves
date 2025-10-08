package state

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// commitmentItem represents a single commitment made by a block generator.
// It links the generator's Waves public key to its corresponding BLS endorser public key.
type commitmentItem struct {
	GeneratorPK crypto.PublicKey `cbor:"0,keyasint,omitempty"`
	EndorserPK  bls.PublicKey    `cbor:"1,keyasint,omitempty"`
}

// commitmentsRecord holds all generator commitments for a specific generation period.
type commitmentsRecord struct {
	Commitments []commitmentItem `cbor:"0,keyasint,omitempty"`
}

func (cr *commitmentsRecord) append(generatorPK crypto.PublicKey, endorserPK bls.PublicKey) {
	cr.Commitments = append(cr.Commitments, commitmentItem{
		GeneratorPK: generatorPK,
		EndorserPK:  endorserPK,
	})
}
func (cr *commitmentsRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(cr) }

func (cr *commitmentsRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, cr) }

// commitments manages the storage and retrieval of generator commitments.
type commitments struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage

	scheme          proto.Scheme
	calculateHashes bool
	hasher          *stateHasher
}

func newCommitments(hs *historyStorage, scheme proto.Scheme, calcHashes bool) *commitments {
	return &commitments{
		db:              hs.db,
		dbBatch:         hs.dbBatch,
		hs:              hs,
		scheme:          scheme,
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

func (c *commitments) store(
	periodStart uint32, generatorPK crypto.PublicKey, endorserPK bls.PublicKey, blockID proto.BlockID,
) error {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil && !errors.Is(err, keyvalue.ErrNotFound) {
		return fmt.Errorf("failed to retrieve commitments record: %w", err)
	}
	var rec commitmentsRecord
	if data != nil {
		if umErr := rec.unmarshalBinary(data); umErr != nil {
			return fmt.Errorf("failed to unmarshal commitments record: %w", umErr)
		}
	}
	rec.append(generatorPK, endorserPK)
	newData, mErr := rec.marshalBinary()
	if mErr != nil {
		return fmt.Errorf("failed to marshal commitments record: %w", mErr)
	}
	if addErr := c.hs.addNewEntry(commitment, key.bytes(), newData, blockID); addErr != nil {
		return fmt.Errorf("failed to add commitment record: %w", addErr)
	}
	return nil
}

// exists checks if a commitment exists for the given period start and generator public key.
func (c *commitments) exists(periodStart uint32, generatorPK crypto.PublicKey) (bool, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return false, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	pkb := generatorPK.Bytes()
	for _, cm := range rec.Commitments {
		if bytes.Equal(pkb, cm.GeneratorPK.Bytes()) {
			return true, nil
		}
	}
	return false, nil
}

// size returns the number of commitments for the given period start.
func (c *commitments) size(periodStart uint32) (int, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return 0, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	return len(rec.Commitments), nil
}

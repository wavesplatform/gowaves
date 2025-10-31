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
	hs *historyStorage
}

func newCommitments(hs *historyStorage) *commitments {
	return &commitments{
		hs: hs,
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
	if len(data) != 0 {
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
	data, err := c.hs.topEntryData(key.bytes())
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

func (c *commitments) newestExists(periodStart uint32, generatorPK crypto.PublicKey) (bool, error) {
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

func (c *commitments) generators(periodStart uint32) ([]crypto.PublicKey, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.topEntryData(key.bytes())
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return nil, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	generators := make([]crypto.PublicKey, len(rec.Commitments))
	for i, cm := range rec.Commitments {
		generators[i] = cm.GeneratorPK
	}
	return generators, nil
}

// newestGenerators returns public keys of generators commited to the given period.
func (c *commitments) newestGenerators(periodStart uint32) ([]crypto.PublicKey, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return nil, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	generators := make([]crypto.PublicKey, len(rec.Commitments))
	for i, cm := range rec.Commitments {
		generators[i] = cm.GeneratorPK
	}
	return generators, nil
}

// size returns the number of commitments for the given period start.
func (c *commitments) size(periodStart uint32) (int, error) {
	generators, err := c.generators(periodStart)
	if err != nil {
		return 0, err
	}
	return len(generators), nil
}

func (c *commitments) newestSize(periodStart uint32) (int, error) {
	generators, err := c.newestGenerators(periodStart)
	if err != nil {
		return 0, err
	}
	return len(generators), nil
}

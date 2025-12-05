package state

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"

	"github.com/fxamacker/cbor/v2"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
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

type commitmentsRecordForStateHashes struct {
	publicKey    crypto.PublicKey
	blsPublicKey bls.PublicKey
}

func (r *commitmentsRecordForStateHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(r.publicKey.Bytes()); err != nil {
		return err
	}
	if _, err := w.Write(r.blsPublicKey.Bytes()); err != nil {
		return err
	}
	return nil
}

func (r *commitmentsRecordForStateHashes) less(other stateComponent) bool {
	o, ok := other.(*commitmentsRecordForStateHashes)
	if !ok {
		panic("commitmentsRecordForStateHashes: invalid type assertion")
	}
	val := bytes.Compare(r.publicKey.Bytes(), o.publicKey.Bytes())
	if val > 0 {
		return false
	} else if val == 0 {
		return bytes.Compare(r.blsPublicKey.Bytes(), o.blsPublicKey.Bytes()) == -1
	}
	return true
}

// commitments manages the storage and retrieval of generator commitments.
type commitments struct {
	hs              *historyStorage
	calculateHashes bool
	hasher          *stateHasher
}

func newCommitments(hs *historyStorage, calcHashes bool) *commitments {
	return &commitments{
		hs:              hs,
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

func (c *commitments) store(
	periodStart uint32, generatorPK crypto.PublicKey, endorserPK bls.PublicKey, blockID proto.BlockID,
) error {
	key := commitmentKey{periodStart: periodStart}
	keyBytes := key.bytes()
	data, err := c.hs.newestTopEntryData(keyBytes)
	if err != nil && !isNotFoundInHistoryOrDBErr(err) {
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
	if c.calculateHashes {
		r := &commitmentsRecordForStateHashes{
			publicKey:    generatorPK,
			blsPublicKey: endorserPK,
		}
		if pErr := c.hasher.push(string(keyBytes), r, blockID); pErr != nil {
			return fmt.Errorf("failed to hash commitment record: %w", pErr)
		}
	}
	if addErr := c.hs.addNewEntry(commitment, keyBytes, newData, blockID); addErr != nil {
		return fmt.Errorf("failed to add commitment record: %w", addErr)
	}
	return nil
}

// exists checks if a commitment exists for the given period start and generator public key.
func (c *commitments) exists(
	periodStart uint32, generatorPK crypto.PublicKey, endorserPK bls.PublicKey,
) (bool, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.topEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	return checkCommitments(data, generatorPK, endorserPK)
}

// newestExists checks if a commitment exists for the given period start and generator public key.
// The function also checks that the endorser PK is not already used by another generator.
func (c *commitments) newestExists(
	periodStart uint32, generatorPK crypto.PublicKey, endorserPK bls.PublicKey,
) (bool, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to retrieve commitment record: %w", err)
	}
	return checkCommitments(data, generatorPK, endorserPK)
}

func (c *commitments) generators(periodStart uint32) ([]crypto.PublicKey, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.topEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
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
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve newest commitment record: %w", err)
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

func checkCommitments(data []byte, generatorPK crypto.PublicKey, endorserPK bls.PublicKey) (bool, error) {
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return false, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	pkb := generatorPK.Bytes()
	ekb := endorserPK.Bytes()
	for _, cm := range rec.Commitments {
		if bytes.Equal(pkb, cm.GeneratorPK.Bytes()) {
			return true, nil
		}
		if bytes.Equal(ekb, cm.EndorserPK.Bytes()) {
			return false, fmt.Errorf("endorser public key is already used by another generator")
		}
	}
	return false, nil
}

// size returns the number of commitments for the given period start.
func (c *commitments) size(periodStart uint32) (int, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.topEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
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

func (c *commitments) newestSize(periodStart uint32) (int, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to retrieve commitment newest record: %w", err)
	}
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return 0, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	return len(rec.Commitments), nil
}

// FindEndorserPKByIndex returns BLS endorser public keys using
// commitment indexes stored in FinalizationVoting.EndorserIndexes.
func (c *commitments) FindEndorserPKByIndex(
	periodStart uint32, index int,
) (bls.PublicKey, error) {
	var empty bls.PublicKey
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return empty, fmt.Errorf("no commitments found for period %d", periodStart)
		}
		return empty, fmt.Errorf("failed to retrieve commitments record: %w", err)
	}

	var rec commitmentsRecord
	if unmarshalErr := rec.unmarshalBinary(data); unmarshalErr != nil {
		return empty, fmt.Errorf("failed to unmarshal commitments: %w", unmarshalErr)
	}

	if index < 0 || index >= len(rec.Commitments) {
		return empty, fmt.Errorf("index %d out of range (size %d)", index, len(rec.Commitments))
	}

	return rec.Commitments[index].EndorserPK, nil
}

func (c *commitments) FindGeneratorPKByEndorserPK(periodStart uint32,
	endorserPK bls.PublicKey) (crypto.PublicKey, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return crypto.PublicKey{}, errors.Errorf("no commitments found for period %d, %v", periodStart, err)
		}
		return crypto.PublicKey{}, errors.Errorf("failed to retrieve commitments record: %v", err)
	}

	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return crypto.PublicKey{}, fmt.Errorf("failed to unmarshal commitments record: %w", umErr)
	}

	endPKb := endorserPK[:]
	for _, cm := range rec.Commitments {
		if bytes.Equal(endPKb, cm.EndorserPK[:]) {
			return cm.GeneratorPK, nil
		}
	}
	return crypto.PublicKey{}, fmt.Errorf("endorser public key not found in commitments for period %d", periodStart)
}

func (c *commitments) CommittedGenerators(periodStart uint32, scheme proto.Scheme) ([]proto.WavesAddress, error) {
	pks, err := c.newestGenerators(periodStart)
	if err != nil {
		return nil, err
	}
	addresses := make([]proto.WavesAddress, len(pks))
	for i, pk := range pks {
		addr, cnvrtErr := proto.NewAddressFromPublicKey(scheme, pk)
		if cnvrtErr != nil {
			return nil, cnvrtErr
		}
		addresses[i] = addr
	}
	return addresses, nil
}

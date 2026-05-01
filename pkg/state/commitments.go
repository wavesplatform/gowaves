package state

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ccoveille/go-safecast/v2"
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

func (cr *commitmentsRecord) lastIndex() (uint32, error) {
	if len(cr.Commitments) == 0 {
		return 0, fmt.Errorf("commitments record is empty")
	}
	return safecast.Convert[uint32](len(cr.Commitments) - 1)
}

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
	}
	if val == 0 {
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
		idx, liErr := rec.lastIndex()
		if liErr != nil {
			return fmt.Errorf("failed to get last index of commitments: %w", liErr)
		}
		shk := commitmentStateHashKey{
			periodStart: periodStart,
			index:       idx,
		}
		r := &commitmentsRecordForStateHashes{
			publicKey:    generatorPK,
			blsPublicKey: endorserPK,
		}
		if pErr := c.hasher.push(shk.string(), r, blockID); pErr != nil {
			return fmt.Errorf("failed to hash commitment record: %w", pErr)
		}
	}
	if addErr := c.hs.addNewEntry(commitment, keyBytes, newData, blockID); addErr != nil {
		return fmt.Errorf("failed to add commitment record: %w", addErr)
	}
	return nil
}

func (c *commitments) newestCommitments(periodStart uint32) ([]commitmentItem, error) {
	key := commitmentKey{periodStart: periodStart}
	data, err := c.hs.newestTopEntryData(key.bytes())
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
	return rec.Commitments, nil
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

// newestGenerators returns public keys of generators commited to the given period.
// Function is used to reset deposits on commited generators accounts.
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
	gs := make([]crypto.PublicKey, len(rec.Commitments))
	for i, cm := range rec.Commitments {
		gs[i] = cm.GeneratorPK
	}
	return gs, nil
}

// checkCommitments verifies that the generator public key is not already committed for the given period.
// Commitment considered existing independent of commited BLS PK value.
// Additionally, function checks that the BLS PK was not used by another commited generator.
// If no Waves PK was used in existing commitments all BLS keys are checked for uniqueness.
// Updating of BLS PK is prohibited.
func checkCommitments(data []byte, generatorPK crypto.PublicKey, endorserPK bls.PublicKey) (bool, error) {
	var rec commitmentsRecord
	if umErr := rec.unmarshalBinary(data); umErr != nil {
		return false, fmt.Errorf("failed to unmarshal commitment record: %w", umErr)
	}
	pkb := generatorPK.Bytes()
	ekb := endorserPK.Bytes()
	for _, cm := range rec.Commitments {
		if bytes.Equal(pkb, cm.GeneratorPK.Bytes()) {
			return true, nil // Commitment exist, no matter the BLS PK value, no second commitment is possible.
		}
		if bytes.Equal(ekb, cm.EndorserPK.Bytes()) {
			return false, fmt.Errorf("endorser public key is already used by another generator")
		}
	}
	return false, nil
}

func (c *commitments) prepareHashes() error {
	if !c.calculateHashes {
		return nil // No-op if hash calculation is disabled.
	}
	return c.hasher.stop()
}

func (c *commitments) reset() {
	if !c.calculateHashes {
		return // No-op if hash calculation is disabled.
	}
	c.hasher.reset()
}

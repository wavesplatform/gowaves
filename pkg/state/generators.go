package state

import (
	"encoding/binary"
	"fmt"
	"io"
	"slices"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type generationBalanceProvider interface {
	newestGeneratingBalance(proto.AddressID, proto.Height) (uint64, error)
}

type GeneratorInfo struct {
	address proto.WavesAddress
	pk      crypto.PublicKey
	blsPK   bls.PublicKey
	balance uint64
}

// bannedGeneratorsRecord is a structure used for CBOR serialization of banned generator indexes.
// It has public fields to allow encoding/decoding with the specified CBOR tags.
type bannedGeneratorsRecord struct {
	Indexes []uint32 `cbor:"0,keyasint,omitempty"`
}

func (r *bannedGeneratorsRecord) appendIndex(index uint32) error {
	for _, existingIndex := range r.Indexes {
		if existingIndex == index {
			return errors.Errorf("index %d is already present in the record", index)
		}
	}
	r.Indexes = append(r.Indexes, index)
	return nil
}

func (r *bannedGeneratorsRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(r) }

func (r *bannedGeneratorsRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, r) }

// bannedGeneratorsKey is a structure used to generate a unique key for storing banned generators in the history storage.
type bannedGeneratorsKey struct {
	periodStart uint32
}

func (k *bannedGeneratorsKey) bytes() []byte {
	buf := make([]byte, 1+uint32Size)
	buf[0] = bannedGeneratorsKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.periodStart)
	return buf
}

type generatorsBalancesRecordForStateHashes struct {
	balances []uint64
}

func newGeneratorsBalancesRecordForStateHashes(size int) *generatorsBalancesRecordForStateHashes {
	return &generatorsBalancesRecordForStateHashes{
		balances: make([]uint64, 0, size),
	}
}

func (r *generatorsBalancesRecordForStateHashes) append(balance uint64) {
	r.balances = append(r.balances, balance)
}

func (r *generatorsBalancesRecordForStateHashes) writeTo(w io.Writer) error {
	for _, balance := range r.balances {
		balanceBytes := make([]byte, uint64Size)
		binary.BigEndian.PutUint64(balanceBytes, balance)
		if _, err := w.Write(balanceBytes); err != nil {
			return fmt.Errorf("failed to write balance to state hash writer: %w", err)
		}
	}
	return nil
}

func (r *generatorsBalancesRecordForStateHashes) less(other stateComponent) bool {
	otherRecord := other.(*generatorsBalancesRecordForStateHashes)
	for i := 0; i < len(r.balances) && i < len(otherRecord.balances); i++ {
		if r.balances[i] < otherRecord.balances[i] {
			return true
		} else if r.balances[i] > otherRecord.balances[i] {
			return false
		}
	}
	return len(r.balances) < len(otherRecord.balances)
}

// generators manages the set of active block generators for the current block.
type generators struct {
	hs              *historyStorage
	settings        *settings.BlockchainSettings
	fs              featuresState
	commitments     *commitments
	balances        generationBalanceProvider
	set             []GeneratorInfo
	calculateHashes bool
	hasher          *stateHasher
}

func newGenerators(
	hs *historyStorage,
	fs featuresState,
	balances generationBalanceProvider,
	commitments *commitments,
	sets *settings.BlockchainSettings,
	calcHashes bool,
) *generators {
	return &generators{
		hs:              hs,
		settings:        sets,
		fs:              fs,
		commitments:     commitments,
		balances:        balances,
		set:             make([]GeneratorInfo, 0),
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

// initialize populates the generator set based on the provided commitments and balances.
// This method should be called upon block header processing.
func (g *generators) initialize(height proto.Height, blockID proto.BlockID) error {
	activationHeight, err := g.fs.activationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		return fmt.Errorf("failed to get activation height for Deterministic Finality feature: %w", err)
	}
	ps, err := CurrentGenerationPeriodStart(activationHeight, height, g.settings.GenerationPeriod)
	if err != nil {
		return fmt.Errorf("failed to calculate current generation period start: %w", err)
	}
	cms, err := g.commitments.newestCommitments(ps)
	if err != nil {
		return fmt.Errorf("failed to initialize generators set: %w", err)
	}
	g.set = slices.Grow(g.set, len(cms))
	generatorsBalancesRecord := newGeneratorsBalancesRecordForStateHashes(len(cms))
	for i, cm := range cms {
		a, aErr := proto.NewAddressFromPublicKey(g.settings.AddressSchemeCharacter, cm.GeneratorPK)
		if aErr != nil {
			return fmt.Errorf("failed to derive address from generator public key at index %d: %w", i, err)
		}
		b, bErr := g.balances.newestGeneratingBalance(a.ID(), height)
		if bErr != nil {
			return fmt.Errorf("failed to get balance for generator at index %d by address '%s': %w",
				i, a.String(), bErr)
		}
		gi := GeneratorInfo{
			address: a,
			pk:      cm.GeneratorPK,
			blsPK:   cm.EndorserPK,
			balance: b,
		}
		g.set = append(g.set, gi)
		if g.calculateHashes {
			generatorsBalancesRecord.append(b)
		}
	}
	if g.calculateHashes {
		key := bannedGeneratorsKey{periodStart: ps}
		if pErr := g.hasher.push(string(key.bytes()), generatorsBalancesRecord, blockID); pErr != nil {
			return fmt.Errorf("failed to hash generators balances record: %w", pErr)
		}
	}
	return nil
}

func (g *generators) generators() []GeneratorInfo {
	return g.set
}

func (g *generators) banGenerator(periodStart, index uint32, blockID proto.BlockID) error {
	if index >= uint32(len(g.set)) {
		return fmt.Errorf("generator index %d is out of bounds for the generator set of size %d",
			index, len(g.set))
	}
	key := bannedGeneratorsKey{periodStart: periodStart}
	keyBytes := key.bytes()
	recordBytes, err := g.hs.newestTopEntryData(keyBytes)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No record found, create new one.
			r := bannedGeneratorsRecord{
				Indexes: []uint32{index},
			}
			data, mErr := r.marshalBinary()
			if mErr != nil {
				return fmt.Errorf("failed to marshal record to binary data: %w", mErr)
			}
			return g.hs.addNewEntry(bannedGenerators, keyBytes, data, blockID)
		}
		return err
	}
	var r bannedGeneratorsRecord
	if uErr := r.unmarshalBinary(recordBytes); uErr != nil {
		return fmt.Errorf("failed to unmarshal record from binary data: %w", uErr)
	}
	if aErr := r.appendIndex(index); aErr != nil {
		return fmt.Errorf("failed to append index to record: %w", aErr)
	}
	recordBytes, err = r.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal record to binary data: %w", err)
	}
	return g.hs.addNewEntry(bannedGenerators, keyBytes, recordBytes, blockID)
}

func (g *generators) prepareHashes() error {
	if !g.calculateHashes {
		return nil // No-op if hash calculation is disabled.
	}
	return g.hasher.stop()
}

func (g *generators) reset() {
	if !g.calculateHashes {
		return // No-op if hash calculation is disabled.
	}
	g.hasher.reset()
}

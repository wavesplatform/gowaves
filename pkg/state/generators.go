package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"slices"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// generationBalanceProvider is an interface that abstracts the retrieval of generating balances for addresses at
// specific heights.
type generationBalanceProvider interface {
	newestGeneratingBalance(proto.AddressID, proto.Height) (uint64, error)
}

// commitmentsProvider is an interface that abstracts the retrieval of generator commitments for a given
// generation period.
type commitmentsProvider interface {
	newestCommitments(periodStart uint32) ([]commitmentItem, error)
}

type GeneratorInfo struct {
	index   uint32
	address proto.WavesAddress
	pk      crypto.PublicKey
	blsPK   bls.PublicKey
	balance uint64
	ban     bool
}

func (g *GeneratorInfo) Index() uint32 {
	return g.index
}

func (g *GeneratorInfo) GenerationBalance() uint64 {
	if g.ban {
		return 0
	}
	return g.balance
}

func (g *GeneratorInfo) BLSPublicKey() bls.PublicKey {
	return g.blsPK
}

func (g *GeneratorInfo) Address() proto.WavesAddress {
	return g.address
}

// bannedGeneratorsRecord is a structure used for CBOR serialization of banned generator indexes.
// It has public fields to allow encoding/decoding with the specified CBOR tags.
type bannedGeneratorsRecord struct {
	Indexes []uint32 `cbor:"0,keyasint,omitempty"`
}

func (r *bannedGeneratorsRecord) appendIndex(index uint32) error {
	if slices.Contains(r.Indexes, index) {
		return errors.Errorf("index %d is already present in the record", index)
	}
	r.Indexes = append(r.Indexes, index)
	return nil
}

func (r *bannedGeneratorsRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(r) }

func (r *bannedGeneratorsRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, r) }

// bannedGeneratorsKey is a structure used to generate a unique key for storing banned generators in the
// history storage.
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
	otherRecord, ok := other.(*generatorsBalancesRecordForStateHashes)
	if !ok {
		panic("generatorsBalancesRecordForStateHashes: invalid type assertion")
	}
	for i := 0; i < len(r.balances) && i < len(otherRecord.balances); i++ {
		if r.balances[i] < otherRecord.balances[i] {
			return true
		} else if r.balances[i] > otherRecord.balances[i] {
			return false
		}
	}
	return len(r.balances) < len(otherRecord.balances)
}

// ByAddress lookup functions allows to find generator by Waves AddressID.
func ByAddress(addr proto.AddressID) func(GeneratorInfo) bool {
	return func(info GeneratorInfo) bool {
		return info.address.ID() == addr
	}
}

// ByBLSPublicKey returns a lookup function that checks if a generator's BLS public key matches the provided one.
func ByBLSPublicKey(pk bls.PublicKey) func(GeneratorInfo) bool {
	return func(info GeneratorInfo) bool {
		return bytes.Equal(info.blsPK.Bytes(), pk.Bytes())
	}
}

// ByIndex returns a lookup function that checks if a generator's index matches the provided one.
func ByIndex(index uint32) func(GeneratorInfo) bool {
	return func(info GeneratorInfo) bool {
		return info.index == index
	}
}

// generators manages the set of generators for the current block.
// The generators storage entity itself lifetime can be larger than a single block, but the generator set is expected
// to be initialized and used within the context of a single block processing. The underlying history storage and
// legacy hasher are expected to be shared across multiple blocks.
type generators struct {
	hs *historyStorage

	fs          featuresState
	balances    generationBalanceProvider
	commitments commitmentsProvider
	settings    *settings.BlockchainSettings

	set                   []GeneratorInfo
	byAddress             map[proto.AddressID]GeneratorInfo
	activationHeight      proto.Height
	generationPeriodStart uint32
	blockID               proto.BlockID
	blockHeight           proto.Height
	blockTimestamp        uint64
	blockGenerator        *GeneratorInfo // Current block generator info.

	calculateHashes bool
	hasher          *stateHasher
}

func newGenerators(
	hs *historyStorage,
	fs featuresState,
	balances generationBalanceProvider,
	commitments commitmentsProvider,
	sets *settings.BlockchainSettings,
	calcHashes bool,
) *generators {
	return &generators{
		hs:              hs,
		fs:              fs,
		balances:        balances,
		commitments:     commitments,
		settings:        sets,
		set:             make([]GeneratorInfo, 0),
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

// initialize populates the generator set based on the provided commitments and balances.
// This method should be called upon block header processing.
// Parameters:
//
//	height - block application height,
//	blockID - ID of the applied block.
func (g *generators) initialize(
	height proto.Height, blockID proto.BlockID, generator crypto.PublicKey, ts uint64,
) error {
	var err error
	g.activationHeight, err = g.fs.newestActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) { // DeterministicFinality feature is not approved or activated.
			return nil
		}
		return fmt.Errorf("failed to get activation height for Deterministic Finality feature: %w", err)
	}
	g.generationPeriodStart, err = CurrentGenerationPeriodStart(g.activationHeight, height, g.settings.GenerationPeriod)
	if err != nil {
		return fmt.Errorf("failed to calculate current generation period start: %w", err)
	}
	cms, err := g.commitments.newestCommitments(g.generationPeriodStart)
	if err != nil {
		return fmt.Errorf("failed to initialize generators set: %w", err)
	}
	// Load saved bans of generators in the current generation period.
	bans, err := g.bans(g.generationPeriodStart)
	if err != nil {
		return fmt.Errorf("failed to retrieve banned generators for the current generation period: %w", err)
	}
	g.blockID = blockID
	g.blockHeight = height
	g.blockTimestamp = ts
	g.set = slices.Grow(g.set, len(cms))
	g.byAddress = make(map[proto.AddressID]GeneratorInfo)
	generatorsBalancesLSHRecord := newGeneratorsBalancesRecordForStateHashes(len(cms))
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
		idx := uint32(i)
		gi := GeneratorInfo{
			index:   idx,
			address: a,
			pk:      cm.GeneratorPK,
			blsPK:   cm.EndorserPK,
			balance: b,
			ban:     slices.Contains(bans, idx),
		}
		g.set = append(g.set, gi)
		if cm.GeneratorPK == generator {
			g.blockGenerator = &gi // Save reference to block generator info.
		}
		if g.calculateHashes {
			generatorsBalancesLSHRecord.append(b)
		}
	}
	if g.calculateHashes {
		key := bannedGeneratorsKey{periodStart: g.generationPeriodStart}
		if pErr := g.hasher.push(string(key.bytes()), generatorsBalancesLSHRecord, blockID); pErr != nil {
			return fmt.Errorf("failed to hash generators balances record: %w", pErr)
		}
	}
	return nil
}

func (g *generators) generator(index uint32) (GeneratorInfo, error) {
	if int(index) >= len(g.set) {

	}
}

func (g *generators) banGenerator(index uint32, blockID proto.BlockID) error {
	if int(index) >= len(g.set) {
		return fmt.Errorf("generator index %d is out of bounds for the generator set of size %d",
			index, len(g.set))
	}

	// Ban generator for the current block.
	g.set[index].ban = true

	// Save ban for processing of the future blocks.
	key := bannedGeneratorsKey{periodStart: g.generationPeriodStart}
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

func (g *generators) bans(periodStart uint32) ([]uint32, error) {
	key := bannedGeneratorsKey{periodStart: periodStart}
	keyBytes := key.bytes()
	recordBytes, err := g.hs.newestTopEntryData(keyBytes)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No record found, return empty bans list.
			return []uint32{}, nil
		}
		return nil, err
	}
	var r bannedGeneratorsRecord
	if uErr := r.unmarshalBinary(recordBytes); uErr != nil {
		return nil, fmt.Errorf("failed to unmarshal record from binary data: %w", uErr)
	}
	return r.Indexes, nil
}

// newestGeneratingBalance retrieves the generating balance for a given address and height. This method checks
// that given address is in the current generators set. If current generator set is empty, it uses the balance
// provider to get the balance for the address.
func (g *generators) newestGeneratingBalance(addr proto.AddressID, height proto.Height) (uint64, error) {
	if len(g.set) == 0 { // Generators set is empty just get balance from state.
		return g.balances.newestGeneratingBalance(addr, height)
	}
	a, err := addr.ToWavesAddress(g.settings.AddressSchemeCharacter)
	if err != nil {
		return 0, fmt.Errorf("failed to convert address ID to Waves address: %w", err)
	}
	if info, ok := g.byAddress[addr]; ok {
		if info.ban {
			return 0, fmt.Errorf("address '%s' is banned from generation", a.String())
		}
		return info.balance, nil
	}
	return 0, fmt.Errorf("address '%s' is not in the current generator set", a.String())
}

func (g *generators) findGenerator(lookup func(GeneratorInfo) bool) (GeneratorInfo, error) {
	for _, info := range g.set {
		if lookup(info) {
			return info, nil
		}
	}
	return GeneratorInfo{}, errors.New("generator is not found")
}

// TotalGenerationBalance returns generation balance of all commited generators.
// If no generators commited (generators set is empty) function returns 0 without an error.
// TODO: Make private.
func (g *generators) TotalGenerationBalance() (uint64, error) {
	if len(g.set) == 0 {
		return 0, nil
	}
	total := uint64(0)
	for _, gen := range g.set {
		if gen.ban {
			continue
		}
		if gen.GenerationBalance() < g.fs.minimalGeneratingBalanceAtHeight(g.blockHeight, g.blockTimestamp) {
			continue
		}
		total += gen.GenerationBalance()
	}
	return total, nil
}

func (g *generators) generatorsByHeight(height proto.Height) ([]GeneratorInfo, error) {
	return nil, errors.New("not implemented")
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

package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// generationBalanceProvider is an interface that abstracts the retrieval of generating balances for addresses at
// specific heights. Usually, for generating balance retrieval the blockchain height is used.
type generationBalanceProvider interface {
	newestGeneratingBalance(proto.AddressID, proto.Height) (uint64, error)
}

type generationBalanceManager interface {
	generationBalanceProvider
	burnDeposit(proto.AddressID, proto.BlockID) error
}

// commitmentsProvider is an interface that abstracts the retrieval of generator commitments for a given
// generation period.
type commitmentsProvider interface {
	newestCommitments(periodStart uint32) ([]commitmentItem, error)
}

type GeneratorInfo struct {
	index     uint32
	address   proto.WavesAddress
	pk        crypto.PublicKey
	blsPK     bls.PublicKey
	balance   uint64
	ban       bool
	threshold uint64
}

func (g *GeneratorInfo) Index() uint32 {
	return g.index
}

func (g *GeneratorInfo) GenerationBalance() uint64 {
	if g.ban {
		return 0
	}
	if g.balance < g.threshold { // If the balance of generator is less than minimal generation balance, return 0.
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

type generatorsKey struct {
	height uint64
}

func (k *generatorsKey) bytes() []byte {
	buf := make([]byte, 1+uint64Size)
	buf[0] = bannedGeneratorsKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type generator struct {
	Balance   uint64 `cbor:"0,keyasint,omitempty"`
	BanHeight uint32 `cbor:"1,keyasint,omitempty"`
}
type generatorsRecord struct {
	Generators          []generator `cbor:"0,keyasint,omitempty"`
	BlockGeneratorIndex uint32      `cbor:"1,keyasint,omitempty"`
	PeriodStart         uint32      `cbor:"2,keyasint,omitempty"`
}

// banGenerator updates existing generator setting the height at which it was banned.
func (r *generatorsRecord) banGenerator(index, height uint32) error {
	if len(g.Generators) <= int(index) {
		return fmt.Errorf("invalid generator index %d", index)
	}
	g := r.Generators[index]
	if g.BanHeight != 0 {
		return fmt.Errorf("generator with index %d is already banned at height %d", index, g.BanHeight)
	}
	r.Generators[index].BanHeight = height
	return nil
}

func (r *generatorsRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(r) }

func (r *generatorsRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, r) }

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
	balanceBytes := make([]byte, uint64Size)
	for _, balance := range r.balances {
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
	for i := 0; i < min(len(r.balances), len(otherRecord.balances)); i++ {
		if r.balances[i] < otherRecord.balances[i] {
			return true
		} else if r.balances[i] > otherRecord.balances[i] {
			return false
		}
	}
	return len(r.balances) < len(otherRecord.balances)
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
	balances    generationBalanceManager
	commitments commitmentsProvider
	settings    *settings.BlockchainSettings

	calculateHashes bool
	hasher          *stateHasher
}

func newGenerators(
	hs *historyStorage,
	fs featuresState,
	balances generationBalanceManager,
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
		calculateHashes: calcHashes,
		hasher:          newStateHasher(),
	}
}

// initialize populates the generator set based on the provided commitments and balances.
// This method should be called upon block header processing.
// Parameters:
//
//	blockchainHeight - height of the state (height of the last applied block),
//	blockID - ID of the applied block.
func (g *generators) initialize(
	blockchainHeight proto.Height, blockID proto.BlockID, genPK crypto.PublicKey, ts uint64,
) error {
	activationHeight, err := g.fs.newestActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // DeterministicFinality feature is not approved or activated.
			return nil
		}
		return fmt.Errorf("failed to get activation height for Deterministic Finality feature: %w", err)
	}
	blockHeight := blockchainHeight + 1
	periodStart, err := CurrentGenerationPeriodStart(activationHeight, blockHeight, g.settings.GenerationPeriod)
	if err != nil {
		return fmt.Errorf("failed to calculate current generation period start: %w", err)
	}
	cms, err := g.commitments.newestCommitments(periodStart)
	if err != nil {
		return fmt.Errorf("failed to initialize generators set: %w", err)
	}
	if len(cms) == 0 {
		return nil
	}
	// Load generators on the previous height to extract banned generators or copy generator set.
	pg, err := g.generators(blockchainHeight)
	if err != nil {
		return fmt.Errorf("failed to retrieve previous generators: %w", err)
	}
	if pg != nil && pg.PeriodStart == periodStart {
		// The generation period did not change, copy generator and update their balances.
		gs := generatorsRecord{
			Generators:          make([]generator, 0, len(pg.Generators)),
			BlockGeneratorIndex: 0,
			PeriodStart:         periodStart,
		}
		for i, gen := range pg.Generators {
			var bh uint32
			var bal uint64
			if gen.BanHeight != 0 {
				// The generator is banned, copy the ban height without querying balance.
				bh = gen.BanHeight
			} else {
				a, aErr := cms[i].address(g.settings.AddressSchemeCharacter)
				if aErr != nil {
					return aErr
				}
				// The initialization happens at the very beginning of the block, so the generation balance is
				// queried at the height of the last applied block (blockchainHeight).
				var bErr error
				bal, bErr = g.balances.newestGeneratingBalance(a.ID(), blockchainHeight)
				if bErr != nil {
					return fmt.Errorf("failed to get balance for generator at index %d by address '%s': %w",
						i, a.String(), bErr)
				}
			}
			gs.Generators = append(gs.Generators, generator{
				Balance:   bal,
				BanHeight: bh,
			})
		}
	}
	// Calculate minimal generation balance for the current height and timestamp.
	threshold := g.fs.minimalGeneratingBalanceAtHeight(blockHeight, ts)
	generatorsBalancesLSHRecord := newGeneratorsBalancesRecordForStateHashes(len(cms))
	for i, cm := range cms {
		a, aErr := proto.NewAddressFromPublicKey(g.settings.AddressSchemeCharacter, cm.GeneratorPK)
		if aErr != nil {
			return fmt.Errorf("failed to derive address from generator public key at index %d: %w", i, aErr)
		}
		// The initialization happens at the very beginning of the block, so the generation balance is queried at
		// the height of the last applied block (blockchainHeight).
		b, bErr := g.balances.newestGeneratingBalance(a.ID(), blockchainHeight)
		if bErr != nil {
			return fmt.Errorf("failed to get balance for generator at index %d by address '%s': %w",
				i, a.String(), bErr)
		}
		idx := uint32(i)
		banHeight, banned := bans[idx]
		if banned && banHeight == g.blockHeight-1 {
			// Generator was banned exactly on previous block, burn the deposit.
			if bdErr := g.balances.burnDeposit(a.ID(), blockID); bdErr != nil {
				return fmt.Errorf("failed to burn deposit of banned generator '%s' with index %d: %w",
					a.String(), i, bdErr)
			}
		}
		gi := GeneratorInfo{
			index:     idx,
			address:   a,
			pk:        cm.GeneratorPK,
			blsPK:     cm.EndorserPK,
			balance:   b,
			ban:       banned,
			threshold: threshold,
		}
		g.set = append(g.set, gi)
		if cm.GeneratorPK == genPK {
			g.blockGeneratorIndex = i // Save index of the current block generator.
		}
		if g.calculateHashes {
			generatorsBalancesLSHRecord.append(b)
		}
	}
	if len(cms) > 0 && g.blockGeneratorIndex == -1 {
		// The block generator was not initialized, which means it is not part of the committed generators.
		// This serves as an additional safety check, since the generator has already been validated by its
		// generation balance.
		return fmt.Errorf("block generator with public key '%s' is not in the committed generators set",
			genPK.String())
	}
	if g.calculateHashes {
		key := bannedGeneratorsKey{periodStart: g.generationPeriodStart}
		if pErr := g.hasher.push(string(key.bytes()), generatorsBalancesLSHRecord, blockID); pErr != nil {
			return fmt.Errorf("failed to hash generators balances record: %w", pErr)
		}
	}
	return nil
}

func (g *generators) generators(height proto.Height) (*generatorsRecord, error) {
	k := generatorsKey{height: height}
	data, err := g.hs.newestTopEntryData(k.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve generators record for height %d: %w", height, err)
	}
	r := new(generatorsRecord)
	if uErr := r.unmarshalBinary(data); uErr != nil {
		return nil, fmt.Errorf("failed to unmarshal generators record for height %d: %w", height, uErr)
	}
	return r, nil
}

func (g *generators) size() int {
	return len(g.set)
}

func (g *generators) string() string {
	sb := strings.Builder{}
	sb.WriteRune('[')
	for i, gen := range g.set {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(gen.Address().String())
	}
	sb.WriteRune(']')
	return sb.String()
}

func (g *generators) generator(index uint32) (GeneratorInfo, error) {
	if int(index) >= len(g.set) {
		return GeneratorInfo{}, fmt.Errorf("generator index %d is out of bounds for generator set of size %d",
			index, len(g.set))
	}
	return g.set[index], nil
}

func (g *generators) banGenerator(index uint32, height proto.Height, blockID proto.BlockID) error {
	if int(index) >= len(g.set) {
		return fmt.Errorf("generator index %d is out of bounds for the generator set of size %d",
			index, len(g.set))
	}
	if g.set[index].ban {
		return nil // Generator already banned, do nothing.
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
				Bans: map[uint32]uint64{index: height},
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
	if aErr := r.appendIndex(index, height); aErr != nil {
		return fmt.Errorf("failed to append index to record: %w", aErr)
	}
	recordBytes, err = r.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal record to binary data: %w", err)
	}
	return g.hs.addNewEntry(bannedGenerators, keyBytes, recordBytes, blockID)
}

func (g *generators) bans(periodStart uint32) (map[uint32]uint64, error) {
	key := bannedGeneratorsKey{periodStart: periodStart}
	keyBytes := key.bytes()
	recordBytes, err := g.hs.newestTopEntryData(keyBytes)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No record found, return empty bans list.
			return map[uint32]uint64{}, nil
		}
		return nil, err
	}
	var r bannedGeneratorsRecord
	if uErr := r.unmarshalBinary(recordBytes); uErr != nil {
		return nil, fmt.Errorf("failed to unmarshal record from binary data: %w", uErr)
	}
	return maps.Clone(r.Bans), nil
}

// newestGeneratingBalance retrieves the generating balance for a given address and height. This method checks
// that given address is in the current generators set. If current generator set is empty, it uses the balance
// provider to get the balance for the address.
func (g *generators) newestGeneratingBalance(addr proto.AddressID, height proto.Height) (uint64, error) {
	if len(g.set) == 0 { // Generators set is empty just get balance from state.
		return g.balances.newestGeneratingBalance(addr, height)
	}
	for _, gen := range g.set {
		if gen.Address().ID() == addr {
			if gen.ban {
				return 0, fmt.Errorf("address '%s' is banned from generation", gen.Address().String())
			}
			return gen.balance, nil
		}
	}
	a, err := addr.ToWavesAddress(g.settings.AddressSchemeCharacter)
	if err != nil {
		return 0, fmt.Errorf("failed to convert address ID to Waves address: %w", err)
	}
	return 0, fmt.Errorf("address '%s' is not in the current generator set %s", a.String(), g.string())
}

func (g *generators) findGenerator(lookup func(GeneratorInfo) bool) (GeneratorInfo, error) {
	for _, info := range g.set {
		if lookup(info) {
			return info, nil
		}
	}
	return GeneratorInfo{}, errors.New("generator is not found")
}

// totalGenerationBalance returns generation balance of all commited generators.
// If no generators commited (generators set is empty) function returns 0 without an error.
// Block generator included in the set.
func (g *generators) totalGenerationBalance() uint64 {
	if len(g.set) == 0 {
		return 0
	}
	total := uint64(0)
	for _, gen := range g.set {
		total += gen.GenerationBalance() // Banned generators or generators with insufficient balance return 0 here.
	}
	return total
}

func (g *generators) blockGenerator() (*GeneratorInfo, error) {
	if g.blockGeneratorIndex < 0 || g.blockGeneratorIndex >= len(g.set) {
		return nil, fmt.Errorf("invalid block generator index %d", g.blockGeneratorIndex)
	}
	r := g.set[g.blockGeneratorIndex]
	return &r, nil
}

func (g *generators) generatorsByHeight(height proto.Height) ([]GeneratorInfo, error) {
	if height != g.blockHeight {
		//TODO: Implement. The implementation is possible for extended API only, when the generators are stored for
		// every height.
		return nil, errors.New("not implemented")
	}
	return slices.Clone(g.set), nil
}

func (g *generators) prepareHashes() error {
	if !g.calculateHashes {
		return nil // No-op if hash calculation is disabled.
	}
	return g.hasher.stop()
}

func (g *generators) flush() {
	// TODO: Implement moving data to generators history if extended API is on.
}

func (g *generators) reset() {
	// TODO move wipe here
	if !g.calculateHashes {
		return // No-op if hash calculation is disabled.
	}
	g.hasher.reset()
}

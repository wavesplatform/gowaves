package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/fxamacker/cbor/v2"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

var ErrNoGeneratorsSet = errors.New("no generators set found")

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
	Index         uint32             `json:"-"`
	Address       proto.WavesAddress `json:"address"`
	PublicKey     crypto.PublicKey
	BLSPublicKey  bls.PublicKey `json:"-"`
	Balance       uint64        `json:"balance"`
	Ban           bool
	TransactionID crypto.Digest `json:"transactionID"`
}

func buildGeneratorInfo(
	index uint32, commitments []commitmentItem, generators []generator, scheme proto.Scheme,
) (GeneratorInfo, error) {
	if cs, gs := len(commitments), len(generators); cs != gs {
		return GeneratorInfo{}, fmt.Errorf("number of commitments %d does not match number of generators %d",
			cs, gs)
	}
	if s := len(commitments); int(index) >= s {
		return GeneratorInfo{}, fmt.Errorf("invalid generator index %d for commitments of size %d", index, s)
	}
	c := commitments[index]
	g := generators[index]
	a, aErr := g.AddressID.ToWavesAddress(scheme)
	if aErr != nil {
		return GeneratorInfo{}, fmt.Errorf("failed to convert address ID to Waves address for generator %d: %w",
			index, aErr)
	}
	gi := GeneratorInfo{
		Index:         index,
		Address:       a,
		PublicKey:     c.GeneratorPK,
		BLSPublicKey:  c.EndorserPK,
		Balance:       g.Balance,
		Ban:           g.BanHeight != 0,
		TransactionID: c.TransactionID,
	}
	return gi, nil
}

func buildInfos(commitments []commitmentItem, generators []generator, scheme proto.Scheme) ([]GeneratorInfo, error) {
	if cs, gs := len(commitments), len(generators); cs != gs {
		return nil, fmt.Errorf("number of commitments %d does not match number of generators %d", cs, gs)
	}
	res := make([]GeneratorInfo, len(commitments))
	for i, c := range commitments {
		idx, err := safecast.Convert[uint32](i)
		if err != nil {
			return nil, fmt.Errorf("failed to convert generator index %d: %w", i, err)
		}
		g := generators[i]
		a, err := g.AddressID.ToWavesAddress(scheme)
		if err != nil {
			return nil, fmt.Errorf("failed to convert address ID to Waves address for generator %d: %w", i, err)
		}
		gi := GeneratorInfo{
			Index:         idx,
			Address:       a,
			PublicKey:     c.GeneratorPK,
			BLSPublicKey:  c.EndorserPK,
			Balance:       g.Balance,
			Ban:           g.BanHeight != 0,
			TransactionID: c.TransactionID,
		}
		res[i] = gi
	}
	return res, nil
}

type generatorsKey struct {
	height uint64
}

func (k *generatorsKey) bytes() []byte {
	buf := make([]byte, 1+uint64Size)
	buf[0] = generatorsKeyPrefix
	binary.BigEndian.PutUint64(buf[1:], k.height)
	return buf
}

type generator struct {
	Balance   uint64          `cbor:"0,keyasint,omitempty"`
	BanHeight uint64          `cbor:"1,keyasint,omitempty"`
	AddressID proto.AddressID `cbor:"2,keyasint,omitempty"`
}

type generatorsRecord struct {
	Generators          []generator `cbor:"0,keyasint,omitempty"`
	BlockGeneratorIndex int32       `cbor:"1,keyasint,omitempty"`
	PeriodStart         uint32      `cbor:"2,keyasint,omitempty"`
}

// banGenerator updates existing generator setting the height at which it was banned.
func (r *generatorsRecord) banGenerator(index uint32, height proto.Height) error {
	if s := len(r.Generators); int(index) >= s {
		return fmt.Errorf("generator index %d is out of bounds for the generator set of size %d", index, s)
	}
	g := r.Generators[index]
	if g.BanHeight != 0 && g.BanHeight <= height {
		return fmt.Errorf("generator with index %d is already banned at height %d", index, g.BanHeight)
	}
	r.Generators[index].BanHeight = height
	return nil
}

func (r *generatorsRecord) marshalBinary() ([]byte, error) { return cbor.Marshal(r) }

func (r *generatorsRecord) unmarshalBinary(data []byte) error { return cbor.Unmarshal(data, r) }

func (r *generatorsRecord) Size() int { return len(r.Generators) }

func (r *generatorsRecord) string(scheme proto.Scheme) (string, error) {
	sb := strings.Builder{}
	sb.WriteRune('(')
	sb.WriteString(strconv.Itoa(int(r.BlockGeneratorIndex)))
	sb.WriteRune(')')
	sb.WriteRune('[')
	for i, gen := range r.Generators {
		if i > 0 {
			sb.WriteString(", ")
		}
		a, aErr := gen.AddressID.ToWavesAddress(scheme)
		if aErr != nil {
			return "", fmt.Errorf("failed to convert generator address ID to Waves address: %w", aErr)
		}
		sb.WriteString(a.String())
		sb.WriteRune('(')
		sb.WriteString(strconv.FormatUint(gen.Balance, 10))
		sb.WriteRune(')')
	}
	sb.WriteRune(']')
	sb.WriteRune('@')
	sb.WriteString(strconv.Itoa(int(r.PeriodStart)))
	return sb.String(), nil
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
func ByBLSPublicKey(pk bls.PublicKey) func(info GeneratorInfo) bool {
	return func(info GeneratorInfo) bool {
		return info.BLSPublicKey == pk
	}
}

// ByIndex returns a lookup function that checks if a generator's index matches the provided one.
func ByIndex(index uint32) func(GeneratorInfo) bool {
	return func(info GeneratorInfo) bool {
		return info.Index == index
	}
}

// generatorsStorage manages the set of generators for the current block.
// The generators storage entity itself lifetime can be larger than a single block, but the generator set is expected
// to be initialized and used within the context of a single block processing. The underlying history storage and
// legacy hasher are expected to be shared across multiple blocks.
type generatorsStorage struct {
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
) *generatorsStorage {
	return &generatorsStorage{
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
func (g *generatorsStorage) initialize(
	blockchainHeight proto.Height, blockID proto.BlockID, genPK crypto.PublicKey, ts uint64,
) error {
	activationHeight, err := g.fs.newestActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // DeterministicFinality feature is not approved or activated.
			return nil // No need to calculate generators state hashes here.
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
	threshold := g.fs.minimalGeneratingBalanceAtHeight(blockHeight, ts)
	// Load generators on the previous height to extract banned generators or copy generator set.
	pg, err := g.generators(blockchainHeight)
	if err != nil && !errors.Is(err, ErrNoGeneratorsSet) {
		return fmt.Errorf("failed to retrieve previous generators: %w", err)
	}
	noCommitments := len(cms) == 0
	noPrevGenerators := errors.Is(err, ErrNoGeneratorsSet)

	switch {
	case noCommitments && noPrevGenerators:
		// No commitments for the current period and no generators from the previous block, nothing to do.
		slog.Debug("No commitments, no previous generators found", slog.Uint64("height", blockHeight))
		return nil

	case noCommitments && pg.PeriodStart != periodStart:
		// No commitments for the current generation period, but if there is a previous generators set with
		// the different generation period, we have to punish conflicting endorsements on the last block of
		// previous generation period.
		slog.Debug("No commitments, but previous generators found", slog.Uint64("height", blockHeight))
		return g.punish(pg, blockchainHeight, blockID)

	case noCommitments && pg.PeriodStart == periodStart:
		// This situation is impossible, no commitments for the current period, but there is a previous generators
		// record with the same start of generation period.
		return fmt.Errorf("impossible state, generation record for height %d exist, "+
			"but commitments for the same period are empty", blockHeight)

	case noPrevGenerators:
		// There are commitments for the current generation period, but the previous generators list is empty,
		// this means that we are at the start of new generation period, after a generation period without commitments.
		// We should initalize a new generator set.
		slog.Debug("Initializing new generators set, no previous generators found", slog.Uint64("height", blockHeight))
		return g.initializeNewGeneratorsSetFromCommitments(cms, periodStart, threshold, genPK, blockHeight,
			blockchainHeight, blockID)

	case pg.PeriodStart == periodStart:
		// The generation period did not change, punish conflicting endorsements then copy the previous generators set
		// and update their balances.
		if pErr := g.punish(pg, blockchainHeight, blockID); pErr != nil {
			return fmt.Errorf("failed to punish conflicting endorsements of height %d: %w", blockchainHeight, pErr)
		}
		return g.copyGeneratorsSetAndUpdateBalances(pg, cms, threshold, genPK, blockHeight, blockchainHeight, blockID)

	default:
		// Here we are crossing the edge of generation period, new commitments and previous generator set with a
		// different generation period start. First of all, we should punish conflicting endorsements for the last
		// block of previous generation period, and then create a new generator set.
		if pErr := g.punish(pg, blockHeight, blockID); pErr != nil {
			return fmt.Errorf("failed to punish conflicting endorsements of height %d: %w", blockchainHeight, pErr)
		}
		return g.initializeNewGeneratorsSetFromCommitments(cms, periodStart, threshold, genPK, blockHeight,
			blockchainHeight, blockID)
	}
}

func (g *generatorsStorage) initializeNewGeneratorsSetFromCommitments(
	commitments []commitmentItem, start uint32, threshold uint64, generatorPK crypto.PublicKey,
	blockHeight, blockchainHeight proto.Height,
	blockID proto.BlockID) error {
	rec := generatorsRecord{
		Generators:          make([]generator, 0, len(commitments)),
		BlockGeneratorIndex: -1,
		PeriodStart:         start,
	}

	// Calculate minimal generation balance for the current height and timestamp.
	generatorsBalancesLSHRecord := newGeneratorsBalancesRecordForStateHashes(len(commitments))
	for i, cm := range commitments {
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
		if b < threshold {
			b = 0 // Set generation balances less than threshold to zero, so they are not eligible for generation.
		}
		if generatorPK == commitments[i].GeneratorPK {
			// The block generator found.
			idx, scErr := safecast.Convert[int32](i)
			if scErr != nil {
				return fmt.Errorf("failed to convert generator index: %w", scErr)
			}
			rec.BlockGeneratorIndex = idx
		}
		gen := generator{
			Balance:   b,
			BanHeight: 0,
			AddressID: a.ID(),
		}
		rec.Generators = append(rec.Generators, gen)
		if g.calculateHashes && b > 0 {
			generatorsBalancesLSHRecord.append(b)
		}
	}
	return g.finishGeneratorsInitialization(rec, generatorsBalancesLSHRecord, generatorPK, blockHeight,
		blockchainHeight, blockID, "New generators set initialized")
}

func (g *generatorsStorage) finishGeneratorsInitialization(
	rec generatorsRecord, shRec stateComponent,
	generatorPK crypto.PublicKey, blockHeight, blockchainHeight proto.Height, blockID proto.BlockID,
	logMessage string,
) error {
	if rec.BlockGeneratorIndex < 0 {
		// The block generator index was not initialized, which means it is not part of the committed generators.
		// This serves as an additional safety check, since the generator has already been validated by its
		// generation balance.
		return fmt.Errorf(
			"block generator with public key '%s' is not in the commitments for the current generation period",
			generatorPK.String())
	}
	if sErr := g.saveGeneratorsRecord(rec, blockHeight, blockID); sErr != nil {
		return fmt.Errorf("failed to save block generator record: %w", sErr)
	}
	str, err := rec.string(g.settings.AddressSchemeCharacter)
	if err != nil {
		return fmt.Errorf("failed to build generators set string for logging: %w", err)
	}
	slog.Debug(logMessage, slog.String("generators", str),
		slog.Uint64("blockHeight", blockHeight), slog.String("blockID", blockID.String()),
		slog.Uint64("blockchainHeight", blockchainHeight))
	return g.pushLegacyStateHashRecord(shRec, blockHeight, blockID)
}

func (g *generatorsStorage) copyGeneratorsSetAndUpdateBalances(
	pg *generatorsRecord, commitments []commitmentItem, threshold uint64, generatorPK crypto.PublicKey,
	blockHeight, blockchainHeight proto.Height,
	blockID proto.BlockID,
) error {
	rec := generatorsRecord{
		Generators:          make([]generator, 0, len(pg.Generators)),
		BlockGeneratorIndex: -1,
		PeriodStart:         pg.PeriodStart,
	}
	generatorsBalancesLSHRecord := newGeneratorsBalancesRecordForStateHashes(len(pg.Generators))
	for i, gen := range pg.Generators {
		ng := generator{
			Balance:   0,
			BanHeight: gen.BanHeight,
			AddressID: gen.AddressID,
		}
		if gen.BanHeight == 0 { // The generator is not banned, query the balance.
			// The initialization happens at the very beginning of the block, so the generation balance is
			// queried at the height of the last applied block (blockchainHeight).
			bal, bErr := g.balances.newestGeneratingBalance(gen.AddressID, blockchainHeight)
			if bErr != nil {
				return fmt.Errorf("failed to get balance for generator at index %d: %w", i, bErr)
			}
			if bal < threshold {
				bal = 0 // Set to zero, generators with insufficient balance excluded from generators set.
			}
			ng.Balance = bal
		}
		if generatorPK == commitments[i].GeneratorPK {
			// The block generator found.
			idx, scErr := safecast.Convert[int32](i)
			if scErr != nil {
				return fmt.Errorf("failed to convert generator index: %w", scErr)
			}
			rec.BlockGeneratorIndex = idx
		}
		rec.Generators = append(rec.Generators, ng)
		if g.calculateHashes && ng.Balance > 0 {
			generatorsBalancesLSHRecord.append(ng.Balance)
		}
	}
	return g.finishGeneratorsInitialization(rec, generatorsBalancesLSHRecord, generatorPK, blockHeight,
		blockchainHeight, blockID, "Generators set updated")
}

func (g *generatorsStorage) punish(pg *generatorsRecord, blockchainHeight proto.Height, blockID proto.BlockID) error {
	if pg == nil { // Sanity check.
		return nil
	}
	for i, cg := range pg.Generators {
		if cg.BanHeight == blockchainHeight { // Burn deposit for the generator banned on previous block.
			if bErr := g.balances.burnDeposit(cg.AddressID, blockID); bErr != nil {
				return fmt.Errorf("failed to burn deposit of previous generator with index %d: %w",
					i, bErr)
			}
			// TODO: Reset deposit here?
		}
	}
	return nil
}

func (g *generatorsStorage) saveGeneratorsRecord(
	record generatorsRecord, height proto.Height, blockID proto.BlockID,
) error {
	key := generatorsKey{height: height}
	data, err := record.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal generators record: %w", err)
	}
	return g.hs.addNewEntry(generators, key.bytes(), data, blockID)
}

func (g *generatorsStorage) pushLegacyStateHashRecord(
	record stateComponent, height proto.Height, blockID proto.BlockID,
) error {
	key := generatorsKey{height: height}
	if g.calculateHashes {
		if err := g.hasher.push(string(key.bytes()), record, blockID); err != nil {
			return fmt.Errorf("failed to hash generators balances record: %w", err)
		}
	}
	return nil
}

func (g *generatorsStorage) generators(height proto.Height) (*generatorsRecord, error) {
	k := generatorsKey{height: height}
	data, err := g.hs.newestTopEntryData(k.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return nil, ErrNoGeneratorsSet
		}
		return nil, fmt.Errorf("failed to retrieve generators record for height %d: %w", height, err)
	}
	r := new(generatorsRecord)
	if uErr := r.unmarshalBinary(data); uErr != nil {
		return nil, fmt.Errorf("failed to unmarshal generators record for height %d: %w", height, uErr)
	}
	return r, nil
}

func (g *generatorsStorage) infos(height proto.Height) ([]GeneratorInfo, error) {
	gs, err := g.generators(height)
	if err != nil {
		return nil, err
	}
	cs, err := g.getCommitments(height)
	if err != nil {
		return nil, err
	}
	return buildInfos(cs, gs.Generators, g.settings.AddressSchemeCharacter)
}

func (g *generatorsStorage) getCommitments(height proto.Height) ([]commitmentItem, error) {
	activationHeight, err := g.fs.newestActivationHeight(int16(settings.DeterministicFinality))
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // DeterministicFinality feature is not approved or activated.
			return nil, nil // Not an error, commitments are empty.
		}
		return nil, fmt.Errorf("failed to get activation height for Deterministic Finality feature: %w", err)
	}
	periodStart, err := CurrentGenerationPeriodStart(activationHeight, height, g.settings.GenerationPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current generation period start: %w", err)
	}
	return g.commitments.newestCommitments(periodStart)
}

func (g *generatorsStorage) string(height proto.Height) (string, error) {
	gs, err := g.generators(height)
	if err != nil {
		if errors.Is(err, ErrNoGeneratorsSet) {
			return "n/a", nil
		}
		return "", fmt.Errorf("failed to build generators set string for height %d: %w", height, err)
	}
	return gs.string(g.settings.AddressSchemeCharacter)
}

func (g *generatorsStorage) generator(index uint32, height proto.Height) (GeneratorInfo, error) {
	gs, err := g.generators(height)
	if err != nil {
		return GeneratorInfo{}, fmt.Errorf("failed to retrieve generator by index %d for height %d: %w",
			index, height, err)
	}
	if s := gs.Size(); int(index) >= s {
		return GeneratorInfo{}, fmt.Errorf("generator index %d is out of bounds for generator set of size %d",
			index, s)
	}
	cms, err := g.getCommitments(height)
	if err != nil {
		return GeneratorInfo{}, fmt.Errorf("failed to get commitments for height %d: %w", height, err)
	}
	return buildGeneratorInfo(index, cms, gs.Generators, g.settings.AddressSchemeCharacter)
}

func (g *generatorsStorage) banGenerator(index uint32, height proto.Height, blockID proto.BlockID) error {
	gs, err := g.generators(height)
	if err != nil {
		return fmt.Errorf("failed to retrieve generator by index %d for height %d: %w", index, height, err)
	}
	// Ban generator for the current block.
	if bErr := gs.banGenerator(index, height); bErr != nil {
		return fmt.Errorf("failed to ban: %w", bErr)
	}

	// Save ban for processing of the future blocks.
	key := generatorsKey{height: height}
	data, err := gs.marshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal generators record for height %d: %w", height, err)
	}
	return g.hs.addNewEntry(generators, key.bytes(), data, blockID)
}

// newestGeneratingBalance retrieves the generating balance for a given address and height. This method checks
// that given address is in the current generators set. If current generator set is empty, it uses the balance
// provider to get the balance for the address.
// Note that the height is the height of the state (blockchain height) here.
func (g *generatorsStorage) newestGeneratingBalance(
	addr proto.AddressID, blockchainHeight proto.Height,
) (uint64, error) {
	blockHeight := blockchainHeight + 1
	gs, err := g.generators(blockHeight)
	if err != nil {
		if errors.Is(err, ErrNoGeneratorsSet) {
			return g.balances.newestGeneratingBalance(addr, blockchainHeight)
		}
		return 0, fmt.Errorf("failed to retrieve generators for block at height %d: %w", blockHeight, err)
	}
	if gs.Size() == 0 { // Generators set is empty just get balance from state.
		return g.balances.newestGeneratingBalance(addr, blockchainHeight)
	}
	for _, gen := range gs.Generators {
		if gen.AddressID == addr {
			if gen.BanHeight != 0 { // Generator is banned, produce specific error for clarity.
				a, aErr := gen.AddressID.ToWavesAddress(g.settings.AddressSchemeCharacter)
				if aErr != nil {
					return 0, fmt.Errorf("failed to convert address ID to Waves address: %w", aErr)
				}
				return 0, fmt.Errorf("address '%s' is banned from generation", a.String())
			}
			return gen.Balance, nil
		}
	}
	// Checked whole generator set, but no generator with given address ID found.
	a, err := addr.ToWavesAddress(g.settings.AddressSchemeCharacter)
	if err != nil {
		return 0, fmt.Errorf("failed to convert address ID to Waves address: %w", err)
	}
	str, err := g.string(blockHeight)
	if err != nil {
		return 0, fmt.Errorf("failed to get newest generation balance for block at height %d: %w",
			blockHeight, err)
	}
	return 0, fmt.Errorf("address '%s' is not in the current generator set %s", a.String(), str)
}

func (g *generatorsStorage) findGenerator(height proto.Height, lookup func(GeneratorInfo) bool) (GeneratorInfo, error) {
	infos, err := g.infos(height)
	if err != nil {
		return GeneratorInfo{}, fmt.Errorf("failed to find generator: %w", err)
	}
	for _, info := range infos {
		if lookup(info) {
			return info, nil
		}
	}
	return GeneratorInfo{}, errors.New("generator is not found")
}

// totalGenerationBalance returns generation balance of all commited generators.
// If no generators commited (generators set is empty) function returns 0 without an error.
// Block generator included in the set.
func (g *generatorsStorage) totalGenerationBalance(height proto.Height) (uint64, error) {
	gs, err := g.generators(height)
	if err != nil {
		if errors.Is(err, ErrNoGeneratorsSet) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to calculate total generation balance for height %d: %w", height, err)
	}
	if gs.Size() == 0 {
		return 0, nil
	}
	total := uint64(0)
	for _, gen := range gs.Generators {
		total += gen.Balance // Banned generators or generators with insufficient balance return 0 here.
	}
	return total, nil
}

func (g *generatorsStorage) blockGenerator(height proto.Height) (uint32, *generator, error) {
	gs, err := g.generators(height)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to retrieve generators for height %d: %w", height, err)
	}
	if bgi := int(gs.BlockGeneratorIndex); bgi < 0 || bgi >= gs.Size() {
		return 0, nil, fmt.Errorf("invalid block generator index %d", bgi)
	}
	idx, err := safecast.Convert[uint32](gs.BlockGeneratorIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to convert block generator index: %w", err)
	}
	bg := gs.Generators[idx]
	return idx, &bg, nil
}

func (g *generatorsStorage) prepareHashes() error {
	if !g.calculateHashes {
		return nil // No-op if hash calculation is disabled.
	}
	return g.hasher.stop()
}

func (g *generatorsStorage) reset() {
	if !g.calculateHashes {
		return // No-op if hash calculation is disabled.
	}
	g.hasher.reset()
}

package state

import (
	"bytes"
	"context"
	stderrs "errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	rollbackMaxBlocks     = 2000
	blocksStorDir         = "blocks_storage"
	keyvalueDir           = "key_value"
	maxScriptsRunsInBlock = 101
)

var empty struct{}

func wrapErr(stateErrorType ErrorType, err error) error {
	var stateError StateError
	switch {
	case errors.As(err, &stateError):
		return err
	default:
		return NewStateError(stateErrorType, err)
	}
}

type blockchainEntitiesStorage struct {
	hs                *historyStorage
	aliases           *aliases
	assets            *assets
	leases            *leases
	scores            *scores
	blocksInfo        *blocksInfo
	balances          *balances
	features          featuresState
	monetaryPolicy    *monetaryPolicy
	ordersVolumes     *ordersVolumes
	accountsDataStor  *accountsDataStorage
	sponsoredAssets   *sponsoredAssets
	scriptsStorage    scriptStorageState
	scriptsComplexity *scriptsComplexity
	invokeResults     *invokeResults
	stateHashes       *stateHashes
	hitSources        *hitSources
	snapshots         *snapshotsAtHeight
	patches           *patchesStorage
	calculateHashes   bool
}

func newBlockchainEntitiesStorage(hs *historyStorage, sets *settings.BlockchainSettings, rw *blockReadWriter, calcHashes bool) (*blockchainEntitiesStorage, error) {
	assets := newAssets(hs.db, hs.dbBatch, hs)
	balances, err := newBalances(hs.db, hs, assets, sets, calcHashes)
	if err != nil {
		return nil, err
	}
	scriptsStorage, err := newScriptsStorage(hs, sets.AddressSchemeCharacter, calcHashes)
	if err != nil {
		return nil, err
	}
	features := newFeatures(rw, hs.db, hs, sets, settings.FeaturesInfo)
	return &blockchainEntitiesStorage{
		hs,
		newAliases(hs, sets.AddressSchemeCharacter, calcHashes),
		assets,
		newLeases(hs, calcHashes),
		newScores(hs),
		newBlocksInfo(hs),
		balances,
		features,
		newMonetaryPolicy(hs, sets),
		newOrdersVolumes(hs),
		newAccountsDataStorage(hs.db, hs.dbBatch, hs, calcHashes),
		newSponsoredAssets(rw, features, hs, sets, calcHashes),
		scriptsStorage,
		newScriptsComplexity(hs),
		newInvokeResults(hs),
		newStateHashes(hs),
		newHitSources(hs),
		newSnapshotsAtHeight(hs, sets.AddressSchemeCharacter),
		newPatchesStorage(hs, sets.AddressSchemeCharacter),
		calcHashes,
	}, nil
}

func (s *blockchainEntitiesStorage) putStateHash(prevHash []byte, height uint64, blockID proto.BlockID) (*proto.StateHash, error) {
	sh := &proto.StateHash{
		BlockID: blockID,
		FieldsHashes: proto.FieldsHashes{
			WavesBalanceHash:  s.balances.wavesHashAt(blockID),
			AssetBalanceHash:  s.balances.assetsHashAt(blockID),
			DataEntryHash:     s.accountsDataStor.hasher.stateHashAt(blockID),
			AccountScriptHash: s.scriptsStorage.getAccountScriptsHasher().stateHashAt(blockID),
			AssetScriptHash:   s.scriptsStorage.getAssetScriptsHasher().stateHashAt(blockID),
			LeaseBalanceHash:  s.balances.leaseHashAt(blockID),
			LeaseStatusHash:   s.leases.hasher.stateHashAt(blockID),
			SponsorshipHash:   s.sponsoredAssets.hasher.stateHashAt(blockID),
			AliasesHash:       s.aliases.hasher.stateHashAt(blockID),
		},
	}
	if err := sh.GenerateSumHash(prevHash); err != nil {
		return nil, err
	}
	if err := s.stateHashes.saveLegacyStateHash(sh, height); err != nil {
		return nil, err
	}
	return sh, nil
}

func (s *blockchainEntitiesStorage) prepareHashes() error {
	if err := s.accountsDataStor.prepareHashes(); err != nil {
		return err
	}
	if err := s.balances.prepareHashes(); err != nil {
		return err
	}
	if err := s.scriptsStorage.prepareHashes(); err != nil {
		return err
	}
	if err := s.leases.prepareHashes(); err != nil {
		return err
	}
	if err := s.sponsoredAssets.prepareHashes(); err != nil {
		return err
	}
	if err := s.aliases.prepareHashes(); err != nil {
		return err
	}
	return nil
}

func (s *blockchainEntitiesStorage) handleLegacyStateHashes(blockchainHeight uint64, blockIds []proto.BlockID) error {
	if !s.calculateHashes {
		return nil
	}
	if blockchainHeight < 1 {
		return errors.New("bad blockchain height, should be greater than 0")
	}
	// Calculate any remaining hashes.
	if err := s.prepareHashes(); err != nil {
		return err
	}
	prevHash, err := s.stateHashes.legacyStateHash(blockchainHeight)
	if err != nil {
		return err
	}
	startHeight := blockchainHeight + 1
	for i, id := range blockIds {
		height := startHeight + uint64(i)
		newPrevHash, err := s.putStateHash(prevHash.SumHash[:], height, id)
		if err != nil {
			return err
		}
		prevHash = newPrevHash
	}
	return nil
}

func (s *blockchainEntitiesStorage) commitUncertain(blockID proto.BlockID) error {
	if err := s.assets.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.accountsDataStor.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.scriptsStorage.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.leases.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.sponsoredAssets.commitUncertain(blockID); err != nil {
		return err
	}
	return nil
}

func (s *blockchainEntitiesStorage) dropUncertain() {
	s.assets.dropUncertain()
	s.accountsDataStor.dropUncertain()
	s.scriptsStorage.dropUncertain()
	s.leases.dropUncertain()
	s.sponsoredAssets.dropUncertain()
}

func (s *blockchainEntitiesStorage) reset() {
	s.hs.reset()
	s.assets.reset()
	s.accountsDataStor.reset()
	s.balances.reset()
	s.scriptsStorage.reset()
	s.leases.reset()
	s.sponsoredAssets.reset()
	s.aliases.reset()
}

func (s *blockchainEntitiesStorage) flush() error {
	s.aliases.flush()
	if err := s.hs.flush(); err != nil {
		return err
	}
	if err := s.accountsDataStor.flush(); err != nil {
		return err
	}
	return nil
}

func checkCompatibility(stateDB *stateDB, params StateParams) error {
	version, err := stateDB.stateVersion()
	if err != nil {
		return errors.Errorf("stateVersion: %v", err)
	}
	if version != StateVersion {
		return errors.Errorf("incompatible storage version %d; current state supports only %d", version, StateVersion)
	}
	hasDataForExtendedApi, err := stateDB.stateStoresApiData()
	if err != nil {
		return errors.Errorf("stateStoresApiData(): %v", err)
	}
	if params.StoreExtendedApiData != hasDataForExtendedApi {
		return errors.Errorf("extended API incompatibility: state stores: %v; want: %v", hasDataForExtendedApi, params.StoreExtendedApiData)
	}
	hasDataForHashes, err := stateDB.stateStoresHashes()
	if err != nil {
		return errors.Errorf("stateStoresHashes: %v", err)
	}
	if params.BuildStateHashes != hasDataForHashes {
		return errors.Errorf("state hashes incompatibility: state stores: %v; want: %v", hasDataForHashes, params.BuildStateHashes)
	}
	return nil
}

func handleAmendFlag(stateDB *stateDB, amend bool) (bool, error) {
	storedAmend, err := stateDB.amendFlag()
	if err != nil {
		return false, errors.Wrap(err, "failed to get stored amend flag")
	}
	if !storedAmend && amend { // update if storedAmend == false and amend == true
		if err := stateDB.updateAmendFlag(amend); err != nil {
			return false, errors.Wrap(err, "failed to update amend flag")
		}
		storedAmend = amend
	}
	return storedAmend, nil
}

type newBlocks struct {
	binary    bool
	binBlocks [][]byte
	blocks    []*proto.Block
	snapshots []*proto.BlockSnapshot
	curPos    int

	rw       *blockReadWriter
	settings *settings.BlockchainSettings
}

func newNewBlocks(rw *blockReadWriter, settings *settings.BlockchainSettings) *newBlocks {
	return &newBlocks{
		rw:       rw,
		settings: settings,
	}
}

func (n *newBlocks) len() int {
	if n.binary {
		if n.curPos > len(n.binBlocks) {
			return 0
		}
		return len(n.binBlocks) - n.curPos
	}
	if n.curPos > len(n.blocks) {
		return 0
	}
	return len(n.blocks) - n.curPos
}

func (n *newBlocks) setNewBinary(blocks [][]byte) {
	n.reset()
	n.binBlocks = blocks
	n.binary = true
}

func (n *newBlocks) setNew(blocks []*proto.Block) {
	n.reset()
	n.blocks = blocks
	n.binary = false
}

func (n *newBlocks) setNewWithSnapshots(blocks []*proto.Block, snapshots []*proto.BlockSnapshot) error {
	if len(blocks) != len(snapshots) {
		return errors.New("the numbers of snapshots doesn't match the number of blocks")
	}
	n.reset()
	n.blocks = blocks
	n.snapshots = snapshots
	n.binary = false
	return nil
}

func (n *newBlocks) setNewBinaryWithSnapshots(blocks [][]byte, snapshots []*proto.BlockSnapshot) error {
	if len(blocks) != len(snapshots) {
		return errors.New("the numbers of snapshots doesn't match the number of blocks")
	}
	n.reset()
	n.binBlocks = blocks
	n.snapshots = snapshots
	n.binary = true
	return nil
}

func (n *newBlocks) next() bool {
	n.curPos++
	if n.binary {
		return n.curPos <= len(n.binBlocks)
	} else {
		return n.curPos <= len(n.blocks)
	}
}

func (n *newBlocks) unmarshalBlock(block *proto.Block, blockBytes []byte) error {
	if n.rw.protobufActivated {
		if err := block.UnmarshalFromProtobuf(blockBytes); err != nil {
			return err
		}
	} else {
		if err := block.UnmarshalBinary(blockBytes, n.settings.AddressSchemeCharacter); err != nil {
			return err
		}
	}
	return nil
}

func getBlockDataWithOptionalSnapshot[T interface{ []byte | *proto.Block }](
	curPos int,
	blocks []T,
	snapshots []*proto.BlockSnapshot,
) (T, *proto.BlockSnapshot, error) {
	if curPos > len(blocks) || curPos < 1 {
		var zero T
		return zero, nil, errors.New("bad current position")
	}
	var (
		pos              = curPos - 1
		block            = blocks[pos]
		optionalSnapshot *proto.BlockSnapshot
	)
	if sl := len(snapshots); sl != 0 { // snapshots aren't empty
		if bl := len(blocks); sl != bl { // if snapshots are present, they must have the same length as blocks
			var zero T
			return zero, nil, errors.Errorf("snapshots and blocks slices have different lengths %d and %d", sl, bl)
		}
		optionalSnapshot = snapshots[pos] // blocks and snapshots have the same length
	}
	return block, optionalSnapshot, nil
}

func (n *newBlocks) current() (*proto.Block, *proto.BlockSnapshot, error) {
	if !n.binary {
		block, optionalSnapshot, err := getBlockDataWithOptionalSnapshot(n.curPos, n.blocks, n.snapshots)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get current deserialized block")
		}
		return block, optionalSnapshot, nil
	}
	blockBytes, optionalSnapshot, err := getBlockDataWithOptionalSnapshot(n.curPos, n.binBlocks, n.snapshots)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get current binary block")
	}
	block := &proto.Block{}
	if unmErr := n.unmarshalBlock(block, blockBytes); unmErr != nil {
		return nil, nil, unmErr
	}
	return block, optionalSnapshot, nil
}

func (n *newBlocks) reset() {
	n.binBlocks = nil
	n.blocks = nil
	n.snapshots = nil
	n.curPos = 0
}

type stateManager struct {
	mu *sync.RWMutex

	// Last added block.
	lastBlock atomic.Value

	genesis *proto.Block
	stateDB *stateDB

	stor *blockchainEntitiesStorage
	rw   *blockReadWriter

	// BlockchainSettings: general info about the blockchain type, constants etc.
	settings *settings.BlockchainSettings
	// Validator: validator for block headers.
	cv *consensus.Validator
	// Appender implements validation/diff management functionality.
	appender *txAppender
	atx      *addressTransactions

	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int

	newBlocks *newBlocks

	enableLightNode bool
}

func initDatabase(
	dataDir, blockStorageDir string,
	amend bool,
	params StateParams,
) (*keyvalue.KeyVal, keyvalue.Batch, *stateDB, bool, error) {
	dbDir := filepath.Join(dataDir, keyvalueDir)
	zap.S().Info("Initializing state database, will take up to few minutes...")
	params.DbParams.BloomFilterParams.Store.WithPath(filepath.Join(blockStorageDir, "bloom"))
	db, err := keyvalue.NewKeyVal(dbDir, params.DbParams)
	if err != nil {
		return nil, nil, nil, false, wrapErr(Other, errors.Wrap(err, "failed to create db"))
	}
	zap.S().Info("Finished initializing database")
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, nil, nil, false, wrapErr(Other, errors.Wrap(err, "failed to create db batch"))
	}
	sdb, err := newStateDB(db, dbBatch, params)
	if err != nil {
		return nil, nil, nil, false, wrapErr(Other, errors.Wrap(err, "failed to create stateDB"))
	}
	if cErr := checkCompatibility(sdb, params); cErr != nil {
		return nil, nil, nil, false, wrapErr(IncompatibilityError, cErr)
	}
	handledAmend, err := handleAmendFlag(sdb, amend)
	if err != nil {
		return nil, nil, nil, false, wrapErr(Other, errors.Wrap(err, "failed to handle amend flag"))
	}
	return db, dbBatch, sdb, handledAmend, nil
}

func initGenesis(state *stateManager, height uint64, settings *settings.BlockchainSettings) error {
	state.setGenesisBlock(&settings.Genesis)
	// 0 state height means that no blocks are found in state, so blockchain history is empty and we have to add genesis
	if height == 0 {
		// Assign unique block number for this block ID, add this number to the list of valid blocks
		if err := state.stateDB.addBlock(settings.Genesis.BlockID()); err != nil {
			return err
		}
		if err := state.addGenesisBlock(); err != nil {
			return errors.Wrap(err, "failed to apply/save genesis")
		}
		// We apply pre-activated features after genesis block, so they aren't active in genesis itself
		if err := state.applyPreActivatedFeatures(settings.PreactivatedFeatures, settings.Genesis.BlockID()); err != nil {
			return errors.Wrap(err, "failed to apply pre-activated features")
		}
	}

	// check the correct blockchain is being loaded
	genesis, err := state.BlockByHeight(1)
	if err != nil {
		return errors.Wrap(err, "failed to get genesis block from state")
	}

	if genErr := settings.Genesis.GenerateBlockID(settings.AddressSchemeCharacter); genErr != nil {
		return errors.Wrap(genErr, "failed to generate genesis block id from config")
	}
	if !bytes.Equal(genesis.ID.Bytes(), settings.Genesis.ID.Bytes()) {
		return errors.New("genesis blocks from state and config mismatch")
	}
	return nil
}

func newStateManager(
	dataDir string,
	amend bool,
	params StateParams,
	settings *settings.BlockchainSettings,
	enableLightNode bool,
) (*stateManager, error) {
	if err := validateSettings(settings); err != nil {
		return nil, err
	}
	if _, err := os.Stat(dataDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(dataDir, 0750); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create state directory: %v", err))
		}
	}
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(blockStorageDir, 0750); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create blocks directory: %v", err))
		}
	}
	// Initialize database.
	db, dbBatch, sdb, handledAmend, err := initDatabase(dataDir, blockStorageDir, amend, params)
	if err != nil {
		return nil, err
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(
		blockStorageDir,
		params.OffsetLen,
		params.HeaderOffsetLen,
		sdb,
		settings.AddressSchemeCharacter,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v", err))
	}
	sdb.setRw(rw)
	hs, err := newHistoryStorage(db, dbBatch, sdb, handledAmend)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create history storage: %v", err))
	}
	stor, err := newBlockchainEntitiesStorage(hs, settings, rw, params.BuildStateHashes)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create blockchain entities storage: %v", err))
	}
	atxParams := &addressTransactionsParams{
		dir:                 blockStorageDir,
		batchedStorMemLimit: AddressTransactionsMemLimit,
		batchedStorMaxKeys:  AddressTransactionsMaxKeys,
		maxFileSize:         MaxAddressTransactionsFileSize,
		providesData:        params.ProvideExtendedApi,
	}
	atx, err := newAddressTransactions(db, sdb, rw, atxParams, handledAmend)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create address transactions storage: %v", err))
	}
	state := &stateManager{
		mu:                        &sync.RWMutex{},
		stateDB:                   sdb,
		stor:                      stor,
		rw:                        rw,
		settings:                  settings,
		atx:                       atx,
		verificationGoroutinesNum: params.VerificationGoroutinesNum,
		newBlocks:                 newNewBlocks(rw, settings),
		enableLightNode:           enableLightNode,
	}
	// Set fields which depend on state.
	// Consensus validator is needed to check block headers.
	snapshotApplier := newBlockSnapshotsApplier(nil, newSnapshotApplierStorages(stor, rw))
	appender, err := newTxAppender(state, rw, stor, settings, sdb, atx, &snapshotApplier)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.appender = appender
	state.cv = consensus.NewValidator(state, settings, params.Time)

	height, err := state.Height()
	if err != nil {
		return nil, err
	}

	if gErr := initGenesis(state, height, settings); gErr != nil {
		return nil, gErr
	}
	if err := state.loadLastBlock(); err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	h, err := state.Height()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.checkProtobufActivation(h + 1)
	return state, nil
}

func (s *stateManager) NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script by account '%s'", account.String())
	}
	tree, err := s.stor.scriptsStorage.newestScriptByAddr(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script by account '%s'", account.String())
	}
	return tree, nil
}

func (s *stateManager) NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script bytes by account '%s'", account.String())
	}
	script, err := s.stor.scriptsStorage.newestScriptBytesByAddr(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script bytes by account '%s'", account.String())
	}
	return script, nil
}

func (s *stateManager) NewestScriptByAsset(asset crypto.Digest) (*ast.Tree, error) {
	assetID := proto.AssetIDFromDigest(asset)
	return s.stor.scriptsStorage.newestScriptByAsset(assetID)
}

// NewestBlockInfoByHeight returns block info by height.
func (s *stateManager) NewestBlockInfoByHeight(height proto.Height) (*proto.BlockInfo, error) {
	header, err := s.NewestHeaderByHeight(height)
	if err != nil {
		return nil, err
	}
	generator, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, header.GeneratorPublicKey)
	if err != nil {
		return nil, err
	}

	vrf, err := s.newestBlockVRF(header, height)
	if err != nil {
		return nil, err
	}
	rewards, err := s.newestBlockRewards(generator, height)
	if err != nil {
		return nil, err
	}

	return proto.BlockInfoFromHeader(header, generator, height, vrf, rewards)
}

func (s *stateManager) setGenesisBlock(genesisBlock *proto.Block) {
	s.genesis = genesisBlock
}

func (s *stateManager) TxValidation(func(TxValidation) error) error {
	panic("call TxValidation method on non thread safe state")
}

func (s *stateManager) MapR(func(StateInfo) (interface{}, error)) (interface{}, error) {
	panic("call MapR on non thread safe state")
}

func (s *stateManager) Map(func(State) error) error {
	panic("call Map on non thread safe state")
}

func (s *stateManager) addGenesisBlock() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initSH, shErr := crypto.FastHash(nil) // zero/initial snapshot state hash according to the specification
	if shErr != nil {
		return shErr
	}

	chans := launchVerifier(ctx, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	if err := s.addNewBlock(s.genesis, nil, chans, 0, nil, nil, initSH); err != nil {
		return err
	}
	if err := s.stor.hitSources.appendBlockHitSource(s.genesis, 1, s.genesis.GenSignature); err != nil {
		return err
	}

	err := s.appender.diffApplier.validateBalancesChanges(s.appender.diffStor.allChanges())
	if err != nil {
		return err
	}

	if err := s.stor.prepareHashes(); err != nil {
		return err
	}
	if _, err := s.stor.putStateHash(nil, 1, s.genesis.BlockID()); err != nil {
		return err
	}
	if verifyError := chans.closeAndWait(); verifyError != nil {
		return wrapErr(ValidationError, verifyError)
	}

	if err := s.flush(); err != nil {
		return wrapErr(ModificationError, err)
	}
	s.reset()
	return nil
}

func (s *stateManager) applyPreActivatedFeatures(features []int16, blockID proto.BlockID) error {
	for _, featureID := range features {
		approvalRequest := &approvedFeaturesRecord{1}
		if err := s.stor.features.approveFeature(featureID, approvalRequest, blockID); err != nil {
			return err
		}
		activationRequest := &activatedFeaturesRecord{1}
		if err := s.stor.features.activateFeature(featureID, activationRequest, blockID); err != nil {
			return err
		}
	}
	if err := s.flush(); err != nil {
		return err
	}
	s.reset()
	return nil
}

func (s *stateManager) checkProtobufActivation(height uint64) {
	activated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockV5), height)
	if activated {
		s.rw.setProtobufActivated()
	}
}

func (s *stateManager) loadLastBlock() error {
	height, err := s.Height()
	if err != nil {
		return errors.Errorf("failed to retrieve height: %v", err)
	}
	lastBlock, err := s.BlockByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get block by height: %v", err)
	}
	s.lastBlock.Store(lastBlock)
	return nil
}

func (s *stateManager) TopBlock() *proto.Block {
	return s.lastBlock.Load().(*proto.Block)
}

func blockVRFCommon(
	settings *settings.BlockchainSettings,
	blockHeader *proto.BlockHeader,
	blockHeight proto.Height,
	hitSourceProvider func(proto.Height) ([]byte, error),
) ([]byte, error) {
	if blockHeader.Version < proto.ProtobufBlockVersion {
		return nil, nil
	}
	pos := consensus.NewFairPosCalculator(settings.DelayDelta, settings.MinBlockTime)
	// use previous height for VRF calculation because given block height should not be included in VRF calculation
	prevHeight := blockHeight - 1
	p := pos.HeightForHit(prevHeight)
	refHitSource, err := hitSourceProvider(p)
	if err != nil {
		return nil, err
	}
	gsp := consensus.VRFGenerationSignatureProvider
	ok, vrf, err := gsp.VerifyGenerationSignature(blockHeader.GeneratorPublicKey, refHitSource, blockHeader.GenSignature)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("invalid VRF")
	}
	return vrf, nil
}

// newestBlockVRF calculates VRF value for the block at given height.
// If block version is less than protobuf block version, returns nil.
func (s *stateManager) newestBlockVRF(blockHeader *proto.BlockHeader, blockHeight proto.Height) ([]byte, error) {
	return blockVRFCommon(s.settings, blockHeader, blockHeight, s.NewestHitSourceAtHeight)
}

// BlockVRF calculates VRF value for the block at given height.
// If block version is less than protobuf block version, returns nil.
func (s *stateManager) BlockVRF(blockHeader *proto.BlockHeader, blockHeight proto.Height) ([]byte, error) {
	return blockVRFCommon(s.settings, blockHeader, blockHeight, s.HitSourceAtHeight)
}

func blockRewardsCommon(
	generatorAddress proto.WavesAddress,
	height proto.Height,
	settings *settings.BlockchainSettings,
	stor *blockchainEntitiesStorage,
	blockRewardActivationHeight proto.Height,
) (proto.Rewards, error) {
	reward, err := stor.monetaryPolicy.rewardAtHeight(height, blockRewardActivationHeight)
	if err != nil {
		return nil, err
	}
	c := newRewardsCalculator(settings, stor.features)
	return c.calculateRewards(generatorAddress, height, reward)
}

// newestBlockRewards calculates block rewards for the block at given height with given generator address.
// If block reward feature is not activated at given height, returns empty proto.Rewards.
func (s *stateManager) newestBlockRewards(generator proto.WavesAddress, height proto.Height) (proto.Rewards, error) {
	blockRewardActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockReward), height)
	if !blockRewardActivated {
		return proto.Rewards{}, nil
	}
	blockRewardActivationHeight, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return nil, err
	}
	return blockRewardsCommon(generator, height, s.settings, s.stor, blockRewardActivationHeight)
}

// BlockRewards calculates block rewards for the block at given height with given generator address.
// If block reward feature is not activated at given height, returns empty proto.Rewards.
func (s *stateManager) BlockRewards(generator proto.WavesAddress, height proto.Height) (proto.Rewards, error) {
	blockRewardActivated := s.stor.features.isActivatedAtHeight(int16(settings.BlockReward), height)
	if !blockRewardActivated {
		return proto.Rewards{}, nil
	}
	blockRewardActivationHeight, err := s.stor.features.activationHeight(int16(settings.BlockReward))
	if err != nil {
		return nil, err
	}
	return blockRewardsCommon(generator, height, s.settings, s.stor, blockRewardActivationHeight)
}

func (s *stateManager) Header(blockID proto.BlockID) (*proto.BlockHeader, error) {
	header, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return header, nil
}

func (s *stateManager) NewestHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	header, err := s.rw.readNewestBlockHeaderByHeight(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return header, nil
}

func (s *stateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Header(blockID)
}

func (s *stateManager) Block(blockID proto.BlockID) (*proto.Block, error) {
	block, err := s.rw.readBlock(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return block, nil
}

func (s *stateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Block(blockID)
}

func (s *stateManager) NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error) {
	leaseFromStore, err := s.stor.leases.newestLeasingInfo(id)
	if err != nil {
		return nil, err
	}
	sender, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, leaseFromStore.SenderPK)
	if err != nil {
		return nil, err
	}
	leaseInfo := proto.LeaseInfo{
		Sender:      sender,
		Recipient:   leaseFromStore.RecipientAddr,
		IsActive:    leaseFromStore.isActive(),
		LeaseAmount: leaseFromStore.Amount,
	}
	return &leaseInfo, nil
}

func (s *stateManager) NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error) {
	info, err := s.stor.scriptsStorage.newestScriptBasicInfoByAddressID(addr.ID())
	if err != nil {
		return crypto.PublicKey{}, errors.Wrap(err, "failed to get script public key")
	}
	return info.PK, nil
}

func (s *stateManager) NewestAccountHasScript(addr proto.WavesAddress) (bool, error) {
	return s.stor.scriptsStorage.newestAccountHasScript(addr)
}

func (s *stateManager) AddingBlockHeight() (uint64, error) {
	return s.rw.addingBlockHeight(), nil
}

func (s *stateManager) NewestHeight() (uint64, error) {
	return s.rw.recentHeight(), nil
}

func (s *stateManager) Height() (uint64, error) {
	height, err := s.stateDB.getHeight()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) BlockIDToHeight(blockID proto.BlockID) (uint64, error) {
	height, err := s.rw.heightByBlockID(blockID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (proto.BlockID, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return proto.BlockID{}, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return proto.BlockID{}, wrapErr(InvalidInputError, errors.Errorf("HeightToBlockID: height %d out of valid range [1, %d]", height, maxHeight))
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return proto.BlockID{}, wrapErr(RetrievalError, err)
	}
	return blockID, nil
}

func (s *stateManager) newestAssetBalance(addr proto.AddressID, asset proto.AssetID) (uint64, error) {
	// Retrieve old balance from historyStorage.
	balance, err := s.stor.balances.newestAssetBalance(addr, asset)
	if err != nil {
		return 0, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := assetBalanceKey{address: addr, asset: asset}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if errors.Is(err, errNotFound) {
		// If there is no diff, old balance is the newest.
		return balance, nil
	} else if err != nil {
		// Something weird happened.
		return 0, err
	}
	balance, aErr := diff.applyToAssetBalance(balance)
	if aErr != nil {
		return 0, errors.Errorf("given account has negative balance at this point: %v", aErr)
	}
	return balance, nil
}

func (s *stateManager) newestWavesBalanceProfile(addr proto.AddressID) (balanceProfile, error) {
	// Retrieve the latest balance from historyStorage.
	profile, err := s.stor.balances.newestWavesBalance(addr)
	if err != nil {
		return balanceProfile{}, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := wavesBalanceKey{address: addr}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if errors.Is(err, errNotFound) {
		// If there is no diff, old balance is the newest.
		return profile, nil
	} else if err != nil {
		// Something weird happened.
		return balanceProfile{}, err
	}
	newProfile, err := diff.applyTo(profile)
	if err != nil {
		return balanceProfile{}, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return newProfile, nil
}

func (s *stateManager) GeneratingBalance(account proto.Recipient, height proto.Height) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, errs.Extend(err, "failed convert recipient to address")
	}
	return s.stor.balances.generatingBalance(addr.ID(), height)
}

// NewestMinerGeneratingBalance returns the generating balance of the miner at the given height.
// This method includes the challenger bonus if the block has a challenged header.
func (s *stateManager) NewestMinerGeneratingBalance(header *proto.BlockHeader, height proto.Height) (uint64, error) {
	minerAddr, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, header.GeneratorPublicKey)
	if err != nil {
		return 0, wrapErr(RetrievalError, errors.Wrapf(err, "failed create get miner address from PK %s",
			header.GeneratorPublicKey,
		))
	}
	minerGB, err := s.stor.balances.newestGeneratingBalance(minerAddr.ID(), height)
	if err != nil {
		return 0, wrapErr(RetrievalError, errors.Wrapf(err, "failed to get generating balance for addr %s",
			minerAddr.String(),
		))
	}
	if ch, ok := header.GetChallengedHeader(); ok { // if the block has challenged header
		chMinerAddr, chErr := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, ch.GeneratorPublicKey)
		if chErr != nil {
			return 0, wrapErr(RetrievalError, errors.Wrapf(chErr, "failed to create challenged miner address from PK %s",
				ch.GeneratorPublicKey,
			))
		}
		challengerBonus, chErr := s.stor.balances.newestGeneratingBalance(chMinerAddr.ID(), height)
		if chErr != nil {
			return 0, wrapErr(RetrievalError, errors.Wrapf(chErr, "failed to get generating balance for addr %s",
				chMinerAddr.String(),
			))
		}
		minerGB += challengerBonus // add challenger bonus to challenger miner's generating balance
	}
	return minerGB, nil
}

func (s *stateManager) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, errs.Extend(err, "failed convert recipient to address")
	}
	profile, err := s.stor.balances.wavesBalance(addr.ID())
	if err != nil {
		return nil, errs.Extend(err, "failed to get waves balance")
	}
	effective, err := profile.effectiveBalanceUnchecked()
	if err != nil {
		return nil, errs.Extend(err, "failed to get effective balance")
	}
	height, err := s.Height()
	if err != nil {
		return nil, errs.Extend(err, "failed to get height")
	}
	generating, err := s.GeneratingBalance(account, height)
	if err != nil {
		return nil, errs.Extend(err, "failed to get generating balance")
	}
	if generating == 0 { // we need to check for challenged addresses only if generating balance is 0
		chEffective, effErr := profile.effectiveBalance(s.stor.balances.isChallengedAddress, addr.ID(), height)
		if effErr != nil {
			return nil, errs.Extend(effErr, "failed to get checked effective balance")
		}
		effective = chEffective
	}
	return &proto.FullWavesBalance{
		Regular:    profile.balance,
		Generating: generating,
		Available:  profile.spendableBalance(),
		Effective:  effective,
		LeaseIn:    uint64(profile.leaseIn),
		LeaseOut:   uint64(profile.leaseOut),
	}, nil
}

// NewestFullWavesBalance returns a full Waves balance of account.
// The method must be used ONLY in the Ride environment.
// The boundaries of the generating balance are calculated for the current height of applying block,
// instead of the last block height.
//
// For example, for the block validation we are use min effective balance of the account from height 1 to 1000.
// This function uses heights from 2 to 1001, where 1001 is the height of the applying block.
// All changes of effective balance during the applying block are affecting the generating balance.
func (s *stateManager) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	bp, err := s.WavesBalanceProfile(addr.ID())
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return bp.ToFullWavesBalance()
}

// WavesBalanceProfile returns WavesBalanceProfile structure retrieved by proto.AddressID of an account.
// This function always returns the newest available state of Waves balance of account.
// Thought, it can't be used during transaction processing, because the state does no hold changes between txs.
// The method must be used ONLY in the Ride environment for retrieving data from state.
// The boundaries of the generating balance are calculated for the current height of applying block,
// instead of the last block height.
//
// For example, for the block validation we are use min effective balance of the account from height 1 to 1000.
// This function uses heights from 2 to 1001, where 1001 is the height of the applying block.
// All changes of effective balance during the applying block are affecting the generating balance.
func (s *stateManager) WavesBalanceProfile(id proto.AddressID) (*types.WavesBalanceProfile, error) {
	profile, err := s.newestWavesBalanceProfile(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	// get the height of the applying block if it is in progress, or the last block height
	height, err := s.AddingBlockHeight()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	var generating uint64
	if gb, gbErr := s.stor.balances.newestGeneratingBalance(id, height); gbErr == nil {
		generating = gb
	}
	var challenged bool
	if generating == 0 { // fast path: we need to check for challenged addresses only if generating balance is 0
		ch, chErr := s.stor.balances.newestIsChallengedAddress(id, height)
		if chErr != nil {
			return nil, wrapErr(RetrievalError, chErr)
		}
		challenged = ch
	}
	return &types.WavesBalanceProfile{
		Balance:    profile.balance,
		LeaseIn:    profile.leaseIn,
		LeaseOut:   profile.leaseOut,
		Generating: generating,
		Challenged: challenged,
	}, nil
}

func (s *stateManager) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	profile, err := s.newestWavesBalanceProfile(addr.ID())
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return profile.balance, nil
}

func (s *stateManager) NewestAssetBalance(account proto.Recipient, asset crypto.Digest) (uint64, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	balance, err := s.newestAssetBalance(addr.ID(), proto.AssetIDFromDigest(asset))
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error) {
	balance, err := s.newestAssetBalance(id, proto.AssetIDFromDigest(asset))
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) WavesBalance(account proto.Recipient) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	profile, err := s.stor.balances.wavesBalance(addr.ID())
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return profile.balance, nil
}

func (s *stateManager) AssetBalance(account proto.Recipient, assetID proto.AssetID) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	balance, err := s.stor.balances.assetBalance(addr.ID(), assetID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) WavesAddressesNumber() (uint64, error) {
	res, err := s.stor.balances.wavesAddressesNumber()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return res, nil
}

func (s *stateManager) topBlock() (*proto.Block, error) {
	height, err := s.Height()
	if err != nil {
		return nil, err
	}
	// Heights start from 1.
	return s.BlockByHeight(height)
}

func (s *stateManager) addFeaturesVotes(block *proto.Block) error {
	// For Block version 2 Features are always empty, so we don't add anything.
	for _, featureID := range block.Features {
		approved, err := s.stor.features.newestIsApproved(featureID)
		if err != nil {
			return err
		}
		if approved {
			continue
		}
		if err := s.stor.features.addVote(featureID, block.BlockID()); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addRewardVote(block *proto.Block, height uint64) error {
	activation, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	isCappedRewardsActivated, err := s.stor.features.newestIsActivated(int16(settings.CappedRewards))
	if err != nil {
		return err
	}
	return s.stor.monetaryPolicy.vote(block.RewardVote, height, activation, isCappedRewardsActivated, block.BlockID())
}

func (s *stateManager) addNewBlock(
	block, parent *proto.Block,
	chans *verifierChans,
	blockchainHeight uint64,
	optionalSnapshot *proto.BlockSnapshot,
	fixSnapshotsToInitialHash []proto.AtomicSnapshot,
	lastSnapshotStateHash crypto.Digest,
) error {
	blockHeight := blockchainHeight + 1
	if err := s.beforeAppendBlock(block, blockHeight); err != nil {
		return err
	}
	transactions := block.Transactions
	if block.TransactionCount != transactions.Count() {
		return errors.Errorf("block.TransactionCount != transactions.Count(), %d != %d", block.TransactionCount, transactions.Count())
	}
	var parentHeader *proto.BlockHeader
	if parent != nil {
		parentHeader = &parent.BlockHeader
	}
	params := &appendBlockParams{
		transactions:              transactions,
		chans:                     chans,
		block:                     &block.BlockHeader,
		parent:                    parentHeader,
		blockchainHeight:          blockchainHeight,
		fixSnapshotsToInitialHash: fixSnapshotsToInitialHash,
		lastSnapshotStateHash:     lastSnapshotStateHash,
		optionalSnapshot:          optionalSnapshot,
	}
	// Check and perform block's transactions, create balance diffs, write transactions to storage.
	if err := s.appender.appendBlock(params); err != nil {
		return err
	}
	return s.afterAppendBlock(block, blockHeight)
}

func (s *stateManager) beforeAppendBlock(block *proto.Block, blockHeight proto.Height) error {
	// Add score.
	if err := s.stor.scores.appendBlockScore(block, blockHeight); err != nil {
		return err
	}
	// Handle challenged header if it exists.
	// Light node fields check performed in ValidateHeaderBeforeBlockApplying.
	if chErr := s.handleChallengedHeaderIfExists(block, blockHeight); chErr != nil {
		return chErr
	}
	// Indicate new block for storage.
	if err := s.rw.startBlock(block.BlockID()); err != nil {
		return err
	}
	// Save block header to block storage.
	return s.rw.writeBlockHeader(&block.BlockHeader)
}

func (s *stateManager) handleChallengedHeaderIfExists(block *proto.Block, blockHeight proto.Height) error {
	if blockHeight < 2 { // no challenges for genesis block
		return nil
	}
	challengedHeader, ok := block.GetChallengedHeader()
	if !ok { // nothing to do, no challenge to handle
		return nil
	}
	var (
		scheme  = s.settings.AddressSchemeCharacter
		blockID = block.BlockID()
	)
	challenger, err := proto.NewAddressFromPublicKey(scheme, block.GeneratorPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create challenger address from public key '%s'",
			challengedHeader.GeneratorPublicKey.String(),
		)
	}
	challenged, err := proto.NewAddressFromPublicKey(scheme, challengedHeader.GeneratorPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create challenged address from public key '%s'",
			challengedHeader.GeneratorPublicKey.String(),
		)
	}
	if chErr := s.stor.balances.storeChallenge(challenger.ID(), challenged.ID(), blockHeight, blockID); chErr != nil {
		return errors.Wrapf(chErr,
			"failed to store challenge at adding block height %d for block '%s'with challenger '%s' and challenged '%s'",
			blockHeight, blockID.String(), challenger.String(), challenged.String(),
		)
	}
	return nil
}

func (s *stateManager) afterAppendBlock(block *proto.Block, blockHeight proto.Height) error {
	// Let block storage know that the current block is over.
	if err := s.rw.finishBlock(block.BlockID()); err != nil {
		return err
	}
	// when block is finished blockchain height is incremented, so we should use 'blockHeight' as height value in actions below

	// Count features votes.
	if err := s.addFeaturesVotes(block); err != nil {
		return err
	}
	blockRewardActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockReward), blockHeight)
	// Count reward vote.
	if blockRewardActivated {
		err := s.addRewardVote(block, blockHeight)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) reset() {
	s.rw.reset()
	s.stor.reset()
	s.stateDB.reset()
	s.appender.reset()
	s.atx.reset()
}

func (s *stateManager) flush() error {
	if err := s.rw.flush(); err != nil {
		return err
	}
	if err := s.stor.flush(); err != nil {
		return err
	}
	if err := s.atx.flush(); err != nil {
		return err
	}
	if err := s.stateDB.flush(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) AddBlock(block []byte) (*proto.Block, error) {
	s.newBlocks.setNewBinary([][]byte{block})
	rs, err := s.addBlocks()
	if err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	s.newBlocks.setNew([]*proto.Block{block})
	rs, err := s.addBlocks()
	if err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddBlocks(blockBytes [][]byte) error {
	s.newBlocks.setNewBinary(blockBytes)
	if _, err := s.addBlocks(); err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddBlocksWithSnapshots(blockBytes [][]byte, snapshots []*proto.BlockSnapshot) error {
	if err := s.newBlocks.setNewBinaryWithSnapshots(blockBytes, snapshots); err != nil {
		return errors.Wrap(err, "failed to set new blocks with snapshots")
	}
	if _, err := s.addBlocks(); err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddDeserializedBlocks(
	blocks []*proto.Block,
) (*proto.Block, error) {
	s.newBlocks.setNew(blocks)
	lastBlock, err := s.addBlocks()
	if err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return nil, err
	}
	return lastBlock, nil
}

func (s *stateManager) AddDeserializedBlocksWithSnapshots(
	blocks []*proto.Block,
	snapshots []*proto.BlockSnapshot,
) (*proto.Block, error) {
	if err := s.newBlocks.setNewWithSnapshots(blocks, snapshots); err != nil {
		return nil, errors.Wrap(err, "failed to set new blocks with snapshots")
	}
	lastBlock, err := s.addBlocks()
	if err != nil {
		if syncErr := s.rw.syncWithDb(); syncErr != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v",
				stderrs.Join(err, syncErr),
			)
		}
		return nil, err
	}
	return lastBlock, nil
}

func (s *stateManager) needToFinishVotingPeriod(blockchainHeight proto.Height) bool {
	nextBlockHeight := blockchainHeight + 1
	votingFinishHeight := (nextBlockHeight % s.settings.ActivationWindowSize(nextBlockHeight)) == 0
	return votingFinishHeight
}

func (s *stateManager) needToRecalculateVotesAfterCappedRewardActivationInVotingPeriod(height proto.Height) (bool, error) {
	cappedRewardsActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.CappedRewards), height)
	if !cappedRewardsActivated { // nothing to do
		return false, nil
	}
	cappedRewardsHeight, err := s.stor.features.newestActivationHeight(int16(settings.CappedRewards))
	if err != nil {
		return false, err
	}
	if height != cappedRewardsHeight { // nothing to do, height is not capped
		return false, nil
	}
	// we're on cappedRewardsHeight, check whether current height is included in voting period or not
	start, end, err := s.blockRewardVotingPeriod(height)
	if err != nil {
		return false, err
	}
	return isBlockRewardVotingPeriod(start, end, height), nil
}

func (s *stateManager) isBlockRewardTermOver(height proto.Height) (bool, error) {
	activated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockReward), height)
	if activated {
		_, end, err := s.blockRewardVotingPeriod(height)
		if err != nil {
			return false, err
		}
		return end == height, nil
	}
	return false, nil
}

func (s *stateManager) blockRewardVotingPeriod(height proto.Height) (start, end proto.Height, err error) {
	activationHeight, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return 0, 0, err
	}
	isCappedRewardsActivated, err := s.stor.features.newestIsActivated(int16(settings.CappedRewards))
	if err != nil {
		return 0, 0, err
	}
	start, end = s.stor.monetaryPolicy.blockRewardVotingPeriod(height, activationHeight, isCappedRewardsActivated)
	return start, end, nil
}

func (s *stateManager) needToResetStolenAliases(height uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to reset stolen aliases in custom blockchains.
		return false, nil
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), height)
	if dataTxActivated {
		dataTxHeight, err := s.stor.features.newestActivationHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		return height == dataTxHeight, nil
	}
	return false, nil
}

// featureActivationHeightForHeight returns the height at which the feature is activated.
// If the feature is not activated at the given height, it returns 0.
func (s *stateManager) featureActivationHeightForHeight(f settings.Feature, h proto.Height) (proto.Height, error) {
	featureIsActivatedAtHeight := s.stor.features.newestIsActivatedAtHeight(int16(f), h)
	if !featureIsActivatedAtHeight { // feature is not activated at the given height, return 0
		return 0, nil
	}
	approvalHeight, err := s.stor.features.newestApprovalHeight(int16(f))
	if err != nil {
		return 0, err
	}
	featureHeight := approvalHeight + s.settings.ActivationWindowSize(h) // calculate feature activation height
	return featureHeight, nil
}

func (s *stateManager) needToCancelLeases(blockHeight uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to cancel leases in custom blockchains.
		return false, nil
	}
	dataTxHeight, err := s.featureActivationHeightForHeight(settings.DataTransaction, blockHeight)
	if err != nil {
		return false, err
	}
	rideV5Height, err := s.featureActivationHeightForHeight(settings.RideV5, blockHeight)
	if err != nil {
		return false, err
	}
	switch blockHeight {
	case s.settings.ResetEffectiveBalanceAtHeight:
		return true, nil
	case s.settings.BlockVersion3AfterHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	case dataTxHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	case rideV5Height:
		// Cancellation of leases to stolen aliases only required for MainNet
		return s.settings.Type == settings.MainNet, nil
	default:
		return false, nil
	}
}

// generateBlockchainFix generates snapshots for blockchain fixes at specific heights.
// For other block heights it returns nil slices.
//
// The changes should be applied in the end of the block processing in the context of the last applied block.
// Though the atomic snapshots must be hashed with initial snapshot of the applying block.
func (s *stateManager) generateBlockchainFix(
	applyingBlockHeight proto.Height,
	applyingBlockID proto.BlockID,
	readOnly bool, // if true, then no changes will be applied and any in memory changes synced to DB
) ([]proto.AtomicSnapshot, error) {
	cancelLeases, err := s.needToCancelLeases(applyingBlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if leases should be cancelled for block %s",
			applyingBlockID.String(),
		)
	}
	if !cancelLeases { // no need to generate snapshots
		return nil, nil
	}
	zap.S().Infof("Generating fix snapshots for the block %s and its height %d",
		applyingBlockID.String(), applyingBlockHeight,
	)
	fixSnapshots, err := s.generateCancelLeasesSnapshots(applyingBlockHeight, readOnly)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate fix snapshots for block %s", applyingBlockID.String())
	}
	zap.S().Infof("Generated fix snapshots count is %d for the block %s and its height %d",
		len(fixSnapshots), applyingBlockID.String(), applyingBlockHeight,
	)
	return fixSnapshots, nil
}

// applyBlockchainFix applies blockchain fixes if fix snapshots are not empty.
// The changes MUST be applied in the end of the block processing in the context of the just applied block.
func (s *stateManager) applyBlockchainFix(justAppliedBlockID proto.BlockID, fixSnapshots []proto.AtomicSnapshot) error {
	if len(fixSnapshots) == 0 { // fast path: nothing to apply
		return nil
	}
	if fixErr := s.appender.appendFixSnapshots(fixSnapshots, justAppliedBlockID); fixErr != nil {
		return errors.Wrapf(fixErr, "failed to append fix snapshots in appender for block %s",
			justAppliedBlockID.String(),
		)
	}
	return nil
}

// saveBlockchainFix saves blockchain fixes if fix snapshots are not empty.
func (s *stateManager) saveBlockchainFix(applyingBlockID proto.BlockID, fixSnapshots []proto.AtomicSnapshot) error {
	if len(fixSnapshots) == 0 { // fast path: nothing to save
		return nil
	}
	if sErr := s.stor.patches.savePatch(applyingBlockID, fixSnapshots); sErr != nil {
		return errors.Wrapf(sErr, "failed to save blockchain patch to the patches storage for the block %s",
			applyingBlockID.String(),
		)
	}
	return nil
}

func (s *stateManager) blockchainHeightAction(blockchainHeight uint64, lastBlock, nextBlock proto.BlockID) error {
	resetStolenAliases, err := s.needToResetStolenAliases(blockchainHeight)
	if err != nil {
		return err
	}
	if resetStolenAliases {
		// we're using nextBlock because it's a current block which we're going to apply
		if dsaErr := s.stor.aliases.disableStolenAliases(nextBlock); dsaErr != nil {
			return dsaErr
		}
	}
	if s.needToFinishVotingPeriod(blockchainHeight) {
		if err := s.finishVoting(blockchainHeight, lastBlock); err != nil {
			return err
		}
		if err := s.stor.features.resetVotes(nextBlock); err != nil {
			return err
		}
	}

	needToRecalculate, err := s.needToRecalculateVotesAfterCappedRewardActivationInVotingPeriod(blockchainHeight)
	if err != nil {
		return err
	}
	if needToRecalculate { // one time action
		if err := s.recalculateVotesAfterCappedRewardActivationInVotingPeriod(blockchainHeight, lastBlock); err != nil {
			return errors.Wrap(err, "failed to recalculate monetary policy votes")
		}
	}

	termIsOver, err := s.isBlockRewardTermOver(blockchainHeight)
	if err != nil {
		return err
	}
	if termIsOver {
		if ubrErr := s.updateBlockReward(lastBlock, blockchainHeight); ubrErr != nil {
			return ubrErr
		}
	}
	return nil
}

func (s *stateManager) finishVoting(height uint64, blockID proto.BlockID) error {
	nextBlockHeight := height + 1
	if err := s.stor.features.finishVoting(nextBlockHeight, blockID); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) updateBlockReward(lastBlockID proto.BlockID, height proto.Height) error {
	blockRewardActivationHeight, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	isCappedRewardsActivated, err := s.stor.features.newestIsActivated(int16(settings.CappedRewards))
	if err != nil {
		return err
	}
	return s.stor.monetaryPolicy.updateBlockReward(
		lastBlockID,
		height,
		blockRewardActivationHeight,
		isCappedRewardsActivated,
	)
}

// generateCancelLeasesSnapshots generates snapshots for lease cancellation blockchain fixes.
// If readOnly is true, then no changes will be applied and any in memory changes synced to DB.
func (s *stateManager) generateCancelLeasesSnapshots(
	blockHeight uint64,
	readOnly bool,
) ([]proto.AtomicSnapshot, error) {
	if !readOnly {
		// Move balance diffs from diffStorage to historyStorage.
		// It must be done before lease cancellation, because
		// lease cancellation iterates through historyStorage.
		if err := s.appender.moveChangesToHistoryStorage(); err != nil {
			return nil, err
		}
	}
	// prepare info about features activation
	dataTxHeight, err := s.featureActivationHeightForHeight(settings.DataTransaction, blockHeight)
	if err != nil {
		return nil, err
	}
	rideV5Height, err := s.featureActivationHeightForHeight(settings.RideV5, blockHeight)
	if err != nil {
		return nil, err
	}
	return s.generateLeasesCancellationWithNewBalancesSnapshots(blockHeight, dataTxHeight, rideV5Height)
}

func (s *stateManager) generateLeasesCancellationWithNewBalancesSnapshots(
	blockchainHeight uint64,
	dataTxHeight uint64, // the height when DataTransaction feature is activated, equals 0 if the feature is not activated
	rideV5Height uint64, // the height when RideV5 feature is activated, equals 0 if the feature is not activated
) ([]proto.AtomicSnapshot, error) {
	switch blockchainHeight {
	case s.settings.ResetEffectiveBalanceAtHeight:
		scheme := s.settings.AddressSchemeCharacter
		cancelledLeasesSnapshots, err := s.stor.leases.generateCancelledLeaseSnapshots(scheme, nil)
		if err != nil {
			return nil, err
		}
		zeroLeaseBalancesSnapshots, err := s.stor.balances.generateZeroLeaseBalanceSnapshotsForAllLeases()
		if err != nil {
			return nil, err
		}
		return joinCancelledLeasesAndLeaseBalances(cancelledLeasesSnapshots, zeroLeaseBalancesSnapshots), nil
	case s.settings.BlockVersion3AfterHeight:
		leaseBalanceSnapshots, overflowAddresses, err := s.stor.balances.generateLeaseBalanceSnapshotsForLeaseOverflows()
		if err != nil {
			return nil, err
		}
		scheme := s.settings.AddressSchemeCharacter
		cancelledLeasesSnapshots, err := s.stor.leases.generateCancelledLeaseSnapshots(scheme, overflowAddresses)
		if err != nil {
			return nil, err
		}
		return joinCancelledLeasesAndLeaseBalances(cancelledLeasesSnapshots, leaseBalanceSnapshots), nil
	case dataTxHeight:
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return nil, err
		}
		validLeaseBalances, err := s.stor.balances.generateCorrectingLeaseBalanceSnapshotsForInvalidLeaseIns(leaseIns)
		if err != nil {
			return nil, err
		}
		return joinCancelledLeasesAndLeaseBalances(nil, validLeaseBalances), nil
	case rideV5Height:
		scheme := s.settings.AddressSchemeCharacter
		cancelledLeasesSnapshots, changes, err := s.stor.leases.cancelLeasesToDisabledAliases(scheme)
		if err != nil {
			return nil, err
		}
		leaseBalanceSnapshots, err := s.stor.balances.generateLeaseBalanceSnapshotsWithProvidedChanges(changes)
		if err != nil {
			return nil, err
		}
		return joinCancelledLeasesAndLeaseBalances(cancelledLeasesSnapshots, leaseBalanceSnapshots), nil
	default:
		return nil, nil
	}
}

func joinCancelledLeasesAndLeaseBalances(
	cancelledLeases []proto.CancelledLeaseSnapshot,
	leaseBalancesSnapshots []proto.LeaseBalanceSnapshot,
) []proto.AtomicSnapshot {
	l := len(cancelledLeases) + len(leaseBalancesSnapshots)
	if l == 0 {
		return nil
	}
	res := make([]proto.AtomicSnapshot, 0, l)
	for i := range cancelledLeases {
		cl := &cancelledLeases[i]
		res = append(res, cl)
	}
	for i := range leaseBalancesSnapshots {
		lbs := &leaseBalancesSnapshots[i]
		res = append(res, lbs)
	}
	return res
}

func (s *stateManager) recalculateVotesAfterCappedRewardActivationInVotingPeriod(height proto.Height, lastBlockID proto.BlockID) error {
	start, end, err := s.blockRewardVotingPeriod(height)
	if err != nil {
		return err
	}
	if !isBlockRewardVotingPeriod(start, end, height) { // sanity check
		return errors.Errorf("height %d is not in voting period %d:%d", height, start, end)
	}
	blockRewardActivationHeight, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	isCappedRewardsActivated, err := s.stor.features.newestIsActivated(int16(settings.CappedRewards))
	if err != nil {
		return err
	}
	for h := start; h <= height; h++ {
		header, err := s.NewestHeaderByHeight(h)
		if err != nil {
			return errors.Wrapf(err, "failed to get newest header by height %d", h)
		}
		// rewrite rewardVotes on h == start and count votes for the rest heights
		if err := s.stor.monetaryPolicy.vote(header.RewardVote, h, blockRewardActivationHeight, isCappedRewardsActivated, lastBlockID); err != nil {
			return errors.Wrapf(err, "failed to add vote for monetary policy at height %d for block %q", height, lastBlockID.String())
		}
	}
	return nil
}

func (s *stateManager) addBlocks() (_ *proto.Block, retErr error) { //nolint:nonamedreturns // needs in defer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		// Reset in-memory storages and load last block in defer.
		s.reset()
		if lbErr := s.loadLastBlock(); lbErr != nil {
			zap.S().Fatalf("Failed to load last block: %v", stderrs.Join(retErr, lbErr))
		}
		s.newBlocks.reset()
	}()

	blocksNumber := s.newBlocks.len()
	if blocksNumber == 0 {
		return nil, wrapErr(InvalidInputError, errors.New("no blocks provided"))
	}

	// Read some useful values for later.
	lastAppliedBlock, tbErr := s.topBlock()
	if tbErr != nil {
		return nil, wrapErr(RetrievalError, tbErr)
	}
	zap.S().Debugf("StateManager: parent (top) block ID: %s, ts: %d", lastAppliedBlock.BlockID().String(), lastAppliedBlock.Timestamp)
	height, hErr := s.Height()
	if hErr != nil {
		return nil, wrapErr(RetrievalError, hErr)
	}
	headers := make([]proto.BlockHeader, blocksNumber)

	// Launch verifier that checks signatures of blocks and transactions.
	chans := launchVerifier(ctx, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	var (
		ids []proto.BlockID
	)
	pos := 0
	for s.newBlocks.next() {
		blockchainCurHeight := height + uint64(pos)
		block, optionalSnapshot, errCurBlock := s.newBlocks.current()
		if errCurBlock != nil {
			return nil, wrapErr(DeserializationError, errCurBlock)
		}

		pErr := s.processBlockInPack(block, optionalSnapshot, lastAppliedBlock, blockchainCurHeight, chans)
		if pErr != nil {
			return nil, pErr
		}

		// Prepare for the next iteration.
		headers[pos] = block.BlockHeader
		pos++
		ids = append(ids, block.BlockID())
		lastAppliedBlock = block
	}
	// Tasks chan can now be closed, since all the blocks and transactions have been already sent for verification.
	// wait for all verifier goroutines
	if verifyError := chans.closeAndWait(); verifyError != nil {
		return nil, wrapErr(ValidationError, verifyError)
	}

	// Retrieve and store legacy state hashes for each of new blocks.
	if shErr := s.stor.handleLegacyStateHashes(height, ids); shErr != nil {
		return nil, wrapErr(ModificationError, shErr)
	}
	// Validate consensus (i.e. that all the new blocks were mined fairly).
	if vErr := s.cv.ValidateHeadersBatch(headers[:pos], height); vErr != nil {
		return nil, wrapErr(ValidationError, vErr)
	}
	// After everything is validated, save all the changes to DB.
	if fErr := s.flush(); fErr != nil {
		return nil, wrapErr(ModificationError, fErr)
	}
	zap.S().Infof(
		"Height: %d; Block ID: %s, GenSig: %s, ts: %d",
		height+uint64(blocksNumber),
		lastAppliedBlock.BlockID().String(),
		base58.Encode(lastAppliedBlock.GenSignature),
		lastAppliedBlock.Timestamp,
	)
	return lastAppliedBlock, nil
}

func (s *stateManager) processBlockInPack(
	block *proto.Block,
	optionalSnapshot *proto.BlockSnapshot,
	lastAppliedBlock *proto.Block,
	blockchainCurHeight uint64,
	chans *verifierChans,
) error {
	if badErr := s.beforeAddingBlock(block, lastAppliedBlock, blockchainCurHeight, chans); badErr != nil {
		return badErr
	}
	sh, errSh := s.stor.stateHashes.newestSnapshotStateHash(blockchainCurHeight)
	if errSh != nil {
		return errors.Wrapf(errSh, "failed to get newest snapshot state hash for height %d",
			blockchainCurHeight,
		)
	}

	// Generate blockchain fix snapshots for the applying block.
	fixSnapshots, gbfErr := s.generateBlockchainFix(blockchainCurHeight+1, block.BlockID(), false)
	if gbfErr != nil {
		return errors.Wrapf(gbfErr, "failed to generate blockchain fix snapshots at block %s",
			block.BlockID().String(),
		)
	}
	if sbfErr := s.saveBlockchainFix(block.BlockID(), fixSnapshots); sbfErr != nil {
		return wrapErr(ModificationError, errors.Wrapf(sbfErr, "failed to save blockchain fix for block %s",
			block.BlockID().String()),
		)
	}

	fixSnapshotsToInitialHash := fixSnapshots // at the block applying stage fix snapshots are only used for hashing
	// Save block to storage, check its transactions, create and save balance diffs for its transactions.
	addErr := s.addNewBlock(
		block, lastAppliedBlock, chans, blockchainCurHeight, optionalSnapshot, fixSnapshotsToInitialHash, sh)
	if addErr != nil {
		return addErr
	}
	if fixErr := s.applyBlockchainFix(block.BlockID(), fixSnapshots); fixErr != nil {
		return errors.Wrapf(fixErr, "failed to apply fix snapshots after block %s applying",
			block.BlockID().String(),
		)
	}
	blockchainCurHeight++ // we've just added a new block and applied blockchain fix, so we have a new height

	if s.needToFinishVotingPeriod(blockchainCurHeight) {
		// If we need to finish voting period on the next block (h+1) then
		// we have to check that protobuf will be activated on next block
		s.checkProtobufActivation(blockchainCurHeight + 1)
	}
	return nil
}

func (s *stateManager) beforeAddingBlock(
	block, lastAppliedBlock *proto.Block,
	blockchainCurHeight proto.Height,
	chans *verifierChans,
) error {
	// Assign unique block number for this block ID, add this number to the list of valid blocks.
	if blErr := s.stateDB.addBlock(block.BlockID()); blErr != nil {
		return wrapErr(ModificationError, blErr)
	}
	// At some blockchain heights specific logic is performed.
	// This includes voting for features, block rewards and so on.
	if err := s.blockchainHeightAction(blockchainCurHeight, lastAppliedBlock.BlockID(), block.BlockID()); err != nil {
		return wrapErr(ModificationError, err)
	}
	if vhErr := s.cv.ValidateHeaderBeforeBlockApplying(&block.BlockHeader, blockchainCurHeight); vhErr != nil {
		return vhErr
	}
	// Send block for signature verification, which works in separate goroutine.
	task := &verifyTask{
		taskType: verifyBlock,
		parentID: lastAppliedBlock.BlockID(),
		block:    block,
	}
	if err := chans.trySend(task); err != nil {
		return err
	}
	hs, err := s.cv.GenerateHitSource(blockchainCurHeight, block.BlockHeader)
	if err != nil {
		return err
	}

	return s.stor.hitSources.appendBlockHitSource(block, blockchainCurHeight+1, hs)
}

func (s *stateManager) checkRollbackHeight(height uint64) error {
	maxHeight, err := s.Height()
	if err != nil {
		return err
	}
	minRollbackHeight, err := s.stateDB.getRollbackMinHeight()
	if err != nil {
		return err
	}
	if height < minRollbackHeight || height > maxHeight {
		return errors.Errorf("invalid height; valid range is: [%d, %d]", minRollbackHeight, maxHeight)
	}
	return nil
}

func (s *stateManager) checkRollbackInput(blockID proto.BlockID) error {
	height, err := s.BlockIDToHeight(blockID)
	if err != nil {
		return err
	}
	return s.checkRollbackHeight(height)
}

func (s *stateManager) RollbackToHeight(height uint64) error {
	if err := s.checkRollbackHeight(height); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	return s.rollbackToImpl(blockID)
}

func (s *stateManager) rollbackToImpl(removalEdge proto.BlockID) error {
	// The database part of rollback.
	if err := s.stateDB.rollback(removalEdge); err != nil {
		return wrapErr(RollbackError, err)
	}
	// After this point Fatalf() is called instead of returning errors,
	// because exiting would lead to incorrect state.
	// Remove blocks from block storage by syncing block storage with the database.
	if err := s.rw.syncWithDb(); err != nil {
		zap.S().Fatalf("Failed to sync block storage with db: %v", err)
	}
	// Clear scripts cache after rollback.
	if err := s.stor.scriptsStorage.clearCache(); err != nil {
		zap.S().Fatalf("Failed to clear scripts cache after rollback: %v", err)
	}
	// Clear features cache
	s.stor.features.clearCache()

	if err := s.stor.flush(); err != nil {
		zap.S().Fatalf("Failed to flush history storage cache after rollback: %v", err)
	}

	if err := s.loadLastBlock(); err != nil {
		zap.S().Fatalf("Failed to load last block after rollback: %v", err)
	}
	zap.S().Infof("Rollback to block with ID '%s' completed", removalEdge.String())
	return nil
}

func (s *stateManager) RollbackTo(removalEdge proto.BlockID) error {
	if err := s.checkRollbackInput(removalEdge); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	return s.rollbackToImpl(removalEdge)
}

func (s *stateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("ScoreAtHeight: %d height out of valid range [1, %d]", height, maxHeight))
	}
	score, err := s.stor.scores.score(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return score, nil
}

func (s *stateManager) HitSourceAtHeight(height uint64) ([]byte, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("HitSourceAtHeight: height %d out of valid range [1, %d]", height, maxHeight))
	}
	return s.stor.hitSources.hitSource(height)
}

func (s *stateManager) NewestHitSourceAtHeight(height uint64) ([]byte, error) {
	maxHeight, err := s.NewestHeight()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("NewestHitSourceAtHeight: height %d out of valid range [1, %d]", height, maxHeight))
	}
	return s.stor.hitSources.newestHitSource(height)
}

func (s *stateManager) CurrentScore() (*big.Int, error) {
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	score, err := s.stor.scores.score(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return score, nil
}

func (s *stateManager) NewestRecipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error) {
	if addr := recipient.Address(); addr != nil {
		return *addr, nil
	}
	return s.stor.aliases.newestAddrByAlias(recipient.Alias().Alias)
}

func (s *stateManager) recipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error) {
	if addr := recipient.Address(); addr != nil {
		return *addr, nil
	}
	return s.stor.aliases.addrByAlias(recipient.Alias().Alias)
}

func (s *stateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	cp := *s.settings
	return &cp, nil
}

func (s *stateManager) ResetValidationList() {
	s.reset()
	if err := s.stor.scriptsStorage.clearCache(); err != nil {
		zap.S().Fatalf("Failed to clearCache scripts cache after UTX validation: %v", err)
	}
}

// ValidateNextTx function must be used for UTX validation only.
func (s *stateManager) ValidateNextTx(
	tx proto.Transaction,
	currentTimestamp,
	parentTimestamp uint64,
	v proto.BlockVersion,
	acceptFailed bool,
) ([]proto.AtomicSnapshot, error) {
	return s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp, v, acceptFailed)
}

func (s *stateManager) CreateNextSnapshotHash(block *proto.Block) (crypto.Digest, error) {
	blockchainHeight, err := s.Height()
	if err != nil {
		return crypto.Digest{}, err
	}
	lastSnapshotStateHash, err := s.stor.stateHashes.snapshotStateHash(blockchainHeight)
	if err != nil {
		return crypto.Digest{}, err
	}
	blockHeight := blockchainHeight + 1
	// Generate blockchain fix snapshots for the given block in read only mode because all
	// changes has been already applied in the context of the last applied block.
	fixSnapshots, gbfErr := s.generateBlockchainFix(blockHeight, block.BlockID(), true)
	if gbfErr != nil {
		return crypto.Digest{}, errors.Wrapf(gbfErr, "failed to generate blockchain fix snapshots at block %s",
			block.BlockID().String(),
		)
	}
	if len(fixSnapshots) != 0 {
		zap.S().Infof(
			"Last fix snapshots has been generated for the snapshot hash calculation of the block %s with height %d",
			block.BlockID().String(),
			blockHeight,
		)
	}
	return s.appender.createNextSnapshotHash(block, blockHeight, lastSnapshotStateHash, fixSnapshots)
}

func (s *stateManager) IsActiveLightNodeNewBlocksFields(blockHeight proto.Height) (bool, error) {
	return s.cv.ShouldIncludeNewBlockFieldsOfLightNodeFeature(blockHeight)
}

func (s *stateManager) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	addr, err := s.stor.aliases.newestAddrByAlias(alias.Alias)
	if err != nil {
		return proto.WavesAddress{}, wrapErr(RetrievalError, err)
	}
	return addr, nil
}

func (s *stateManager) AddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	addr, err := s.stor.aliases.addrByAlias(alias.Alias)
	if err != nil {
		return proto.WavesAddress{}, wrapErr(RetrievalError, err)
	}
	return addr, nil
}

func (s *stateManager) AliasesByAddr(addr proto.WavesAddress) ([]string, error) {
	aliases, err := s.stor.aliases.aliasesByAddr(addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return aliases, nil
}

func (s *stateManager) VotesNumAtHeight(featureID int16, height proto.Height) (uint64, error) {
	votesNum, err := s.stor.features.featureVotesAtHeight(featureID, height)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return votesNum, nil
}

func (s *stateManager) VotesNum(featureID int16) (uint64, error) {
	votesNum, err := s.stor.features.featureVotes(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return votesNum, nil
}

func (s *stateManager) IsActivated(featureID int16) (bool, error) {
	activated, err := s.stor.features.newestIsActivated(featureID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return activated, nil
}

func (s *stateManager) NewestIsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	return s.stor.features.newestIsActivatedAtHeight(featureID, height), nil
}

func (s *stateManager) IsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	return s.stor.features.isActivatedAtHeight(featureID, height), nil
}

func (s *stateManager) NewestActivationHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.newestActivationHeight(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) ActivationHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.activationHeight(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) IsApproved(featureID int16) (bool, error) {
	approved, err := s.stor.features.isApproved(featureID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return approved, nil
}

func (s *stateManager) IsApprovedAtHeight(featureID int16, height uint64) (bool, error) {
	return s.stor.features.isApprovedAtHeight(featureID, height), nil
}

func (s *stateManager) ApprovalHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.approvalHeight(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) AllFeatures() ([]int16, error) {
	features, err := s.stor.features.allFeatures()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return features, nil
}

func (s *stateManager) EstimatorVersion() (int, error) {
	rideV6, err := s.IsActivated(int16(settings.RideV6))
	if err != nil {
		return 0, err
	}
	if rideV6 {
		return 4, nil
	}

	blockV5, err := s.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return 0, err
	}
	if blockV5 {
		return 3, nil
	}

	blockReward, err := s.IsActivated(int16(settings.BlockReward))
	if err != nil {
		return 0, err
	}
	if blockReward {
		return 2, nil
	}

	smartAccounts, err := s.IsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return 0, err
	}
	if smartAccounts {
		return 1, nil
	}
	return 0, errors.New("inactive RIDE")
}

// Accounts data storage.

func (s *stateManager) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entries, err := s.stor.accountsDataStor.retrieveEntries(addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entries, nil
}

func (s *stateManager) IsStateUntouched(account proto.Recipient) (bool, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	entryExist, err := s.stor.accountsDataStor.newestEntryExists(addr)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return !entryExist, nil
}

func (s *stateManager) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestIntegerEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveIntegerEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBooleanEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveBooleanEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestStringEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveStringEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBinaryEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveBinaryEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

// NewestTransactionByID returns transaction by given ID. This function must be used only in Ride evaluator.
// WARNING! Function returns error if a transaction exists but failed or elided.
func (s *stateManager) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	tx, status, err := s.rw.readNewestTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if status.IsNotSucceeded() {
		return nil, wrapErr(RetrievalError, errors.Errorf("transaction is not succeeded, status=%d", status))
	}
	return tx, nil
}

func (s *stateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	tx, _, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return tx, nil
}

func (s *stateManager) TransactionByIDWithStatus(id []byte) (proto.Transaction, proto.TransactionStatus, error) {
	tx, status, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, 0, wrapErr(RetrievalError, err)
	}
	return tx, status, nil
}

// NewestTransactionHeightByID returns transaction's height by given ID. This function must be used only in Ride evaluator.
// WARNING! Function returns error if a transaction exists but failed.
func (s *stateManager) NewestTransactionHeightByID(id []byte) (uint64, error) {
	txHeight, status, err := s.rw.newestTransactionHeightByID(id)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	if status.IsNotSucceeded() {
		return 0, wrapErr(RetrievalError, errors.Errorf("transaction is not succeeded, status=%d", status))
	}
	return txHeight, nil
}

func (s *stateManager) TransactionHeightByID(id []byte) (uint64, error) {
	txHeight, _, err := s.rw.transactionHeightByID(id)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return txHeight, nil
}

func (s *stateManager) NewAddrTransactionsIterator(addr proto.Address) (TransactionIterator, error) {
	providesData, err := s.ProvidesExtendedApi()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !providesData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for transactions by address API"))
	}
	iter, err := s.atx.newTransactionsByAddrIterator(addr)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return iter, nil
}

func (s *stateManager) NewestAssetIsSponsored(asset crypto.Digest) (bool, error) {
	assetID := proto.AssetIDFromDigest(asset)
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) AssetIsSponsored(assetID proto.AssetID) (bool, error) {
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) NewestAssetConstInfo(assetID proto.AssetID) (*proto.AssetConstInfo, error) {
	info, err := s.stor.assets.newestConstInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.Issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.AssetConstInfo{
		ID:          proto.ReconstructDigest(assetID, info.Tail),
		IssueHeight: info.IssueHeight,
		Issuer:      issuer,
		Decimals:    info.Decimals,
	}, nil
}

func (s *stateManager) NewestAssetInfo(asset crypto.Digest) (*proto.AssetInfo, error) {
	assetID := proto.AssetIDFromDigest(asset)
	info, err := s.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.Issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.AssetInfo{
		AssetConstInfo: proto.AssetConstInfo{
			ID:          proto.ReconstructDigest(assetID, info.Tail),
			IssueHeight: info.IssueHeight,
			Issuer:      issuer,
			Decimals:    info.Decimals,
		},
		Quantity:        info.quantity.Uint64(),
		IssuerPublicKey: info.Issuer,

		Reissuable: info.reissuable,
		Scripted:   scripted,
		Sponsored:  sponsored,
	}, nil
}

// NewestFullAssetInfo is used to request full asset info from RIDE,
// because of that we don't try to get issue transaction info.
func (s *stateManager) NewestFullAssetInfo(asset crypto.Digest) (*proto.FullAssetInfo, error) {
	ai, err := s.NewestAssetInfo(asset)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	assetID := proto.AssetIDFromDigest(asset)
	info, err := s.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: nil, // Always return nil in this function because this field is not used later on
	}
	isSponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.newestAssetCost(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.NewestWavesBalance(proto.NewRecipientFromAddress(ai.Issuer))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if isScripted {
		scriptInfo, err := s.NewestScriptInfoByAsset(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.ScriptInfo = *scriptInfo
	}
	return res, nil
}

func (s *stateManager) IsAssetExist(assetID proto.AssetID) (bool, error) {
	// this is the fastest way to understand whether asset exist or not
	switch _, err := s.stor.assets.constInfo(assetID); {
	case err == nil:
		return true, nil
	case errors.Is(err, errs.UnknownAsset{}):
		return false, nil
	default:
		return false, wrapErr(RetrievalError, err)
	}
}

// AssetInfo returns stable (stored in DB) information about an asset by given ID.
// If there is no asset for the given ID error of type `errs.UnknownAsset` is returned.
// Errors of types `state.RetrievalError` returned in case of broken DB.
func (s *stateManager) AssetInfo(assetID proto.AssetID) (*proto.AssetInfo, error) {
	info, err := s.stor.assets.assetInfo(assetID)
	if err != nil {
		if errors.Is(err, errs.UnknownAsset{}) {
			return nil, err
		}
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.Issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted, err := s.stor.scriptsStorage.isSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return &proto.AssetInfo{
		AssetConstInfo: proto.AssetConstInfo{
			ID:          proto.ReconstructDigest(assetID, info.Tail),
			IssueHeight: info.IssueHeight,
			Issuer:      issuer,
			Decimals:    info.Decimals,
		},
		Quantity:        info.quantity.Uint64(),
		IssuerPublicKey: info.Issuer,

		Reissuable: info.reissuable,
		Scripted:   scripted,
		Sponsored:  sponsored,
	}, nil
}

func (s *stateManager) FullAssetInfo(assetID proto.AssetID) (*proto.FullAssetInfo, error) {
	ai, err := s.AssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	info, err := s.stor.assets.assetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	txID := crypto.Digest(ai.ID)             // explicitly show that full asset ID is a crypto.Digest and equals txID
	tx, _ := s.TransactionByID(txID.Bytes()) // Explicitly ignore error here, in case of error tx is nil as expected
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: tx,
	}

	isSponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.assetCost(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.WavesBalance(proto.NewRecipientFromAddress(ai.Issuer))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted, err := s.stor.scriptsStorage.isSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isScripted {
		scriptInfo, err := s.ScriptInfoByAsset(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.ScriptInfo = *scriptInfo
	}
	return res, nil
}

func (s *stateManager) EnrichedFullAssetInfo(assetID proto.AssetID) (*proto.EnrichedFullAssetInfo, error) {
	fa, err := s.FullAssetInfo(assetID)
	if err != nil {
		return nil, err
	}
	constInfo, err := s.stor.assets.constInfo(assetID)
	if err != nil {
		if errors.Is(err, errs.UnknownAsset{}) {
			return nil, err
		}
		return nil, wrapErr(RetrievalError, err)
	}
	res := &proto.EnrichedFullAssetInfo{
		FullAssetInfo:   *fa,
		SequenceInBlock: constInfo.IssueSequenceInBlock,
	}
	return res, nil
}

func (s *stateManager) NFTList(account proto.Recipient, limit uint64, afterAssetID *proto.AssetID) ([]*proto.FullAssetInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	nfts, err := s.stor.balances.nftList(addr.ID(), limit, afterAssetID, height, s.stor.features)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	infos := make([]*proto.FullAssetInfo, len(nfts))
	for i, nft := range nfts {
		info, err := s.FullAssetInfo(proto.AssetIDFromDigest(nft))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		infos[i] = info
	}
	return infos, nil
}

func (s *stateManager) ScriptBasicInfoByAccount(account proto.Recipient) (*proto.ScriptBasicInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	hasScript, err := s.stor.scriptsStorage.accountHasScript(addr)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasScript {
		return nil, proto.ErrNotFound
	}
	info, err := s.stor.scriptsStorage.scriptBasicInfoByAddressID(addr.ID())
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptBasicInfo{
		PK:             info.PK,
		ScriptLen:      info.ScriptLen,
		LibraryVersion: info.LibraryVersion,
		HasVerifier:    info.HasVerifier,
		IsDApp:         info.IsDApp,
	}, nil
}

func (s *stateManager) ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAddr(addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	est, err := s.stor.scriptsComplexity.scriptComplexityByAddress(addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) ScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	est, err := s.stor.scriptsComplexity.scriptComplexityByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) NewestScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.newestScriptBytesByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	est, err := s.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) IsActiveLeasing(leaseID crypto.Digest) (bool, error) {
	isActive, err := s.stor.leases.isActive(leaseID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return isActive, nil
}

func (s *stateManager) InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error) {
	hasData, err := s.storesExtendedApiData()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for invoke results"))
	}
	res, err := s.stor.invokeResults.invokeResult(s.settings.AddressSchemeCharacter, invokeID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return res, nil
}

func (s *stateManager) storesExtendedApiData() (bool, error) {
	stores, err := s.stateDB.stateStoresApiData()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return stores, nil
}

func (s *stateManager) ProvidesExtendedApi() (bool, error) {
	hasData, err := s.storesExtendedApiData()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	if !hasData {
		// State does not have extended API data.
		return false, nil
	}
	// State has data for extended API, but we need to make sure it is served.
	return s.atx.providesData(), nil
}

func (s *stateManager) ProvidesStateHashes() (bool, error) {
	provides, err := s.stateDB.stateStoresHashes()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return provides, nil
}

func (s *stateManager) LegacyStateHashAtHeight(height proto.Height) (*proto.StateHash, error) {
	hasData, err := s.ProvidesStateHashes()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for state hashes"))
	}
	sh, err := s.stor.stateHashes.legacyStateHash(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return sh, nil
}

func (s *stateManager) SnapshotStateHashAtHeight(height proto.Height) (crypto.Digest, error) {
	sh, err := s.stor.stateHashes.snapshotStateHash(height)
	if err != nil {
		return crypto.Digest{}, wrapErr(RetrievalError, err)
	}
	return sh, nil
}

func (s *stateManager) IsNotFound(err error) bool {
	return IsNotFound(err)
}

func (s *stateManager) StartProvidingExtendedApi() error {
	if err := s.atx.startProvidingData(); err != nil {
		return wrapErr(ModificationError, err)
	}
	return nil
}

func (s *stateManager) PersistAddressTransactions() error {
	return s.atx.persist()
}

func (s *stateManager) ShouldPersistAddressTransactions() (bool, error) {
	return s.atx.shouldPersist()
}

// RewardAtHeight return reward for the block at the given height.
// It takes into account the reward multiplier introduced with the feature #23 (Boost Block Reward).
func (s *stateManager) RewardAtHeight(height proto.Height) (uint64, error) {
	blockRewardActivated := s.stor.features.isActivatedAtHeight(int16(settings.BlockReward), height)
	if !blockRewardActivated {
		return 0, nil
	}
	blockRewardActivationHeight, err := s.stor.features.activationHeight(int16(settings.BlockReward))
	if err != nil {
		return 0, err
	}
	reward, err := s.stor.monetaryPolicy.rewardAtHeight(height, blockRewardActivationHeight)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	multiplier, err := rewardMultiplier(s.settings, s.stor.features, height)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return multiplier * reward, nil
}

func (s *stateManager) RewardVotes(height proto.Height) (proto.RewardVotes, error) {
	activation, err := s.stor.features.activationHeight(int16(settings.BlockReward))
	if err != nil {
		return proto.RewardVotes{}, err
	}
	isCappedRewardsActivated, err := s.stor.features.isActivated(int16(settings.CappedRewards))
	if err != nil {
		return proto.RewardVotes{}, err
	}
	v, err := s.stor.monetaryPolicy.votes(height, activation, isCappedRewardsActivated)
	if err != nil {
		return proto.RewardVotes{}, err
	}
	return proto.RewardVotes{Increase: v.increase, Decrease: v.decrease}, nil
}

func (s *stateManager) getInitialTotalWavesAmount() uint64 {
	totalAmount := uint64(0)
	for _, tx := range s.genesis.Transactions {
		txG, ok := tx.(*proto.Genesis)
		if !ok {
			panic(fmt.Sprintf("tx type (%T) must be genesis tx type", tx))
		}
		totalAmount += txG.Amount
	}
	return totalAmount
}

// TotalWavesAmount returns total amount of Waves in the system at the given height.
// It returns the initial Waves amount of 100 000 000 before activation of feature #14 "BlockReward".
// It takes into account the reward multiplier introduced with the feature #23 "BoostBlockReward".
func (s *stateManager) TotalWavesAmount(height proto.Height) (uint64, error) {
	initialTotalAmount := s.getInitialTotalWavesAmount()
	blockRewardActivated := s.stor.features.isActivatedAtHeight(int16(settings.BlockReward), height)
	if !blockRewardActivated {
		return initialTotalAmount, nil
	}
	blockRewardActivationHeight, err := s.stor.features.activationHeight(int16(settings.BlockReward))
	if err != nil {
		return 0, err
	}

	rewardBoostActivationHeight, rewardBoostLastHeight, err := rewardBoostFeatureInfo(height, s.stor.features, s.settings)
	if err != nil {
		return 0, err
	}

	amount, err := s.stor.monetaryPolicy.totalAmountAtHeight(height, initialTotalAmount, blockRewardActivationHeight,
		rewardBoostActivationHeight, rewardBoostLastHeight)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return amount, nil
}

func rewardBoostFeatureInfo(
	h proto.Height,
	feat featuresState,
	bs *settings.BlockchainSettings,
) (proto.Height, proto.Height, error) {
	rewardBoostActivated := feat.isActivatedAtHeight(int16(settings.BoostBlockReward), h)
	if !rewardBoostActivated {
		return 0, 0, nil
	}
	rewardBoostActivationHeight, err := feat.activationHeight(int16(settings.BoostBlockReward))
	if err != nil {
		return 0, 0, err
	}
	rewardBoostLastHeight := rewardBoostActivationHeight + bs.BlockRewardBoostPeriod - 1
	return rewardBoostActivationHeight, rewardBoostLastHeight, nil
}

func (s *stateManager) SnapshotsAtHeight(height proto.Height) (proto.BlockSnapshot, error) {
	return s.stor.snapshots.getSnapshots(height)
}

func (s *stateManager) Close() error {
	if err := s.atx.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	if err := s.rw.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	if err := s.stateDB.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	return nil
}

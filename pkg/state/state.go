package state

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
	"go.uber.org/zap"
)

const (
	rollbackMaxBlocks = 2000
	blocksStorDir     = "blocks_storage"
	keyvalueDir       = "key_value"

	maxScriptsRunsInBlock       = 101
	maxScriptsComplexityInBlock = 1000000
)

var empty struct{}

func wrapErr(stateErrorType ErrorType, err error) error {
	switch err.(type) {
	case StateError:
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
	features          *features
	monetaryPolicy    *monetaryPolicy
	ordersVolumes     *ordersVolumes
	accountsDataStor  *accountsDataStorage
	sponsoredAssets   *sponsoredAssets
	scriptsStorage    *scriptsStorage
	scriptsComplexity *scriptsComplexity
	invokeResults     *invokeResults
	stateHashes       *stateHashes
	hitSources        *hitSources
	calculateHashes   bool
}

func newBlockchainEntitiesStorage(hs *historyStorage, sets *settings.BlockchainSettings, rw *blockReadWriter, calcHashes bool) (*blockchainEntitiesStorage, error) {
	balances, err := newBalances(hs.db, hs, calcHashes)
	if err != nil {
		return nil, err
	}
	scriptsStorage, err := newScriptsStorage(hs, calcHashes)
	if err != nil {
		return nil, err
	}
	features := newFeatures(rw, hs.db, hs, sets, settings.FeaturesInfo)
	return &blockchainEntitiesStorage{
		hs,
		newAliases(hs.db, hs.dbBatch, hs, calcHashes),
		newAssets(hs.db, hs.dbBatch, hs),
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
		newHitSources(hs, rw),
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
			AccountScriptHash: s.scriptsStorage.accountScriptsHasher.stateHashAt(blockID),
			AssetScriptHash:   s.scriptsStorage.assetScriptsHasher.stateHashAt(blockID),
			LeaseBalanceHash:  s.balances.leaseHashAt(blockID),
			LeaseStatusHash:   s.leases.hasher.stateHashAt(blockID),
			SponsorshipHash:   s.sponsoredAssets.hasher.stateHashAt(blockID),
			AliasesHash:       s.aliases.hasher.stateHashAt(blockID),
		},
	}
	if err := sh.GenerateSumHash(prevHash); err != nil {
		return nil, err
	}
	if err := s.stateHashes.saveStateHash(sh, height); err != nil {
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

func (s *blockchainEntitiesStorage) handleStateHashes(blockchainHeight uint64, blockIds []proto.BlockID, initialisation bool) error {
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
	prevHash, err := s.stateHashes.stateHash(blockchainHeight, !initialisation)
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
	if err := s.sponsoredAssets.commitUncertain(blockID); err != nil {
		return err
	}
	return nil
}

func (s *blockchainEntitiesStorage) dropUncertain() {
	s.assets.dropUncertain()
	s.accountsDataStor.dropUncertain()
	s.scriptsStorage.dropUncertain()
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

func (s *blockchainEntitiesStorage) flush(initialisation bool) error {
	s.aliases.flush()
	if err := s.hs.flush(!initialisation); err != nil {
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

type newBlocks struct {
	binary    bool
	binBlocks [][]byte
	blocks    []*proto.Block
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

func (n *newBlocks) current() (*proto.Block, error) {
	if !n.binary {
		if n.curPos > len(n.blocks) || n.curPos < 1 {
			return nil, errors.New("bad current position")
		}
		return n.blocks[n.curPos-1], nil
	}
	if n.curPos > len(n.binBlocks) || n.curPos < 1 {
		return nil, errors.New("bad current position")
	}
	blockBytes := n.binBlocks[n.curPos-1]
	b := &proto.Block{}
	if err := n.unmarshalBlock(b, blockBytes); err != nil {
		return nil, err
	}
	return b, nil
}

func (n *newBlocks) reset() {
	n.binBlocks = nil
	n.blocks = nil
	n.curPos = 0
}

type stateManager struct {
	mu *sync.RWMutex // `mu` is used outside of state and returned in Mutex() function.

	// Last added block.
	lastBlock unsafe.Pointer

	genesis proto.Block
	stateDB *stateDB

	stor  *blockchainEntitiesStorage
	rw    *blockReadWriter
	peers *peerStorage

	// BlockchainSettings: general info about the blockchain type, constants etc.
	settings *settings.BlockchainSettings
	// ConsensusValidator: validator for block headers.
	cv *consensus.ConsensusValidator
	// Appender implements validation/diff management functionality.
	appender *txAppender
	atx      *addressTransactions

	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int

	newBlocks *newBlocks
}

func newStateManager(dataDir string, params StateParams, settings *settings.BlockchainSettings) (*stateManager, error) {
	err := validateSettings(settings)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0755); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create state directory: %v", err))
		}
	}
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create blocks directory: %v", err))
		}
	}
	// Initialize database.
	dbDir := filepath.Join(dataDir, keyvalueDir)
	zap.S().Info("Initializing state database, will take up to few minutes...")
	params.DbParams.BloomFilterParams.Store.WithPath(filepath.Join(blockStorageDir, "bloom"))
	db, err := keyvalue.NewKeyVal(dbDir, params.DbParams)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create db: %v", err))
	}
	zap.S().Info("Finished initializing database")
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create db batch: %v", err))
	}
	stateDB, err := newStateDB(db, dbBatch, params)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create stateDB: %v", err))
	}
	if err := checkCompatibility(stateDB, params); err != nil {
		return nil, wrapErr(IncompatibilityError, err)
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(
		blockStorageDir,
		params.OffsetLen,
		params.HeaderOffsetLen,
		stateDB,
		settings.AddressSchemeCharacter,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v", err))
	}
	stateDB.setRw(rw)
	hs, err := newHistoryStorage(db, dbBatch, stateDB)
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
	atx, err := newAddressTransactions(
		db,
		stateDB,
		rw,
		atxParams,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create address transactions storage: %v", err))
	}
	state := &stateManager{
		mu:                        &sync.RWMutex{},
		stateDB:                   stateDB,
		stor:                      stor,
		rw:                        rw,
		settings:                  settings,
		atx:                       atx,
		peers:                     newPeerStorage(db),
		verificationGoroutinesNum: params.VerificationGoroutinesNum,
		newBlocks:                 newNewBlocks(rw, settings),
	}
	// Set fields which depend on state.
	// Consensus validator is needed to check block headers.
	appender, err := newTxAppender(state, rw, stor, settings, stateDB, atx)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.appender = appender
	cv, err := consensus.NewConsensusValidator(state, params.Time)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.cv = cv
	// Handle genesis block.
	if err := state.handleGenesisBlock(settings.Genesis); err != nil {
		return nil, wrapErr(Other, err)
	}
	if err := state.loadLastBlock(); err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if err := state.checkProtobufActivation(); err != nil {
		return nil, wrapErr(Other, err)
	}
	return state, nil
}

func (s *stateManager) Mutex() *lock.RwMutex {
	return lock.NewRwMutex(s.mu)
}

func (s *stateManager) Peers() ([]proto.TCPAddr, error) {
	return s.peers.peers()
}

func (s *stateManager) setGenesisBlock(genesisBlock proto.Block) error {
	s.genesis = genesisBlock
	return nil
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
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)
	if err := s.addNewBlock(&s.genesis, nil, true, chans, 0); err != nil {
		return err
	}
	if err := s.stor.hitSources.saveHitSource(s.genesis.GenSignature, 1); err != nil {
		return err
	}
	close(chans.tasksChan)
	if err := s.appender.applyAllDiffs(true); err != nil {
		return err
	}
	if err := s.stor.prepareHashes(); err != nil {
		return err
	}
	if _, err := s.stor.putStateHash(nil, 1, s.genesis.BlockID()); err != nil {
		return err
	}
	verifyError := <-chans.errChan
	if verifyError != nil {
		return wrapErr(ValidationError, verifyError)
	}
	if err := s.flush(true); err != nil {
		return wrapErr(ModificationError, err)
	}
	s.reset()
	return nil
}

func (s *stateManager) applyPreactivatedFeatures(features []int16, blockID proto.BlockID) error {
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
	if err := s.flush(true); err != nil {
		return err
	}
	s.reset()
	return nil
}

func (s *stateManager) handleGenesisBlock(block proto.Block) error {
	height, err := s.Height()
	if err != nil {
		return err
	}

	if err := s.setGenesisBlock(block); err != nil {
		return err
	}
	// If the storage is new (data dir does not contain any data), genesis block must be applied.
	if height == 0 {
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockID()); err != nil {
			return err
		}
		if err := s.addGenesisBlock(); err != nil {
			return errors.Errorf("failed to apply/save genesis: %v", err)
		}
		// We apply preactivated features after genesis block, so they aren't active in genesis itself.
		if err := s.applyPreactivatedFeatures(s.settings.PreactivatedFeatures, block.BlockID()); err != nil {
			return errors.Errorf("failed to apply preactivated features: %v\n", err)
		}
	}
	return nil
}

func (s *stateManager) checkProtobufActivation() error {
	activated, err := s.stor.features.newestIsActivated(int16(settings.BlockV5))
	if err != nil {
		return errors.Errorf("newestIsActivated() failed: %v", err)
	}
	if !activated {
		return nil
	}
	s.rw.setProtobufActivated()
	return nil
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
	atomic.StorePointer(&s.lastBlock, unsafe.Pointer(lastBlock))
	return nil
}

func (s *stateManager) TopBlock() *proto.Block {
	return (*proto.Block)(atomic.LoadPointer(&s.lastBlock))
}

func (s *stateManager) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	var vrf []byte = nil
	if blockHeader.Version >= proto.ProtoBlockVersion {
		pos := &consensus.FairPosCalculatorV2{} // BlockV5 and FairPoSV2 are activated at the same time
		gsp := &consensus.VRFGenerationSignatureProvider{}
		hitSourceHeader, err := s.NewestHeaderByHeight(pos.HeightForHit(height))
		if err != nil {
			return nil, err
		}
		_, vrf, err = gsp.VerifyGenerationSignature(blockHeader.GenPublicKey, hitSourceHeader.GenSignature.Bytes(), blockHeader.GenSignature)
		if err != nil {
			return nil, err
		}
	}
	return vrf, nil
}

func (s *stateManager) Header(blockID proto.BlockID) (*proto.BlockHeader, error) {
	header, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return header, nil
}

func (s *stateManager) NewestHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.rw.newestBlockIDByHeight(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	header, err := s.rw.readNewestBlockHeader(blockID)
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

func (s *stateManager) newestAssetBalance(addr proto.Address, asset []byte) (uint64, error) {
	// Retrieve old balance from historyStorage.
	balance, err := s.stor.balances.newestAssetBalance(addr, asset, true)
	if err != nil {
		return 0, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := assetBalanceKey{address: addr, asset: asset}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if err == errNotFound {
		// If there is no diff, old balance is the newest.
		return balance, nil
	} else if err != nil {
		// Something weird happened.
		return 0, err
	}
	balance, err = diff.applyToAssetBalance(balance)
	if err != nil {
		return 0, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return balance, nil
}

func (s *stateManager) newestWavesBalanceProfile(addr proto.Address) (*balanceProfile, error) {
	// Retrieve the latest balance from historyStorage.
	profile, err := s.stor.balances.newestWavesBalance(addr, true)
	if err != nil {
		return nil, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := wavesBalanceKey{address: addr}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if err == errNotFound {
		// If there is no diff, old balance is the newest.
		return profile, nil
	} else if err != nil {
		// Something weird happened.
		return nil, err
	}
	newProfile, err := diff.applyTo(profile)
	if err != nil {
		return nil, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return newProfile, nil
}

func (s *stateManager) GeneratingBalance(account proto.Recipient) (uint64, error) {
	height, err := s.Height()
	if err != nil {
		return 0, errs.Extend(err, "failed to get height")
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	return s.EffectiveBalance(account, start, end)
}

func (s *stateManager) NewestGeneratingBalance(account proto.Recipient) (uint64, error) {
	height, err := s.NewestHeight()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	return s.NewestEffectiveBalance(account, start, end)
}

func (s *stateManager) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, errs.Extend(err, "failed convert recipient to address")
	}
	profile, err := s.stor.balances.wavesBalance(*addr, true)
	if err != nil {
		return nil, errs.Extend(err, "failed to get waves balance")
	}
	effective, err := profile.effectiveBalance()
	if err != nil {
		return nil, errs.Extend(err, "failed to get effective balance")
	}
	generating, err := s.GeneratingBalance(account)
	if err != nil {
		return nil, errs.Extend(err, "failed to get generating balance")
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

func (s *stateManager) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	profile, err := s.newestWavesBalanceProfile(*addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	effective, err := profile.effectiveBalance()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	generating, err := s.NewestGeneratingBalance(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
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

func (s *stateManager) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	if asset == nil {
		profile, err := s.newestWavesBalanceProfile(*addr)
		if err != nil {
			return 0, wrapErr(RetrievalError, err)
		}
		return profile.balance, nil
	}
	balance, err := s.newestAssetBalance(*addr, asset)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) AccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	if asset == nil {
		profile, err := s.stor.balances.wavesBalance(*addr, true)
		if err != nil {
			return 0, wrapErr(RetrievalError, err)
		}
		return profile.balance, nil
	}
	balance, err := s.stor.balances.assetBalance(*addr, asset, true)
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
	err = s.stor.monetaryPolicy.vote(block.RewardVote, height, activation, block.BlockID())
	if err != nil {
		return err
	}
	return nil
}

func (s *stateManager) addNewBlock(block, parent *proto.Block, initialisation bool, chans *verifierChans, height uint64) error {
	blockHeight := height + 1
	// Add score.
	if err := s.stor.scores.appendBlockScore(block, blockHeight, !initialisation); err != nil {
		return err
	}
	// Indicate new block for storage.
	if err := s.rw.startBlock(block.BlockID()); err != nil {
		return err
	}
	// Save block header to block storage.
	if err := s.rw.writeBlockHeader(&block.BlockHeader); err != nil {
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
		transactions:   transactions,
		chans:          chans,
		block:          &block.BlockHeader,
		parent:         parentHeader,
		height:         height,
		initialisation: initialisation,
	}
	// Check and perform block's transactions, create balance diffs, write transactions to storage.
	if err := s.appender.appendBlock(params); err != nil {
		return err
	}
	// Let block storage know that the current block is over.
	if err := s.rw.finishBlock(block.BlockID()); err != nil {
		return err
	}
	// Count features votes.
	if err := s.addFeaturesVotes(block); err != nil {
		return err
	}
	blockRewardActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockReward), height)
	// Count reward vote.
	if blockRewardActivated {
		err := s.addRewardVote(block, height)
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

func (s *stateManager) flush(initialisation bool) error {
	if err := s.rw.flush(); err != nil {
		return err
	}
	if err := s.stor.flush(initialisation); err != nil {
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
	rs, err := s.addBlocks(false)
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	s.newBlocks.setNew([]*proto.Block{block})
	rs, err := s.addBlocks(false)
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddNewBlocks(blockBytes [][]byte) error {
	s.newBlocks.setNewBinary(blockBytes)
	if _, err := s.addBlocks(false); err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddNewDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	s.newBlocks.setNew(blocks)
	lastBlock, err := s.addBlocks(false)
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return lastBlock, nil
}

func (s *stateManager) AddOldBlocks(blockBytes [][]byte) error {
	s.newBlocks.setNewBinary(blockBytes)
	if _, err := s.addBlocks(true); err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddOldDeserializedBlocks(blocks []*proto.Block) error {
	s.newBlocks.setNew(blocks)
	if _, err := s.addBlocks(true); err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) needToFinishVotingPeriod(blockchainHeight uint64) bool {
	nextBlockHeight := blockchainHeight + 1
	votingFinishHeight := (nextBlockHeight % s.settings.ActivationWindowSize(nextBlockHeight)) == 0
	return votingFinishHeight
}

func (s *stateManager) isBlockRewardTermOver(height uint64) (bool, error) {
	feature := int16(settings.BlockReward)
	activated := s.stor.features.newestIsActivatedAtHeight(feature, height)
	if activated {
		activation, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
		if err != nil {
			return false, err
		}
		_, end := blockRewardTermBoundaries(height, activation, s.settings.FunctionalitySettings)
		return end == height, nil
	}
	return false, nil
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

func (s *stateManager) needToCancelLeases(blockchainHeight uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to cancel leases in custom blockchains.
		return false, nil
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), blockchainHeight)
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(blockchainHeight)
	}
	switch blockchainHeight {
	case s.settings.ResetEffectiveBalanceAtHeight:
		return true, nil
	case s.settings.BlockVersion3AfterHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	case dataTxHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	default:
		return false, nil
	}
}

type heightActionParams struct {
	blockchainHeight uint64
	lastBlock        proto.BlockID
	nextBlock        proto.BlockID
	initialisation   bool
}

func (s *stateManager) blockchainHeightAction(params *heightActionParams) error {
	cancelLeases, err := s.needToCancelLeases(params.blockchainHeight)
	if err != nil {
		return err
	}
	if cancelLeases {
		if err := s.cancelLeases(params.blockchainHeight, params.lastBlock, params.initialisation); err != nil {
			return err
		}
	}
	resetStolenAliases, err := s.needToResetStolenAliases(params.blockchainHeight)
	if err != nil {
		return err
	}
	if resetStolenAliases {
		if err := s.stor.aliases.disableStolenAliases(); err != nil {
			return err
		}
	}
	if s.needToFinishVotingPeriod(params.blockchainHeight) {
		if err := s.finishVoting(params.blockchainHeight, params.lastBlock, params.initialisation); err != nil {
			return err
		}
		if err := s.stor.features.resetVotes(params.nextBlock); err != nil {
			return err
		}
	}
	termIsOver, err := s.isBlockRewardTermOver(params.blockchainHeight)
	if err != nil {
		return err
	}
	if termIsOver {
		if err := s.updateBlockReward(params.blockchainHeight, params.lastBlock, params.initialisation); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) finishVoting(height uint64, blockID proto.BlockID, initialisation bool) error {
	nextBlockHeight := height + 1
	if err := s.stor.features.finishVoting(nextBlockHeight, blockID); err != nil {
		return err
	}
	// Check if protobuf is now activated.
	// blockReadWriter will mark current offset as
	// start of protobuf-encoded objects.
	if err := s.checkProtobufActivation(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) updateBlockReward(height uint64, blockID proto.BlockID, initialisation bool) error {
	if err := s.stor.monetaryPolicy.updateBlockReward(height, blockID); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) cancelLeases(height uint64, blockID proto.BlockID, initialisation bool) error {
	// Move balance diffs from diffStorage to historyStorage.
	// It must be done before lease cancellation, because
	// lease cancellation iterates through historyStorage.
	if err := s.appender.moveChangesToHistoryStorage(initialisation); err != nil {
		return err
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), height)
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(height)
	}
	if height == s.settings.ResetEffectiveBalanceAtHeight {
		if err := s.stor.leases.cancelLeases(nil, blockID); err != nil {
			return err
		}
		if err := s.stor.balances.cancelAllLeases(blockID); err != nil {
			return err
		}
	} else if height == s.settings.BlockVersion3AfterHeight {
		overflowAddrs, err := s.stor.balances.cancelLeaseOverflows(blockID)
		if err != nil {
			return err
		}
		if err := s.stor.leases.cancelLeases(overflowAddrs, blockID); err != nil {
			return err
		}
	} else if dataTxActivated && height == dataTxHeight {
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelInvalidLeaseIns(leaseIns, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addBlocks(initialisation bool) (*proto.Block, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		// Reset in-memory storages and load last block in defer.
		s.reset()
		if err := s.loadLastBlock(); err != nil {
			zap.S().Fatalf("Failed to load last block: %v", err)
		}
		s.newBlocks.reset()
	}()

	blocksNumber := s.newBlocks.len()
	if blocksNumber == 0 {
		return nil, wrapErr(InvalidInputError, errors.New("no blocks provided"))
	}

	// Read some useful values for later.
	lastAppliedBlock, err := s.topBlock()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	zap.S().Debugf("StateManager: parent (top) block ID: %s, ts: %d", lastAppliedBlock.BlockID().String(), lastAppliedBlock.Timestamp)
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	headers := make([]proto.BlockHeader, blocksNumber)

	// Launch verifier that checks signatures of blocks and transactions.
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	var ids []proto.BlockID
	pos := 0
	for s.newBlocks.next() {
		curHeight := height + uint64(pos)
		block, err := s.newBlocks.current()
		if err != nil {
			return nil, wrapErr(DeserializationError, err)
		}
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockID()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// At some blockchain heights specific logic is performed.
		// This includes voting for features, block rewards and so on.
		params := &heightActionParams{
			blockchainHeight: curHeight,
			lastBlock:        lastAppliedBlock.BlockID(),
			nextBlock:        block.BlockID(),
			initialisation:   initialisation,
		}
		if err := s.blockchainHeightAction(params); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// Send block for signature verification, which works in separate goroutine.
		task := &verifyTask{
			taskType: verifyBlock,
			parentID: lastAppliedBlock.BlockID(),
			block:    block,
		}
		select {
		case verifyError := <-chans.errChan:
			return nil, verifyError
		case chans.tasksChan <- task:
		}
		// Save block to storage, check its transactions, create and save balance diffs for its transactions.
		if err := s.addNewBlock(block, lastAppliedBlock, initialisation, chans, curHeight); err != nil {
			return nil, err
		}
		headers[pos] = block.BlockHeader
		pos++
		ids = append(ids, block.BlockID())
		lastAppliedBlock = block
	}
	// Tasks chan can now be closed, since all the blocks and transactions have been already sent for verification.
	close(chans.tasksChan)
	// Apply all the balance diffs accumulated from this blocks batch.
	// This also validates diffs for negative balances.
	if err := s.appender.applyAllDiffs(initialisation); err != nil {
		return nil, err
	}
	// Retrieve and store state hashes for each of new blocks.
	if err := s.stor.handleStateHashes(height, ids, initialisation); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	// Validate consensus (i.e. that all of the new blocks were mined fairly).
	if err := s.cv.ValidateHeaders(headers[:pos], height); err != nil {
		return nil, wrapErr(ValidationError, err)
	}
	// Check the result of signatures verification.
	verifyError := <-chans.errChan
	if verifyError != nil {
		return nil, wrapErr(ValidationError, verifyError)
	}
	// After everything is validated, save all the changes to DB.
	if err := s.flush(initialisation); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	zap.S().Infof(
		"Height: %d; Block ID: %s, sig: %s, ts: %d",
		height+uint64(blocksNumber),
		lastAppliedBlock.BlockID().String(),
		base58.Encode(lastAppliedBlock.GenSignature),
		lastAppliedBlock.Timestamp,
	)
	return lastAppliedBlock, nil
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
	// Clear scripts cache.
	if err := s.stor.scriptsStorage.clear(); err != nil {
		zap.S().Fatalf("Failed to clear scripts cache after rollback: %v", err)
	}
	if err := s.loadLastBlock(); err != nil {
		zap.S().Fatalf("Failed to load last block after rollback: %v", err)
	}
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
	score, err := s.stor.scores.score(height, true)
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
		return nil, wrapErr(InvalidInputError,
			errors.Errorf("HitSourceAtHeight: height %d out of valid range [%d, %d]", height, 1, maxHeight))
	}
	hs, err := s.stor.hitSources.hitSource(height, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return hs, nil
}

func (s *stateManager) SaveHitSources(startHeight uint64, hitSources [][]byte) error {
	for i, hs := range hitSources {
		err := s.stor.hitSources.saveHitSource(hs, uint64(i+1)+startHeight)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) CurrentScore() (*big.Int, error) {
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.ScoreAtHeight(height)
}

func (s *stateManager) newestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	if recipient.Address == nil {
		return s.stor.aliases.newestAddrByAlias(recipient.Alias.Alias, true)
	}
	return recipient.Address, nil
}

func (s *stateManager) recipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	if recipient.Address == nil {
		return s.stor.aliases.addrByAlias(recipient.Alias.Alias, true)
	}
	return recipient.Address, nil
}

func (s *stateManager) EffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, errs.Extend(err, "failed convert recipient to address ")
	}
	effectiveBalance, err := s.stor.balances.minEffectiveBalanceInRange(*addr, startHeight, endHeight)
	if err != nil {
		return 0, errs.Extend(err, fmt.Sprintf("failed get min effective balance: startHeight: %d, endHeight: %d", startHeight, endHeight))
	}
	return effectiveBalance, nil
}

func (s *stateManager) NewestEffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	effectiveBalance, err := s.stor.balances.newestMinEffectiveBalanceInRange(*addr, startHeight, endHeight)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return effectiveBalance, nil
}

func (s *stateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	cp := *s.settings
	return &cp, nil
}

func (s *stateManager) SavePeers(peers []proto.TCPAddr) error {
	return s.peers.savePeers(peers)

}

func (s *stateManager) ResetValidationList() {
	s.reset()
	if err := s.stor.scriptsStorage.clear(); err != nil {
		zap.S().Fatalf("Failed to clear scripts cache after UTX validation: %v", err)
	}
}

// For UTX validation.
func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, v proto.BlockVersion, acceptFailed bool) error {
	if err := s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp, v, acceptFailed); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	addr, err := s.stor.aliases.newestAddrByAlias(alias.Alias, true)
	if err != nil {
		return proto.Address{}, wrapErr(RetrievalError, err)
	}
	return *addr, nil
}

func (s *stateManager) AddrByAlias(alias proto.Alias) (proto.Address, error) {
	addr, err := s.stor.aliases.addrByAlias(alias.Alias, true)
	if err != nil {
		return proto.Address{}, wrapErr(RetrievalError, err)
	}
	return *addr, nil
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
	activated, err := s.stor.features.isActivated(featureID)
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

// Accounts data storage.

func (s *stateManager) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestEntry(*addr, key, true)
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
	entries, err := s.stor.accountsDataStor.retrieveEntries(*addr, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entries, nil
}

func (s *stateManager) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveEntry(*addr, key, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestIntegerEntry(*addr, key, true)
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
	entry, err := s.stor.accountsDataStor.retrieveIntegerEntry(*addr, key, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBooleanEntry(*addr, key, true)
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
	entry, err := s.stor.accountsDataStor.retrieveBooleanEntry(*addr, key, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestStringEntry(*addr, key, true)
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
	entry, err := s.stor.accountsDataStor.retrieveStringEntry(*addr, key, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBinaryEntry(*addr, key, true)
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
	entry, err := s.stor.accountsDataStor.retrieveBinaryEntry(*addr, key, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	tx, _, err := s.rw.readNewestTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
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

func (s *stateManager) TransactionByIDWithStatus(id []byte) (proto.Transaction, bool, error) {
	tx, status, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, false, wrapErr(RetrievalError, err)
	}
	return tx, status, nil
}

func (s *stateManager) NewestTransactionHeightByID(id []byte) (uint64, error) {
	txHeight, err := s.rw.newestTransactionHeightByID(id)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return txHeight, nil
}

func (s *stateManager) TransactionHeightByID(id []byte) (uint64, error) {
	txHeight, err := s.rw.transactionHeightByID(id)
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

func (s *stateManager) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID, true)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) AssetIsSponsored(assetID crypto.Digest) (bool, error) {
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID, true)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	info, err := s.stor.assets.newestAssetInfo(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted := s.stor.scriptsStorage.newestIsSmartAsset(assetID, true)
	return &proto.AssetInfo{
		ID:              assetID,
		Quantity:        info.quantity.Uint64(),
		Decimals:        byte(info.decimals),
		Issuer:          issuer,
		IssuerPublicKey: info.issuer,
		Reissuable:      info.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

func (s *stateManager) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	ai, err := s.NewestAssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	info, err := s.stor.assets.newestAssetInfo(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	tx, err := s.NewestTransactionByID(assetID.Bytes())
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: tx,
	}
	isSponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.newestAssetCost(assetID, true)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.NewestAccountBalance(proto.NewRecipientFromAddress(ai.Issuer), nil)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted := s.stor.scriptsStorage.newestIsSmartAsset(assetID, true)
	if isScripted {
		scriptInfo, err := s.NewestScriptInfoByAsset(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.ScriptInfo = *scriptInfo
	}
	return res, nil
}

func (s *stateManager) AssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	info, err := s.stor.assets.assetInfo(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted, err := s.stor.scriptsStorage.isSmartAsset(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return &proto.AssetInfo{
		ID:              assetID,
		Quantity:        info.quantity.Uint64(),
		Decimals:        byte(info.decimals),
		Issuer:          issuer,
		IssuerPublicKey: info.issuer,
		Reissuable:      info.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

func (s *stateManager) FullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	ai, err := s.AssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	info, err := s.stor.assets.assetInfo(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	tx, err := s.TransactionByID(assetID.Bytes())
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: tx,
	}
	isSponsored, err := s.stor.sponsoredAssets.isSponsored(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.assetCost(assetID, true)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.AccountBalance(proto.NewRecipientFromAddress(ai.Issuer), nil)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted, err := s.stor.scriptsStorage.isSmartAsset(assetID, true)
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

func (s *stateManager) NFTList(account proto.Recipient, limit uint64, afterAssetID []byte) ([]*proto.FullAssetInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	fn := s.stor.assets.assetInfo
	nfts, err := s.stor.balances.nftList(*addr, limit, afterAssetID, fn)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	infos := make([]*proto.FullAssetInfo, len(nfts))
	for i, nft := range nfts {
		info, err := s.FullAssetInfo(nft)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		infos[i] = info
	}
	return infos, nil
}

func (s *stateManager) ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAddr(*addr, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	complexity, err := s.stor.scriptsComplexity.scriptComplexityByAddress(*addr, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	// TODO: switch complexity to DApp's complexity if verifier is incorrect for DApp.
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Base64:     text,
		Complexity: complexity.verifierComplexity,
	}, nil
}

func (s *stateManager) ScriptInfoByAsset(assetID crypto.Digest) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAsset(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	complexity, err := s.stor.scriptsComplexity.scriptComplexityByAsset(assetID, true)
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
		Base64:     text,
		Complexity: complexity.complexity,
	}, nil
}

func (s *stateManager) NewestScriptInfoByAsset(assetID crypto.Digest) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.newestScriptBytesByAsset(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	complexity, err := s.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID, true)
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
		Base64:     text,
		Complexity: complexity.complexity,
	}, nil
}

func (s *stateManager) IsActiveLeasing(leaseID crypto.Digest) (bool, error) {
	isActive, err := s.stor.leases.isActive(leaseID, true)
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
	res, err := s.stor.invokeResults.invokeResult(s.settings.AddressSchemeCharacter, invokeID, true)
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

func (s *stateManager) StateHashAtHeight(height uint64) (*proto.StateHash, error) {
	hasData, err := s.ProvidesStateHashes()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for state hashes"))
	}
	sh, err := s.stor.stateHashes.stateHash(height, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
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
	return s.atx.persist(true)
}

func (s *stateManager) ShouldPersistAddressTransactions() (bool, error) {
	return s.atx.shouldPersist()
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

package state

import (
	"context"
	"encoding/base64"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
	aliases, err := newAliases(hs.db, hs.dbBatch, hs, calcHashes)
	if err != nil {
		return nil, err
	}
	assets, err := newAssets(hs.db, hs.dbBatch, hs)
	if err != nil {
		return nil, err
	}
	blocksInfo, err := newBlocksInfo(hs.db, hs.dbBatch)
	if err != nil {
		return nil, err
	}
	balances, err := newBalances(hs.db, hs, calcHashes)
	if err != nil {
		return nil, err
	}
	features, err := newFeatures(rw, hs.db, hs, sets, settings.FeaturesInfo)
	if err != nil {
		return nil, err
	}
	monetaryPolicy, err := newMonetaryPolicy(hs.db, hs, sets)
	if err != nil {
		return nil, err
	}
	accountsDataStor, err := newAccountsDataStorage(hs.db, hs.dbBatch, hs, calcHashes)
	if err != nil {
		return nil, err
	}
	ordersVolumes, err := newOrdersVolumes(hs)
	if err != nil {
		return nil, err
	}
	sponsoredAssets, err := newSponsoredAssets(rw, features, hs, sets, calcHashes)
	if err != nil {
		return nil, err
	}
	scriptsStorage, err := newScriptsStorage(hs, calcHashes)
	if err != nil {
		return nil, err
	}
	scriptsComplexity, err := newScriptsComplexity(hs)
	if err != nil {
		return nil, err
	}
	invokeResults, err := newInvokeResults(hs)
	if err != nil {
		return nil, err
	}
	return &blockchainEntitiesStorage{
		hs,
		aliases,
		assets,
		newLeases(hs.db, hs, calcHashes),
		newScores(hs.db, hs.dbBatch),
		blocksInfo,
		balances,
		features,
		monetaryPolicy,
		ordersVolumes,
		accountsDataStor,
		sponsoredAssets,
		scriptsStorage,
		scriptsComplexity,
		invokeResults,
		newStateHashes(hs.db, hs.dbBatch),
		newHitSources(hs.db, hs.dbBatch),
		calcHashes,
	}, nil
}

func (s *blockchainEntitiesStorage) putStateHash(prevHash []byte, height uint64, blockID proto.BlockID) (*proto.StateHash, error) {
	sh := &proto.StateHash{
		BlockID:           blockID,
		WavesBalanceHash:  s.balances.wavesHashAt(blockID),
		AssetBalanceHash:  s.balances.assetsHashAt(blockID),
		DataEntryHash:     s.accountsDataStor.hasher.stateHashAt(blockID),
		AccountScriptHash: s.scriptsStorage.accountScriptsHasher.stateHashAt(blockID),
		AssetScriptHash:   s.scriptsStorage.assetScriptsHasher.stateHashAt(blockID),
		LeaseBalanceHash:  s.balances.leaseHashAt(blockID),
		LeaseStatusHash:   s.leases.hasher.stateHashAt(blockID),
		SponsorshipHash:   s.sponsoredAssets.hasher.stateHashAt(blockID),
		AliasesHash:       s.aliases.hasher.stateHashAt(blockID),
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

func (s *blockchainEntitiesStorage) handleStateHashes(blockchainHeight uint64, blockIds []proto.BlockID) error {
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
	prevHash, err := s.stateHashes.stateHash(blockchainHeight)
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

func (s *blockchainEntitiesStorage) rollback(newHeight, oldHeight uint64) error {
	if err := s.scores.rollback(newHeight, oldHeight); err != nil {
		return err
	}
	if err := s.hitSources.rollback(newHeight, oldHeight); err != nil {
		return err
	}
	if s.calculateHashes {
		if err := s.stateHashes.rollback(newHeight, oldHeight); err != nil {
			return err
		}
	}
	return nil
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

	// Miscellaneous/utility fields.
	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int
	// Indicates whether lease cancellations were performed.
	leasesCl0, leasesCl1, leasesCl2 bool
	// Indicates that stolen aliases were disabled.
	disabledStolenAliases bool
	// The height when last features voting took place.
	lastVotingHeight             uint64
	lastBlockRewardTermEndHeight uint64
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
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(
		blockStorageDir,
		params.OffsetLen,
		params.HeaderOffsetLen,
		db,
		dbBatch,
		settings.AddressSchemeCharacter,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v", err))
	}
	stateDB, err := newStateDB(db, dbBatch, rw, params)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create stateDB: %v", err))
	}
	if err := checkCompatibility(stateDB, params); err != nil {
		return nil, wrapErr(IncompatibilityError, err)
	}
	if err := stateDB.syncRw(); err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to sync block storage and DB: %v", err))
	}
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
	// Add score of genesis block.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	genesisScore, err := CalculateScore(s.genesis.BaseTarget)
	if err != nil {
		return err
	}
	if err := s.stor.scores.addScore(&big.Int{}, genesisScore, 1); err != nil {
		return err
	}
	if err := s.stor.hitSources.saveHitSource(s.genesis.GenSignature, 1); err != nil {
		return err
	}
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)
	if err := s.addNewBlock(&s.genesis, nil, true, chans, 0); err != nil {
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
	if err := s.reset(true); err != nil {
		return wrapErr(ModificationError, err)
	}
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
	if err := s.reset(true); err != nil {
		return err
	}
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
		// TODO: we apply preactivated features after genesis block, so they aren't active in genesis itself.
		// Probably it makes sense though, because genesis must be block version 1.
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
	height, err := s.rw.currentHeight()
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
		return proto.BlockID{}, wrapErr(InvalidInputError, errors.New("HeightToBlockID: height out of valid range"))
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return proto.BlockID{}, wrapErr(RetrievalError, err)
	}
	return blockID, nil
}

func (s *stateManager) newestAssetBalance(addr proto.Address, asset []byte) (uint64, error) {
	// Retrieve old balance.
	balance, err := s.stor.balances.assetBalance(addr, asset, true)
	if err != nil {
		return 0, err
	}
	// Retrieve latest balance diff as for the moment of this function call.
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

func (s *stateManager) newestWavesBalance(addr proto.Address) (uint64, error) {
	// Retrieve old balance.
	profile, err := s.stor.balances.wavesBalance(addr, true)
	if err != nil {
		return 0, err
	}
	// Retrieve latest balance diff as for the moment of this function call.
	key := wavesBalanceKey{address: addr}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if err == errNotFound {
		// If there is no diff, old balance is the newest.
		return profile.balance, nil
	} else if err != nil {
		// Something weird happened.
		return 0, err
	}
	newProfile, err := diff.applyTo(profile)
	if err != nil {
		return 0, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return newProfile.balance, nil
}

func (s *stateManager) GeneratingBalance(account proto.Recipient) (uint64, error) {
	height, err := s.Height()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	return s.EffectiveBalanceStable(account, start, end)
}

func (s *stateManager) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	profile, err := s.stor.balances.wavesBalance(*addr, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	effective, err := profile.effectiveBalance()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	generating, err := s.GeneratingBalance(account)
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
		balance, err := s.newestWavesBalance(*addr)
		if err != nil {
			return 0, wrapErr(RetrievalError, err)
		}
		return balance, nil
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
		approved, err := s.stor.features.isApproved(featureID)
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
	activation, err := s.ActivationHeight(int16(settings.BlockReward))
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
	blockRewardActivated, err := s.IsActiveAtHeight(int16(settings.BlockReward), height)
	if err != nil {
		return err
	}
	// Count reward vote.
	if blockRewardActivated {
		err := s.addRewardVote(block, height)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) reset(initialisation bool) error {
	s.rw.reset()
	s.stor.reset()
	s.stateDB.reset()
	s.appender.reset()
	if err := s.atx.reset(!initialisation); err != nil {
		return err
	}
	return nil
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

func (s *stateManager) undoBlockAddition() error {
	if err := s.reset(false); err != nil {
		return err
	}
	if err := s.stateDB.syncRw(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) AddBlock(block []byte) (*proto.Block, error) {
	b := &proto.Block{}
	err := b.UnmarshalBinary(block, s.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, err
	}
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	rs, err := s.addBlocks([]*proto.Block{b}, false)
	if err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) addBlock(block *proto.Block) (*proto.Block, error) {
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	rs, err := s.addBlocks([]*proto.Block{block}, false)
	if err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	return s.addBlock(block)
}

func (s *stateManager) AddNewBlocks(blockBytes [][]byte) error {
	var blocks []*proto.Block
	for _, bts := range blockBytes {
		block := &proto.Block{}
		err := block.UnmarshalBinary(bts, s.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	if _, err := s.addBlocks(blocks, false); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddNewDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	lastBlock, err := s.addBlocks(blocks, false)
	if err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return nil, err
	}
	return lastBlock, nil
}

func (s *stateManager) AddOldBlocks(blockBytes [][]byte) error {
	var blocks []*proto.Block
	for _, bts := range blockBytes {
		block := &proto.Block{}
		err := block.UnmarshalBinary(bts, s.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	if _, err := s.addBlocks(blocks, true); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddOldDeserializedBlocks(blocks []*proto.Block) error {
	// Make sure appender doesn't store any diffs from previous validations (e.g. UTX).
	s.appender.reset()
	if _, err := s.addBlocks(blocks, true); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not rollback to previous state after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) needToResetVotes(blockHeight uint64) bool {
	return (blockHeight % s.settings.ActivationWindowSize(blockHeight)) == 0
}

func (s *stateManager) needToFinishVotingPeriod(blockchainHeight uint64) bool {
	nextBlockHeight := blockchainHeight + 1
	votingFinishHeight := (nextBlockHeight % s.settings.ActivationWindowSize(nextBlockHeight)) == 0
	if votingFinishHeight {
		return s.lastVotingHeight != nextBlockHeight
	}
	return false
}

func (s *stateManager) isBlockRewardTermOver(height uint64) (bool, error) {
	feature := int16(settings.BlockReward)
	activated, err := s.IsActiveAtHeight(feature, height)
	if err != nil {
		return false, err
	}
	if activated {
		activation, err := s.ActivationHeight(int16(settings.BlockReward))
		if err != nil {
			return false, err
		}
		_, end := blockRewardTermBoundaries(height, activation, s.settings.FunctionalitySettings)
		if end == height {
			return s.lastBlockRewardTermEndHeight != height, nil
		}
	}
	return false, nil
}

func (s *stateManager) needToResetStolenAliases(height uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to reset stolen aliases in custom blockchains.
		return false, nil
	}
	dataTxActivated, err := s.IsActiveAtHeight(int16(settings.DataTransaction), height)
	if err != nil {
		return false, err
	}
	if dataTxActivated {
		dataTxHeight, err := s.ActivationHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		if height == dataTxHeight {
			return !s.disabledStolenAliases, nil
		}
	}
	return false, nil
}

func (s *stateManager) needToCancelLeases(curBlockHeight uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to cancel leases in custom blockchains.
		return false, nil
	}
	dataTxActivated, err := s.IsActiveAtHeight(int16(settings.DataTransaction), curBlockHeight)
	if err != nil {
		return false, err
	}
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.ApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(curBlockHeight)
	}
	switch curBlockHeight {
	case s.settings.ResetEffectiveBalanceAtHeight:
		return !s.leasesCl0, nil
	case s.settings.BlockVersion3AfterHeight:
		// Only needed for MainNet.
		return !s.leasesCl1 && (s.settings.Type == settings.MainNet), nil
	case dataTxHeight:
		// Only needed for MainNet.
		return !s.leasesCl2 && (s.settings.Type == settings.MainNet), nil
	default:
		return false, nil
	}
}

type breakerTask struct {
	// ID of latest block before performing task.
	blockID proto.BlockID
	// Indicates that the task to perform before calling addBlocks() is to reset stolen aliases.
	resetStolenAliases bool
	// Indicates that the task to perform before calling addBlocks() is to finish features voting period.
	finishVotingPeriod bool
	// Indication of the end of block reward term and block reward voting period.
	finishBlockRewardTerm bool
}

func (s *stateManager) needToBreakAddingBlocks(curHeight uint64, task *breakerTask) (bool, error) {
	resetStolenAliases, err := s.needToResetStolenAliases(curHeight)
	if err != nil {
		return false, err
	}
	if resetStolenAliases {
		task.resetStolenAliases = true
		return true, nil
	}
	if s.needToFinishVotingPeriod(curHeight) {
		task.finishVotingPeriod = true
		return true, nil
	}
	termIsOver, err := s.isBlockRewardTermOver(curHeight)
	if err != nil {
		return false, err
	}
	if termIsOver {
		task.finishBlockRewardTerm = true
	}
	return false, nil
}

func (s *stateManager) finishVoting(blockID proto.BlockID, initialisation bool) error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	nextBlockHeight := height + 1
	if err := s.stor.features.finishVoting(nextBlockHeight, blockID); err != nil {
		return err
	}
	s.lastVotingHeight = nextBlockHeight
	// Check if protobuf is now activated.
	// blockReadWriter will mark current offset as
	// start of protobuf-encoded objects.
	if err := s.checkProtobufActivation(); err != nil {
		return err
	}
	if err := s.flush(initialisation); err != nil {
		return err
	}
	if err := s.reset(initialisation); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) updateBlockReward(blockID proto.BlockID, initialisation bool) error {
	h, err := s.Height()
	if err != nil {
		return err
	}
	if err := s.stor.monetaryPolicy.updateBlockReward(h, blockID); err != nil {
		return err
	}
	s.lastBlockRewardTermEndHeight = h
	if err := s.flush(initialisation); err != nil {
		return err
	}
	if err := s.reset(initialisation); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) cancelLeases(curBlockHeight uint64, blockID proto.BlockID) error {
	dataTxActivated, err := s.IsActiveAtHeight(int16(settings.DataTransaction), curBlockHeight)
	if err != nil {
		return err
	}
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.ApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(curBlockHeight)
	}
	if curBlockHeight == s.settings.ResetEffectiveBalanceAtHeight {
		if err := s.stor.leases.cancelLeases(nil, blockID); err != nil {
			return err
		}
		if err := s.stor.balances.cancelAllLeases(blockID); err != nil {
			return err
		}
		s.leasesCl0 = true
	} else if curBlockHeight == s.settings.BlockVersion3AfterHeight {
		overflowAddrs, err := s.stor.balances.cancelLeaseOverflows(blockID)
		if err != nil {
			return err
		}
		if err := s.stor.leases.cancelLeases(overflowAddrs, blockID); err != nil {
			return err
		}
		s.leasesCl1 = true
	} else if dataTxActivated && curBlockHeight == dataTxHeight {
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelInvalidLeaseIns(leaseIns, blockID); err != nil {
			return err
		}
		s.leasesCl2 = true
	}
	return nil
}

func (s *stateManager) handleBreak(blocksToFinish []*proto.Block, initialisation bool, task *breakerTask) (*proto.Block, error) {
	if task == nil {
		return nil, wrapErr(Other, errors.New("handleBreak received empty task"))
	}
	if task.finishVotingPeriod {
		if err := s.finishVoting(task.blockID, initialisation); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
	}
	if task.finishBlockRewardTerm {
		if err := s.updateBlockReward(task.blockID, initialisation); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
	}
	if task.resetStolenAliases {
		// Need to reset stolen aliases due to bugs in historical blockchain.
		if err := s.stor.aliases.disableStolenAliases(); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		s.disabledStolenAliases = true
	}
	if len(blocksToFinish) == 0 {
		return s.TopBlock(), nil
	}
	return s.addBlocks(blocksToFinish, initialisation)
}

func (s *stateManager) addBlocks(blocks []*proto.Block, initialisation bool) (*proto.Block, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	blocksNumber := len(blocks)
	if blocksNumber == 0 {
		return nil, wrapErr(InvalidInputError, errors.New("no blocks provided"))
	}

	// Read some useful values for later.
	parent, err := s.topBlock()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	zap.S().Debugf("StateManager: parent (top) block ID: %s", parent.BlockID().String())
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	prevScore, err := s.stor.scores.score(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	headers := make([]proto.BlockHeader, blocksNumber)

	// Some 'events', like finish of voting periods or cancelling invalid leases, happen (or happened)
	// at defined height of the blockchain.
	// When such events occur inside of the blocks batch, this batch must be splitted, so the event
	// can be performed with consistent database state, with all the recent changes being saved to disk.
	// After performing the event, addBlocks() calls itself with the rest of the blocks batch.
	// blocksToFinish stores these blocks, breakerInfo specifies type of the event.
	var blocksToFinish []*proto.Block
	breakerInfo := &breakerTask{blockID: parent.BlockID()}

	// Launch verifier that checks signatures of blocks and transactions.
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	var lastBlock *proto.Block
	var ids []proto.BlockID
	needToCancelLeases := false
	curBlockHeight := height + 1
	for i, block := range blocks {
		curHeight := height + uint64(i)
		curBlockHeight = curHeight + 1
		breakAdding, err := s.needToBreakAddingBlocks(curHeight, breakerInfo)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		if breakAdding {
			// Need to break at this height, so we split block batch in order to cancel and finish with the rest blocks after.
			blocksToFinish = blocks[i:]
			break
		}
		breakerInfo.blockID = block.BlockID()
		// Send block for signature verification, which works in separate goroutine.
		task := &verifyTask{
			taskType: verifyBlock,
			parentID: parent.BlockID(),
			block:    block,
		}
		select {
		case verifyError := <-chans.errChan:
			return nil, wrapErr(ValidationError, verifyError)
		case chans.tasksChan <- task:
		}
		lastBlock = block
		// Add score.
		score, err := CalculateScore(block.BaseTarget)
		if err != nil {
			return nil, wrapErr(Other, err)
		}
		if err := s.stor.scores.addScore(prevScore, score, curBlockHeight); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		prevScore = score
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockID()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		if s.needToResetVotes(curBlockHeight) {
			// When next voting period starts, we need to put 0 as votes number
			// for all features at first (current) block.
			// This is not handled as breaker task on purpose:
			// featureVotes() operates with fresh records, so we do not need to flush() votes.
			if err := s.stor.features.resetVotes(block.BlockID()); err != nil {
				return nil, wrapErr(ModificationError, err)
			}
		}
		// Save block to storage, check its transactions, create and save balance diffs for its transactions.
		if err := s.addNewBlock(block, parent, initialisation, chans, curHeight); err != nil {
			return nil, wrapErr(TxValidationError, err)
		}
		headers[i] = block.BlockHeader
		parent = block
		ids = append(ids, block.BlockID())
		needToCancelLeases, err = s.needToCancelLeases(curBlockHeight)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		if needToCancelLeases {
			blocksToFinish = blocks[i+1:]
			break
		}
	}
	// Tasks chan can now be closed, since all the blocks and transactions have been already sent for verification.
	close(chans.tasksChan)
	// Apply all the balance diffs accumulated from this blocks batch.
	// This also validates diffs for negative balances.
	if err := s.appender.applyAllDiffs(initialisation); err != nil {
		return nil, wrapErr(TxValidationError, err)
	}
	if needToCancelLeases {
		// Need to cancel leases due to bugs in historical blockchain.
		if err := s.cancelLeases(curBlockHeight, lastBlock.BlockID()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
	}
	// Retrieve and store state hashes for each of new blocks.
	if err := s.stor.handleStateHashes(height, ids); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	// Validate consensus (i.e. that all of the new blocks were mined fairly).
	if err := s.cv.ValidateHeaders(headers[:len(headers)-len(blocksToFinish)], height); err != nil {
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
	// Reset in-memory storages.
	if err := s.reset(initialisation); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	if err := s.loadLastBlock(); err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	// Check if we need to perform some event and call addBlocks() again.
	if blocksToFinish != nil {
		return s.handleBreak(blocksToFinish, initialisation, breakerInfo)
	}
	if lastBlock != nil {
		zap.S().Infof("Height: %d; Block ID: %s", height+uint64(blocksNumber), lastBlock.BlockID().String())
	}
	return lastBlock, nil
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
	if err := s.RollbackTo(blockID); err != nil {
		return wrapErr(RollbackError, err)
	}
	return nil
}

func (s *stateManager) rollbackToImpl(removalEdge proto.BlockID) error {
	// TODO: this is not really atomic.
	if err := s.checkRollbackInput(removalEdge); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	curHeight, err := s.rw.currentHeight()
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	for height := curHeight; height > 0; height-- {
		blockID, err := s.rw.blockIDByHeight(height)
		if err != nil {
			return wrapErr(RetrievalError, err)
		}
		if blockID == removalEdge {
			break
		}
		if err := s.stateDB.rollbackBlock(blockID); err != nil {
			return wrapErr(RollbackError, err)
		}
		if err := s.stor.blocksInfo.rollback(blockID); err != nil {
			return wrapErr(RollbackError, err)
		}
	}
	// Remove blocks from block storage.
	if err := s.rw.rollback(removalEdge, true); err != nil {
		return wrapErr(RollbackError, err)
	}
	// Rollback entities stored by block height.
	newHeight, err := s.Height()
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	oldHeight := curHeight + 1
	if err := s.stor.rollback(newHeight, oldHeight); err != nil {
		return wrapErr(RollbackError, err)
	}
	// Clear scripts cache.
	if err := s.stor.scriptsStorage.clear(); err != nil {
		return wrapErr(RollbackError, err)
	}
	if err := s.loadLastBlock(); err != nil {
		return wrapErr(RetrievalError, err)
	}
	return nil
}

func (s *stateManager) RollbackTo(removalEdge proto.BlockID) error {
	if err := s.rollbackToImpl(removalEdge); err != nil {
		if err1 := s.stateDB.syncRw(); err1 != nil {
			zap.S().Fatalf("Failed to rollback and can not sync state components after failure: %v", err1)
		}
		return err
	}
	return nil
}

func (s *stateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.New("ScoreAtHeight: height out of valid range"))
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
		return nil, wrapErr(InvalidInputError,
			errors.Errorf("HitSourceAtHeight: height %d out of valid range [%d, %d]", height, 1, maxHeight))
	}
	hs, err := s.stor.hitSources.hitSource(height)
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

func (s *stateManager) EffectiveBalanceStable(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	effectiveBalance, err := s.stor.balances.minEffectiveBalanceInRangeStable(*addr, startHeight, endHeight)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return effectiveBalance, nil
}

func (s *stateManager) EffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	effectiveBalance, err := s.stor.balances.minEffectiveBalanceInRange(*addr, startHeight, endHeight)
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
	s.appender.resetValidationList()
}

// For UTX validation.
func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, v proto.BlockVersion, vrf []byte, acceptFailed bool) error {
	if err := s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp, v, vrf, acceptFailed); err != nil {
		return wrapErr(TxValidationError, err)
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
	votesNum, err := s.stor.features.featureVotesStable(featureID)
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
	//TODO: use transaction failure status
	tx, _, err := s.rw.readNewestTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return tx, nil
}

func (s *stateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	//TODO: use transaction failure status
	tx, _, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return tx, nil
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
	scripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID, true)
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
	isScripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID, true)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
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
	sh, err := s.stor.stateHashes.stateHash(height)
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

func (s *stateManager) PersisAddressTransactions() error {
	return s.atx.persist(true, true)
}

func (s *stateManager) ShouldPersisAddressTransactions() (bool, error) {
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

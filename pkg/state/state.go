package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
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
}

func newBlockchainEntitiesStorage(hs *historyStorage, sets *settings.BlockchainSettings, rw *blockReadWriter) (*blockchainEntitiesStorage, error) {
	aliases, err := newAliases(hs.db, hs.dbBatch, hs)
	if err != nil {
		return nil, err
	}
	assets, err := newAssets(hs.db, hs.dbBatch, hs)
	if err != nil {
		return nil, err
	}
	leases, err := newLeases(hs.db, hs)
	if err != nil {
		return nil, err
	}
	scores, err := newScores(hs.db, hs.dbBatch)
	if err != nil {
		return nil, err
	}
	blocksInfo, err := newBlocksInfo(hs.db, hs.dbBatch)
	if err != nil {
		return nil, err
	}
	balances, err := newBalances(hs.db, hs)
	if err != nil {
		return nil, err
	}
	features, err := newFeatures(hs.db, hs, sets, settings.FeaturesInfo)
	if err != nil {
		return nil, err
	}
	monetaryPolicy, err := newMonetaryPolicy(hs.db, hs, sets)
	if err != nil {
		return nil, err
	}
	accountsDataStor, err := newAccountsDataStorage(hs.db, hs.dbBatch, hs)
	if err != nil {
		return nil, err
	}
	ordersVolumes, err := newOrdersVolumes(hs)
	if err != nil {
		return nil, err
	}
	sponsoredAssets, err := newSponsoredAssets(rw, features, hs, sets)
	if err != nil {
		return nil, err
	}
	scriptsStorage, err := newScriptsStorage(hs)
	if err != nil {
		return nil, err
	}
	scriptsComplexity, err := newScriptsComplexity(hs)
	if err != nil {
		return nil, err
	}
	invokeResults, err := newInvokeResults(hs, aliases)
	if err != nil {
		return nil, err
	}
	return &blockchainEntitiesStorage{
		hs,
		aliases,
		assets,
		leases,
		scores,
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
	}, nil
}

func (s *blockchainEntitiesStorage) reset() {
	s.hs.reset()
	s.assets.reset()
	s.accountsDataStor.reset()
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

func checkCompatibility(stateDB *stateDB, extendedApi bool) error {
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
	if extendedApi != hasDataForExtendedApi {
		return errors.Errorf("extended API incompatibility: state stores: %v; want: %v", hasDataForExtendedApi, extendedApi)
	}
	return nil
}

type stateManager struct {
	mu *sync.RWMutex // `mu` is used outside of state and returned in Mutex() function.

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
	rw, err := newBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db, dbBatch)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v", err))
	}
	stateDB, err := newStateDB(db, dbBatch, rw, params.StoreExtendedApiData)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create stateDB: %v", err))
	}
	if err := checkCompatibility(stateDB, params.StoreExtendedApiData); err != nil {
		return nil, wrapErr(IncompatibilityError, err)
	}
	if err := stateDB.syncRw(); err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to sync block storage and DB: %v", err))
	}
	hs, err := newHistoryStorage(db, dbBatch, stateDB)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create history storage: %v", err))
	}
	stor, err := newBlockchainEntitiesStorage(hs, settings, rw)
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
	cv, err := consensus.NewConsensusValidator(state)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.cv = cv
	// Handle genesis block.
	if err := state.handleGenesisBlock(settings.Genesis); err != nil {
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
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum)
	if err := s.addNewBlock(&s.genesis, nil, true, chans, 0); err != nil {
		return err
	}
	close(chans.tasksChan)
	if err := s.appender.applyAllDiffs(true); err != nil {
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

func (s *stateManager) applyPreactivatedFeatures(features []int16, blockID crypto.Signature) error {
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
		if err := s.stateDB.addBlock(block.BlockSignature); err != nil {
			return err
		}
		if err := s.addGenesisBlock(); err != nil {
			return errors.Errorf("failed to apply/save genesis: %v", err)
		}
		// TODO: we apply preactivated features after genesis block, so they aren't active in genesis itself.
		// Probably it makes sense though, because genesis must be block version 1.
		if err := s.applyPreactivatedFeatures(s.settings.PreactivatedFeatures, block.BlockSignature); err != nil {
			return errors.Errorf("failed to apply preactivated features: %v\n", err)
		}
	}
	return nil
}

func (s *stateManager) Header(blockID crypto.Signature) (*proto.BlockHeader, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		if err == keyvalue.ErrNotFound {
			return nil, wrapErr(NotFoundError, err)
		}
		return nil, wrapErr(RetrievalError, err)
	}
	var header proto.BlockHeader
	if err := header.UnmarshalHeaderFromBinary(headerBytes); err != nil {
		return nil, wrapErr(DeserializationError, err)
	}
	return &header, nil
}

func (s *stateManager) HeaderBytes(blockID crypto.Signature) ([]byte, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return headerBytes, nil
}

func (s *stateManager) NewestHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.rw.newestBlockIDByHeight(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	headerBytes, err := s.rw.readNewestBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	var header proto.BlockHeader
	if err := header.UnmarshalHeaderFromBinary(headerBytes); err != nil {
		return nil, wrapErr(DeserializationError, err)
	}
	return &header, nil
}

func (s *stateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Header(blockID)
}

func (s *stateManager) HeaderBytesByHeight(height uint64) ([]byte, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.HeaderBytes(blockID)
}

func (s *stateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	header, err := s.Header(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	transactions, err := s.rw.readTransactionsBlock(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	block := proto.Block{BlockHeader: *header}
	block.Transactions = proto.NewReprFromBytes(transactions, block.TransactionCount)
	return &block, nil
}

func (s *stateManager) BlockBytes(blockID crypto.Signature) ([]byte, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	transactions, err := s.rw.readTransactionsBlock(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	blockBytes, err := proto.AppendHeaderBytesToTransactions(headerBytes, transactions)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return blockBytes, nil
}

func (s *stateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Block(blockID)
}

func (s *stateManager) BlockBytesByHeight(height uint64) ([]byte, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.BlockBytes(blockID)
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

func (s *stateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	height, err := s.rw.heightByBlockID(blockID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return crypto.Signature{}, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return crypto.Signature{}, wrapErr(InvalidInputError, errors.New("HeightToBlockID: height out of valid range"))
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return crypto.Signature{}, wrapErr(RetrievalError, err)
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
		if err := s.stor.features.addVote(featureID, block.BlockSignature); err != nil {
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
	err = s.stor.monetaryPolicy.vote(block.RewardVote, height, activation, block.BlockSignature)
	if err != nil {
		return err
	}
	return nil
}

func (s *stateManager) addNewBlock(block, parent *proto.Block, initialisation bool, chans *verifierChans, height uint64) error {
	// Check the block version.
	blockRewardActivated, err := s.IsActivated(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	if blockRewardActivated && block.Version != proto.RewardBlockVersion {
		return errors.Errorf("invalid block version %d after activation of BlockReward feature", block.Version)
	}
	// Indicate new block for storage.
	if err := s.rw.startBlock(block.BlockSignature); err != nil {
		return err
	}
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		return err
	}
	// Save block header to block storage.
	if err := s.rw.writeBlockHeader(block.BlockSignature, headerBytes); err != nil {
		return err
	}
	transactions, err := block.Transactions.Transactions()
	if err != nil {
		return err
	}
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
	if err := s.rw.finishBlock(block.BlockSignature); err != nil {
		return err
	}
	// Count features votes.
	if err := s.addFeaturesVotes(block); err != nil {
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
	err := b.UnmarshalBinary(block)
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
		err := block.UnmarshalBinary(bts)
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

func (s *stateManager) blocksToBinary(blocks []*proto.Block) ([][]byte, error) {
	var blocksBytes [][]byte
	for _, block := range blocks {
		blockBytes, err := block.MarshalBinary()
		if err != nil {
			return nil, err
		}
		blocksBytes = append(blocksBytes, blockBytes)
	}
	return blocksBytes, nil
}

func (s *stateManager) AddNewDeserializedBlocks(blocks []*proto.Block) error {
	blocksBytes, err := s.blocksToBinary(blocks)
	if err != nil {
		return wrapErr(SerializationError, err)
	}
	return s.AddNewBlocks(blocksBytes)
}

func (s *stateManager) AddOldBlocks(blockBytes [][]byte) error {
	var blocks []*proto.Block
	for _, bts := range blockBytes {
		block := &proto.Block{}
		err := block.UnmarshalBinary(bts)
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
	blocksBytes, err := s.blocksToBinary(blocks)
	if err != nil {
		return wrapErr(SerializationError, err)
	}
	return s.AddOldBlocks(blocksBytes)
}

func (s *stateManager) needToFinishVotingPeriod(height uint64) bool {
	votingFinishHeight := (height % s.settings.ActivationWindowSize(height)) == 0
	if votingFinishHeight {
		return s.lastVotingHeight != height
	}
	return false
}

func (s *stateManager) isBlockRewardTermOver(height uint64) (bool, error) {
	feature := int16(settings.BlockReward)
	activated, err := s.IsActivated(feature)
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
	dataTxActivated, err := s.IsActivated(int16(settings.DataTransaction))
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

func (s *stateManager) needToCancelLeases(height uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to cancel leases in custom blockchains.
		return false, nil
	}
	dataTxActivated, err := s.IsActivated(int16(settings.DataTransaction))
	if err != nil {
		return false, err
	}
	dataTxHeight := uint64(0)
	if dataTxActivated {
		dataTxHeight, err = s.ActivationHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
	}
	switch height {
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
	blockID crypto.Signature
	// Indicates that the task to perform before calling addBlocks() is to cancel leases.
	cancelLeases bool
	// Indicates that the task to perform before calling addBlocks() is to reset stolen aliases.
	resetStolenAliases bool
	// Indicates that the task to perform before calling addBlocks() is to finish features voting period.
	finishVotingPeriod bool
	// Indication of the end of block reward term and block reward voting period.
	finishBlockRewardTerm bool
}

func (s *stateManager) needToBreakAddingBlocks(curHeight uint64, task *breakerTask) (bool, error) {
	cancelLeases, err := s.needToCancelLeases(curHeight)
	if err != nil {
		return false, err
	}
	if cancelLeases {
		task.cancelLeases = true
		return true, nil
	}
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

func (s *stateManager) finishVoting(blockID crypto.Signature, initialisation bool) error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	if err := s.stor.features.finishVoting(height, blockID); err != nil {
		return err
	}
	s.lastVotingHeight = height
	if err := s.flush(initialisation); err != nil {
		return err
	}
	if err := s.reset(initialisation); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) updateBlockReward(blockID crypto.Signature, initialisation bool) error {
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

func (s *stateManager) cancelLeases(blockID crypto.Signature) error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	dataTxActivated, err := s.IsActivated(int16(settings.DataTransaction))
	if err != nil {
		return err
	}
	dataTxHeight := uint64(0)
	if dataTxActivated {
		dataTxHeight, err = s.ActivationHeight(int16(settings.DataTransaction))
		if err != nil {
			return err
		}
	}
	if height == s.settings.ResetEffectiveBalanceAtHeight {
		if err := s.stor.leases.cancelLeases(nil, blockID); err != nil {
			return err
		}
		if err := s.stor.balances.cancelAllLeases(blockID); err != nil {
			return err
		}
		s.leasesCl0 = true
	} else if height == s.settings.BlockVersion3AfterHeight {
		overflowAddrs, err := s.stor.balances.cancelLeaseOverflows(blockID)
		if err != nil {
			return err
		}
		if err := s.stor.leases.cancelLeases(overflowAddrs, blockID); err != nil {
			return err
		}
		s.leasesCl1 = true
	} else if dataTxActivated && height == dataTxHeight {
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelInvalidLeaseIns(leaseIns, blockID); err != nil {
			return err
		}
		s.leasesCl2 = true
	}
	if err := s.flush(true); err != nil {
		return err
	}
	if err := s.reset(true); err != nil {
		return err
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
	if task.cancelLeases {
		// Need to cancel leases due to bugs in historical blockchain.
		if err := s.cancelLeases(task.blockID); err != nil {
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
	zap.S().Debugf("StateManager: parent (top) block signature: %s", parent.BlockSignature.String())
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
	breakerInfo := &breakerTask{blockID: parent.BlockSignature}

	// Launch verifier that checks signatures of blocks and transactions.
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum)

	var lastBlock *proto.Block
	for i, block := range blocks {
		curHeight := height + uint64(i)
		breakAdding, err := s.needToBreakAddingBlocks(curHeight, breakerInfo)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		if breakAdding {
			// Need to break at this height, so we split block batch in order to cancel and finish with the rest blocks after.
			blocksToFinish = blocks[i:]
			break
		}
		breakerInfo.blockID = block.BlockSignature
		blockBytes, err := blockToBytes(block)
		if err != nil {
			return nil, err
		}
		// Send block for signature verification, which works in separate goroutine.
		task := &verifyTask{
			taskType:   verifyBlock,
			parentSig:  parent.BlockSignature,
			block:      block,
			blockBytes: blockBytes[:len(blockBytes)-crypto.SignatureSize],
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
		if err := s.stor.scores.addScore(prevScore, score, curHeight+1); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		prevScore = score
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockSignature); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// Save block to storage, check its transactions, create and save balance diffs for its transactions.
		if err := s.addNewBlock(block, parent, initialisation, chans, curHeight); err != nil {
			return nil, wrapErr(TxValidationError, err)
		}
		headers[i] = block.BlockHeader
		parent = block
	}
	// Tasks chan can now be closed, since all the blocks and transactions have been already sent for verification.
	close(chans.tasksChan)
	// Apply all the balance diffs accumulated from this blocks batch.
	// This also validates diffs for negative balances.
	if err := s.appender.applyAllDiffs(initialisation); err != nil {
		return nil, wrapErr(TxValidationError, err)
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
	// Check if we need to perform some event and call addBlocks() again.
	if blocksToFinish != nil {
		return s.handleBreak(blocksToFinish, initialisation, breakerInfo)
	}
	zap.S().Infof("State: blocks to height %d added, block sig: %s", height+uint64(blocksNumber), lastBlock.BlockSignature)
	return lastBlock, nil
}

func (s *stateManager) checkRollbackInput(blockID crypto.Signature) error {
	height, err := s.BlockIDToHeight(blockID)
	if err != nil {
		return err
	}
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

func (s *stateManager) RollbackToHeight(height uint64) error {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	if err := s.checkRollbackInput(blockID); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	if err := s.RollbackTo(blockID); err != nil {
		return wrapErr(RollbackError, err)
	}
	return nil
}

func (s *stateManager) rollbackToImpl(removalEdge crypto.Signature) error {
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
		if bytes.Equal(blockID[:], removalEdge[:]) {
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
	// Remove scores of deleted blocks.
	newHeight, err := s.Height()
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	oldHeight := curHeight + 1
	if err := s.stor.scores.rollback(newHeight, oldHeight); err != nil {
		return wrapErr(RollbackError, err)
	}
	// Clear scripts cache.
	if err := s.stor.scriptsStorage.clear(); err != nil {
		return wrapErr(RollbackError, err)
	}
	return nil
}

func (s *stateManager) RollbackTo(removalEdge crypto.Signature) error {
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
	return s.settings, nil
}

func (s *stateManager) SavePeers(peers []proto.TCPAddr) error {
	return s.peers.savePeers(peers)

}

func (s *stateManager) ResetValidationList() {
	s.appender.resetValidationList()
}

// For UTX validation.
func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, v proto.BlockVersion) error {
	if err := s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp, v); err != nil {
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

func (s *stateManager) IsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	h, err := s.stor.features.activationHeight(featureID)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return h >= height, nil
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
	txBytes, err := s.rw.readNewestTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	tx, err := proto.BytesToTransaction(txBytes)
	if err != nil {
		return nil, wrapErr(DeserializationError, err)
	}
	return tx, nil
}

func (s *stateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	txBytes, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	tx, err := proto.BytesToTransaction(txBytes)
	if err != nil {
		return nil, wrapErr(DeserializationError, err)
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
	// TODO: switch complexity to DApp's complexity if verifier is incorrect for DApp.
	return &proto.ScriptInfo{
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
	return &proto.ScriptInfo{
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
	res, err := s.stor.invokeResults.invokeResult(invokeID, true)
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

func (s *stateManager) IsNotFound(err error) bool {
	return IsNotFound(err)
}

func (s *stateManager) StartProvidingExtendedApi() error {
	if err := s.atx.startProvidingData(); err != nil {
		return wrapErr(ModificationError, err)
	}
	return nil
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

func blockToBytes(block *proto.Block) ([]byte, error) {
	buf := bytebufferpool.Get()
	_, err := block.WriteTo(buf)
	if err != nil {
		bytebufferpool.Put(buf)
		return nil, err
	}
	blockBytes := make([]byte, len(buf.B))
	copy(blockBytes, buf.B)
	bytebufferpool.Put(buf)
	return blockBytes, nil
}

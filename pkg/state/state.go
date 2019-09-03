package state

import (
	"bytes"
	"context"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

const (
	rollbackMaxBlocks = 2000
	blocksStorDir     = "blocks_storage"
	keyvalueDir       = "key_value"
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

func NewNotFoundError(err error) error {
	return wrapErr(NotFoundError, err)
}

type blockchainEntitiesStorage struct {
	hs               *historyStorage
	aliases          *aliases
	assets           *assets
	leases           *leases
	scores           *scores
	blocksInfo       *blocksInfo
	balances         *balances
	features         *features
	accountsDataStor *accountsDataStorage
	sponsoredAssets  *sponsoredAssets
}

func newBlockchainEntitiesStorage(hs *historyStorage, stateDB *stateDB, sets *settings.BlockchainSettings) (*blockchainEntitiesStorage, error) {
	aliases, err := newAliases(hs.db, hs.dbBatch, stateDB, hs)
	if err != nil {
		return nil, err
	}
	assets, err := newAssets(hs.db, hs.dbBatch, stateDB, hs)
	if err != nil {
		return nil, err
	}
	leases, err := newLeases(hs.db, stateDB, hs)
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
	balances, err := newBalances(hs.db, stateDB, hs)
	if err != nil {
		return nil, err
	}
	features, err := newFeatures(hs.db, hs.dbBatch, hs, stateDB, sets, settings.FeaturesInfo)
	if err != nil {
		return nil, err
	}
	accountsDataStor, err := newAccountsDataStorage(hs.db, hs.dbBatch, hs.rw, stateDB)
	if err != nil {
		return nil, err
	}
	sponsoredAssets, err := newSponsoredAssets(hs.rw, features, stateDB, hs, sets)
	if err != nil {
		return nil, err
	}
	return &blockchainEntitiesStorage{hs, aliases, assets, leases, scores, blocksInfo, balances, features, accountsDataStor, sponsoredAssets}, nil
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

type txAppender struct {
	// rw is needed to check for duplicate tx IDs.
	rw *blockReadWriter

	settings *settings.BlockchainSettings

	// TransactionHandler is handler for any operations on transactions.
	txHandler *transactionHandler
	// Block differ is used to create diffs from blocks.
	blockDiffer *blockDiffer
	// Storage for diffs of transactions coming in added blocks.
	diffStorAppendedBlocks *diffStorage
	// Ids of all transactions whose diffs are currently in diffStorAppendedBlocks.
	// This is needed to check that transaction ids are unique.
	appendedBlocksTxIds map[string]struct{}
	// Storage for diffs of transactions coming for validation without blocks.
	diffStorNoBlocks *diffStorage
	// Ids of all transactions whose diffs are currently in diffStorNoBlocks.
	// This is needed to check that transaction ids are unique.
	noBlocksTxIds map[string]struct{}
	// diffApplier is used to both validate and apply balance diffs.
	diffApplier *diffApplier
}

func newTxAppender(rw *blockReadWriter, stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*txAppender, error) {
	genesis, err := settings.GenesisGetter.Get()
	if err != nil {
		return nil, err
	}
	txHandler, err := newTransactionHandler(genesis.BlockSignature, stor, settings)
	if err != nil {
		return nil, err
	}
	blockDiffer, err := newBlockDiffer(txHandler, stor, settings)
	if err != nil {
		return nil, err
	}
	diffStorAppendedBlocks, err := newDiffStorage()
	if err != nil {
		return nil, err
	}
	diffStorNoBlocks, err := newDiffStorage()
	if err != nil {
		return nil, err
	}
	diffApplier, err := newDiffApplier(stor.balances)
	if err != nil {
		return nil, err
	}
	return &txAppender{
		rw:                     rw,
		settings:               settings,
		txHandler:              txHandler,
		blockDiffer:            blockDiffer,
		appendedBlocksTxIds:    make(map[string]struct{}),
		diffStorAppendedBlocks: diffStorAppendedBlocks,
		noBlocksTxIds:          make(map[string]struct{}),
		diffStorNoBlocks:       diffStorNoBlocks,
		diffApplier:            diffApplier,
	}, nil
}

func (a *txAppender) checkDuplicateTxIdsImpl(id []byte, recentIds map[string]struct{}) error {
	// Check recent.
	if _, ok := recentIds[string(id)]; ok {
		return errors.Errorf("transaction with ID %v already in state", id)
	}
	// Check DB.
	if _, err := a.rw.readTransaction(id); err == nil {
		return errors.Errorf("transaction with ID %v already in state", id)
	}
	return nil
}

func (a *txAppender) checkDuplicateTxIds(tx proto.Transaction, recentIds map[string]struct{}, timestamp uint64) error {
	if tx.GetTypeVersion().Type == proto.PaymentTransaction {
		// Payment transactions are deprecated.
		return nil
	}
	if tx.GetTypeVersion().Type == proto.CreateAliasTransaction {
		if (timestamp >= a.settings.StolenAliasesWindowTimeStart) && (timestamp <= a.settings.StolenAliasesWindowTimeEnd) {
			// At this period alias transactions might have duplicate IDs due to bugs in historical blockchain.
			return nil
		}
	}
	txID, err := tx.GetID()
	if err != nil {
		return err
	}
	return a.checkDuplicateTxIdsImpl(txID, recentIds)
}

type appendBlockParams struct {
	transactions   []proto.Transaction
	block, parent  *proto.BlockHeader
	height         uint64
	initialisation bool
}

func (a *txAppender) appendBlock(params *appendBlockParams) error {
	hasParent := (params.parent != nil)
	for _, tx := range params.transactions {
		checkerInfo := &checkerInfo{
			initialisation:   params.initialisation,
			currentTimestamp: params.block.Timestamp,
			blockID:          params.block.BlockSignature,
			height:           params.height,
		}
		if err := a.checkDuplicateTxIds(tx, a.appendedBlocksTxIds, params.block.Timestamp); err != nil {
			return err
		}
		// Add transaction ID.
		txID, err := tx.GetID()
		if err != nil {
			return err
		}
		a.appendedBlocksTxIds[string(txID)] = empty
		if hasParent {
			checkerInfo.parentTimestamp = params.parent.Timestamp
		}
		if err := a.txHandler.checkTx(tx, checkerInfo); err != nil {
			return err
		}
		if err := a.txHandler.performTx(tx, &performerInfo{params.initialisation, params.block.BlockSignature}); err != nil {
			return err
		}
	}
	blockDiff, err := a.blockDiffer.createBlockDiff(params.transactions, params.block, params.initialisation, hasParent)
	if err != nil {
		return err
	}
	if err := a.diffStorAppendedBlocks.saveBlockDiff(blockDiff); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) applyAllDiffs(initialisation bool) error {
	changes := a.diffStorAppendedBlocks.allChanges()
	a.appendedBlocksTxIds = make(map[string]struct{})
	a.diffStorAppendedBlocks.reset()
	if err := a.diffApplier.applyBalancesChanges(changes, !initialisation); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) validateSingleTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	dummy := make(map[string]struct{})
	if err := a.checkDuplicateTxIds(tx, dummy, currentTimestamp); err != nil {
		return err
	}
	// Check tx signature and data.
	if err := checkTx(tx); err != nil {
		return err
	}
	// Check tx data against state.
	checkerInfo := &checkerInfo{initialisation: false, currentTimestamp: currentTimestamp, parentTimestamp: parentTimestamp}
	if err := a.txHandler.checkTx(tx, checkerInfo); err != nil {
		return err
	}
	diff, err := a.txHandler.createDiffTx(tx, &differInfo{initialisation: false, blockTime: currentTimestamp})
	if err != nil {
		return err
	}
	if err := a.diffApplier.validateBalancesChanges(diff.balancesChanges(), true); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) resetValidationList() {
	a.noBlocksTxIds = make(map[string]struct{})
	a.diffStorNoBlocks.reset()
}

func (a *txAppender) validateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	if err := a.checkDuplicateTxIds(tx, a.noBlocksTxIds, currentTimestamp); err != nil {
		return err
	}
	// Add transaction ID.
	txID, err := tx.GetID()
	if err != nil {
		return err
	}
	a.noBlocksTxIds[string(txID)] = empty
	// Check tx signature and data.
	if err := checkTx(tx); err != nil {
		return err
	}
	// Check tx data against state.
	checkerInfo := &checkerInfo{initialisation: false, currentTimestamp: currentTimestamp, parentTimestamp: parentTimestamp}
	if err := a.txHandler.checkTx(tx, checkerInfo); err != nil {
		return err
	}
	diff, err := a.txHandler.createDiffTx(tx, &differInfo{initialisation: false, blockTime: currentTimestamp})
	if err != nil {
		return err
	}
	changes, err := a.diffStorNoBlocks.changesByTxDiff(diff)
	if err != nil {
		return err
	}
	if err := a.diffApplier.validateBalancesChanges(changes, true); err != nil {
		return err
	}
	if err := a.diffStorNoBlocks.saveBalanceChanges(changes); err != nil {
		return err
	}
	return nil
}

func (a *txAppender) reset() {
	a.appendedBlocksTxIds = make(map[string]struct{})
	a.diffStorAppendedBlocks.reset()
	a.blockDiffer.reset()
}

type stateManager struct {
	mu *sync.RWMutex

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

	// Miscellaneous/utility fields.
	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int
	// Indicates whether lease cancellations were performed.
	leasesCl0, leasesCl1, leasesCl2 bool
	// Indicates that stolen aliases were disabled.
	disabledStolenAliases bool
	// The height when last features voting took place.
	lastVotingHeight uint64
}

func (s *stateManager) Mutex() *lock.RwMutex {
	return lock.NewRwMutex(s.mu)
}

func (s *stateManager) Peers() ([]proto.TCPAddr, error) {
	return s.peers.peers()
}

func newStateManager(dataDir string, params StateParams, settings *settings.BlockchainSettings) (*stateManager, error) {
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create blocks directory: %v\n", err))
		}
	}
	// Initialize database.
	dbDir := filepath.Join(dataDir, keyvalueDir)
	log.Printf("Initializing state database, will take up to few minutes...\n")
	db, err := keyvalue.NewKeyVal(dbDir, params.DbParams)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create db: %v\n", err))
	}
	log.Printf("Finished initializing database.\n")
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create db batch: %v\n", err))
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create stateDB: %v\n", err))
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db, dbBatch)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v\n", err))
	}
	if err := stateDB.syncRw(rw); err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to sync block storage and DB: %v\n", err))
	}
	hs, err := newHistoryStorage(db, dbBatch, rw, stateDB)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create history storage: %v\n", err))
	}
	stor, err := newBlockchainEntitiesStorage(hs, stateDB, settings)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create blockchain entities storage: %v\n", err))
	}
	appender, err := newTxAppender(rw, stor, settings)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state := &stateManager{
		stateDB:                   stateDB,
		stor:                      stor,
		rw:                        rw,
		settings:                  settings,
		peers:                     newPeerStorage(db),
		appender:                  appender,
		verificationGoroutinesNum: params.VerificationGoroutinesNum,
		mu:                        &sync.RWMutex{},
	}
	// Set fields which depend on state.
	// Consensus validator is needed to check block headers.
	cv, err := consensus.NewConsensusValidator(state)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.cv = cv
	// Handle genesis block.
	if err := state.handleGenesisBlock(settings.GenesisGetter); err != nil {
		return nil, wrapErr(Other, err)
	}
	return state, nil
}

func (s *stateManager) setGenesisBlock(genesisBlock *proto.Block) error {
	s.genesis = *genesisBlock
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
	if err := s.reset(); err != nil {
		return wrapErr(ModificationError, err)
	}
	return nil
}

func (s *stateManager) applyPreactivatedFeatures(features []int16) error {
	genesisID := s.genesis.BlockSignature
	genesisBlockNum, err := s.stateDB.blockIdToNum(genesisID)
	if err != nil {
		return err
	}
	for _, featureID := range features {
		approvalRequest := &approvedFeaturesRecord{1, genesisBlockNum}
		if err := s.stor.features.approveFeature(featureID, approvalRequest); err != nil {
			return err
		}
		activationRequest := &activatedFeaturesRecord{1, genesisBlockNum}
		if err := s.stor.features.activateFeature(featureID, activationRequest); err != nil {
			return err
		}
	}
	if err := s.flush(true); err != nil {
		return err
	}
	if err := s.reset(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) handleGenesisBlock(g settings.GenesisGetter) error {
	height, err := s.Height()
	if err != nil {
		return err
	}

	block, err := g.Get()
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
		if err := s.applyPreactivatedFeatures(s.settings.PreactivatedFeatures); err != nil {
			return errors.Errorf("failed to apply preactivated features: %v\n", err)
		}
		if err := s.addGenesisBlock(); err != nil {
			return errors.Errorf("failed to apply/save genesis: %v\n", err)
		}
	}
	return nil
}

func (s *stateManager) Header(blockID crypto.Signature) (*proto.BlockHeader, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		if err == keyvalue.ErrNotFound {
			return nil, NewNotFoundError(err)
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
		return crypto.Signature{}, wrapErr(InvalidInputError, errors.New("height out of valid range"))
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return crypto.Signature{}, wrapErr(RetrievalError, err)
	}
	return blockID, nil
}

func (s *stateManager) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	if asset == nil {
		profile, err := s.stor.balances.newestWavesBalance(*addr, true)
		if err != nil {
			return 0, wrapErr(RetrievalError, err)
		}
		return profile.balance, nil
	}
	balance, err := s.stor.balances.newestAssetBalance(*addr, asset, true)
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
			log.Printf("Block has vote for featureID %v, but it is already approved.", featureID)
			continue
		}
		if err := s.stor.features.addVote(featureID, block.BlockSignature); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addNewBlock(block, parent *proto.Block, initialisation bool, chans *verifierChans, height uint64) error {
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
	for _, tx := range transactions {
		// Send transaction for signature/data verification.
		task := &verifyTask{
			taskType: verifyTx,
			tx:       tx,
		}
		select {
		case verifyError := <-chans.errChan:
			return verifyError
		case chans.tasksChan <- task:
		}
		// Save transaction to storage.
		txID, err := tx.GetID()
		if err != nil {
			return err
		}

		// TODO: not all transactions implement WriteTo.
		bts, err := tx.MarshalBinary()
		if err != nil {
			return err
		}
		if err := s.rw.writeTransaction(txID, bts); err != nil {
			return err
		}
	}
	var parentHeader *proto.BlockHeader
	if parent != nil {
		parentHeader = &parent.BlockHeader
	}
	params := &appendBlockParams{
		transactions:   transactions,
		block:          &block.BlockHeader,
		parent:         parentHeader,
		height:         height,
		initialisation: initialisation,
	}
	// Check and perform block's transactions, create balance diffs.
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
	return nil
}

func (s *stateManager) reset() error {
	s.rw.reset()
	s.stor.reset()
	s.stateDB.reset()
	s.appender.reset()
	return nil
}

func (s *stateManager) flush(initialisation bool) error {
	if err := s.rw.flush(); err != nil {
		return err
	}
	if err := s.stor.flush(initialisation); err != nil {
		return err
	}
	if err := s.stateDB.flush(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) undoBlockAddition() error {
	if err := s.reset(); err != nil {
		return err
	}
	if err := s.stateDB.syncRw(s.rw); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) AddBlock(block []byte) (*proto.Block, error) {
	rs, err := s.addBlocks([][]byte{block}, false)
	if err != nil {
		if err := s.undoBlockAddition(); err != nil {
			panic("Failed to add blocks and can not rollback to previous state after failure.")
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	blockBytes, err := block.MarshalBinary()
	if err != nil {
		return nil, wrapErr(SerializationError, err)
	}
	return s.AddBlock(blockBytes)
}

func (s *stateManager) AddNewBlocks(blocks [][]byte) error {
	if _, err := s.addBlocks(blocks, false); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			panic("Failed to add blocks and can not rollback to previous state after failure.")
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

func (s *stateManager) AddOldBlocks(blocks [][]byte) error {
	if _, err := s.addBlocks(blocks, true); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			panic("Failed to add blocks and can not rollback to previous state after failure.")
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
	// Indicates that the task to perform before calling addBlocks() is to cancel leases.
	cancelLeases bool
	// Indicates that the task to perfrom before calling addBlocks() is to reset stolen aliases.
	resetStolenAliases bool
	// Indicates that the task to perform before calling addBlocks() is to finish features voting period.
	finishVotingPeriod bool
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
	return false, nil
}

func (s *stateManager) finishVoting() error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	last, err := s.topBlock()
	if err != nil {
		return err
	}
	if err := s.stor.features.finishVoting(height, last.BlockSignature); err != nil {
		return err
	}
	s.lastVotingHeight = height
	if err := s.flush(true); err != nil {
		return err
	}
	if err := s.reset(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) cancelLeases() error {
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
		if err := s.stor.leases.cancelLeases(nil); err != nil {
			return err
		}
		if err := s.stor.balances.cancelAllLeases(); err != nil {
			return err
		}
		s.leasesCl0 = true
	} else if height == s.settings.BlockVersion3AfterHeight {
		overflowAddrs, err := s.stor.balances.cancelLeaseOverflows()
		if err != nil {
			return err
		}
		if err := s.stor.leases.cancelLeases(overflowAddrs); err != nil {
			return err
		}
		s.leasesCl1 = true
	} else if dataTxActivated && height == dataTxHeight {
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelInvalidLeaseIns(leaseIns); err != nil {
			return err
		}
		s.leasesCl2 = true
	}
	if err := s.flush(true); err != nil {
		return err
	}
	if err := s.reset(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) handleBreak(blocksToFinish [][]byte, initialisation bool, task *breakerTask) (*proto.Block, error) {
	if task == nil {
		return nil, wrapErr(Other, errors.New("handleBreak received empty task"))
	}
	if task.finishVotingPeriod {
		if err := s.finishVoting(); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
	}
	if task.cancelLeases {
		// Need to cancel leases due to bugs in historical blockchain.
		if err := s.cancelLeases(); err != nil {
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

func (s *stateManager) addBlocks(blocks [][]byte, initialisation bool) (*proto.Block, error) {
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
	var blocksToFinish [][]byte
	breakerInfo := &breakerTask{}

	// Launch verifier that checks signatures of blocks and transactions.
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum)

	var lastBlock *proto.Block
	for i, blockBytes := range blocks {
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
		var block proto.Block
		if err := block.UnmarshalBinary(blockBytes); err != nil {
			return nil, wrapErr(DeserializationError, err)
		}
		// Send block for signature verification, which works in separate goroutine.
		task := &verifyTask{
			taskType:   verifyBlock,
			parentSig:  parent.BlockSignature,
			block:      &block,
			blockBytes: blockBytes[:len(blockBytes)-crypto.SignatureSize],
		}
		select {
		case verifyError := <-chans.errChan:
			return nil, wrapErr(ValidationError, verifyError)
		case chans.tasksChan <- task:
		}
		lastBlock = &block
		// Add score.
		score, err := CalculateScore(block.BaseTarget)
		if err != nil {
			return nil, wrapErr(Other, err)
		}
		if err := s.stor.scores.addScore(prevScore, score, s.rw.recentHeight()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		prevScore = score
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockSignature); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// Save block to storage, check its transactions, create and save balance diffs for its transactions.
		if err := s.addNewBlock(&block, parent, initialisation, chans, curHeight); err != nil {
			return nil, wrapErr(TxValidationError, err)
		}
		headers[i] = block.BlockHeader
		parent = &block
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
	if err := s.reset(); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	// Check if we need to perform some event and call addBlocks() again.
	if blocksToFinish != nil {
		return s.handleBreak(blocksToFinish, initialisation, breakerInfo)
	}
	log.Printf("State: blocks to height %d added.\n", height+uint64(blocksNumber))
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
	return nil
}

func (s *stateManager) RollbackTo(removalEdge crypto.Signature) error {
	if err := s.rollbackToImpl(removalEdge); err != nil {
		if err1 := s.stateDB.syncRw(s.rw); err1 != nil {
			panic("Failed to rollback and can not sync state components after failure.")
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
		return nil, wrapErr(InvalidInputError, errors.New("height out of valid range"))
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

func (s *stateManager) ValidateSingleTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	if err := s.appender.validateSingleTx(tx, currentTimestamp, parentTimestamp); err != nil {
		return wrapErr(TxValidationError, err)
	}
	return nil
}

func (s *stateManager) ResetValidationList() {
	s.appender.resetValidationList()
}

func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	if err := s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp); err != nil {
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

func (s *stateManager) IsActivated(featureID int16) (bool, error) {
	activated, err := s.stor.features.isActivated(featureID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return activated, nil
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

// Accounts data storage.

func (s *stateManager) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.newestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestEntry(*addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveNewestIntegerEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveIntegerEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveNewestBooleanEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveBooleanEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveNewestStringEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveStringEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveNewestBinaryEntry(*addr, key)
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
	entry, err := s.stor.accountsDataStor.retrieveBinaryEntry(*addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
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

func (s *stateManager) Close() error {
	if err := s.rw.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	if err := s.stateDB.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	return nil
}

package state

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	rollbackMaxBlocks      = 2000
	blocksStorDir          = "blocks_storage"
	keyvalueDir            = "key_value"
	bloomBitsPerKeyDefault = 10
)

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func genesisFilePath(s *settings.BlockchainSettings) (string, error) {
	dir, err := getLocalDir()
	if err != nil {
		return "", err
	}
	switch s.Type {
	case settings.MainNet:
		return filepath.Join(dir, "genesis", "mainnet.json"), nil
	case settings.TestNet:
		return filepath.Join(dir, "genesis", "testnet.json"), nil
	default:
		if _, err := os.Stat(s.GenesisCfgPath); err != nil {
			return "", err
		}
		return s.GenesisCfgPath, nil
	}
}

type blockchainEntitiesStorage struct {
	hs       *historyStorage
	aliases  *aliases
	assets   *assets
	leases   *leases
	scores   *scores
	balances *balances
	features *features
}

func newBlockchainEntitiesStorage(hs *historyStorage, sets *settings.BlockchainSettings) (*blockchainEntitiesStorage, error) {
	aliases, err := newAliases(hs)
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
	balances, err := newBalances(hs.db, hs)
	if err != nil {
		return nil, err
	}
	features, err := newFeatures(hs.db, hs.dbBatch, hs, sets, settings.FeaturesInfo)
	if err != nil {
		return nil, err
	}
	return &blockchainEntitiesStorage{hs, aliases, assets, leases, scores, balances, features}, nil
}

func (s *blockchainEntitiesStorage) reset() {
	s.hs.reset()
}

func (s *blockchainEntitiesStorage) flush(initialisation bool) error {
	return s.hs.flush(!initialisation)
}

type stateManager struct {
	genesis proto.Block
	stateDB *stateDB

	stor  *blockchainEntitiesStorage
	rw    *blockReadWriter
	peers *peerStorage

	rb *recentBlocks

	settings *settings.BlockchainSettings
	cv       *consensus.ConsensusValidator

	// Transaction validator that performs checking against state for single transactions without block.
	standaloneTv *transactionValidator
	// Transaction validator that performs checking against state for multiple transactions without block.
	multiTxTv *transactionValidator

	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int

	// Indicates whether lease cancellations were performed.
	leasesCl0, leasesCl1, leasesCl2 bool

	// The height when last features voting took place.
	lastVotingHeight uint64
}

func newStateManager(dataDir string, params StateParams, settings *settings.BlockchainSettings) (*stateManager, error) {
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create blocks directory: %v\n", err)}
		}
	}
	// Initialize database.
	dbDir := filepath.Join(dataDir, keyvalueDir)
	db, err := keyvalue.NewKeyVal(dbDir, params.BloomParams)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create db: %v\n", err)}
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create db batch: %v\n", err)}
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create stateDB: %v\n", err)}
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db, dbBatch)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create block storage: %v\n", err)}
	}
	if err := stateDB.syncRw(rw); err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to sync block storage and DB: %v\n", err)}
	}
	// rb is in-memory storage for IDs of recent blocks.
	rb, err := newRecentBlocks(rollbackMaxBlocks, rw)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create recent blocks stor: %v\n", err)}
	}
	hs, err := newHistoryStorage(db, dbBatch, stateDB, rb)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create history storage: %v\n", err)}
	}
	stor, err := newBlockchainEntitiesStorage(hs, settings)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create blockchain entities storage: %v\n", err)}
	}
	state := &stateManager{
		stateDB:                   stateDB,
		stor:                      stor,
		rw:                        rw,
		rb:                        rb,
		settings:                  settings,
		peers:                     newPeerStorage(db),
		verificationGoroutinesNum: params.VerificationGoroutinesNum,
	}
	// Set fields which depend on state.
	// Consensus validator is needed to check block headers.
	cv, err := consensus.NewConsensusValidator(state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	state.cv = cv
	// Handle genesis block.
	genesisPath, err := genesisFilePath(settings)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	if err := state.handleGenesisBlock(genesisPath); err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	// Set transaction validators for transactions without blocks (used by miner, UTX pool).
	if err := state.setValidators(); err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	return state, nil
}

func (s *stateManager) Peers() ([]proto.TCPAddr, error) {
	return s.peers.peers()
}

func (s *stateManager) setValidators() error {
	var err error
	s.standaloneTv, err = newTransactionValidator(s.genesis.BlockSignature, s.stor, s.settings)
	if err != nil {
		return err
	}
	s.multiTxTv, err = newTransactionValidator(s.genesis.BlockSignature, s.stor, s.settings)
	if err != nil {
		return err
	}
	return nil
}

func (s *stateManager) setGenesisBlock(genesisCfgPath string) error {
	genesisFile, err := os.Open(genesisCfgPath)
	if err != nil {
		return errors.Errorf("failed to open genesis file: %v\n", err)
	}
	jsonParser := json.NewDecoder(genesisFile)
	if err := jsonParser.Decode(&s.genesis); err != nil {
		return errors.Errorf("failed to parse JSON of genesis block: %v\n", err)
	}
	if err := genesisFile.Close(); err != nil {
		return errors.Errorf("failed to close genesis file: %v\n", err)
	}
	return nil
}

func (s *stateManager) addGenesisBlock() error {
	// Add score of genesis block.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	genesisScore, err := calculateScore(s.genesis.BaseTarget)
	if err != nil {
		return err
	}
	if err := s.stor.scores.addScore(&big.Int{}, genesisScore, 1); err != nil {
		return err
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.stor, s.settings)
	if err != nil {
		return err
	}
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum)
	if err := s.addNewBlock(tv, &s.genesis, nil, true, chans); err != nil {
		return err
	}
	close(chans.tasksChan)
	if err := tv.validateTransactions(true, true); err != nil {
		return err
	}
	verifyError := <-chans.errChan
	if verifyError != nil {
		return StateError{errorType: BlockValidationError, originalError: err}
	}
	if err := s.flush(true); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.reset(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	return nil
}

func (s *stateManager) applyPreactivatedFeatures(features []int16) error {
	genesisID := s.genesis.BlockSignature
	for _, featureID := range features {
		approvalRequest := &approvedFeaturesRecord{1, genesisID}
		if err := s.stor.features.approveFeature(featureID, approvalRequest); err != nil {
			return err
		}
		activationRequest := &activatedFeaturesRecord{1, genesisID}
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

func (s *stateManager) handleGenesisBlock(genesisCfgPath string) error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	if err := s.setGenesisBlock(genesisCfgPath); err != nil {
		return err
	}
	// If the storage is new (data dir does not contain any data), genesis block must be applied.
	if height == 0 {
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
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	var header proto.BlockHeader
	if err := header.UnmarshalHeaderFromBinary(headerBytes); err != nil {
		return nil, StateError{errorType: DeserializationError, originalError: err}
	}
	return &header, nil
}

func (s *stateManager) HeaderBytes(blockID crypto.Signature) ([]byte, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return headerBytes, nil
}

func (s *stateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.Header(blockID)
}

func (s *stateManager) HeaderBytesByHeight(height uint64) ([]byte, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.HeaderBytes(blockID)
}

func (s *stateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	header, err := s.Header(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	transactions, err := s.rw.readTransactionsBlock(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	block := proto.Block{BlockHeader: *header}
	block.Transactions = make([]byte, len(transactions))
	copy(block.Transactions, transactions)
	return &block, nil
}

func (s *stateManager) BlockBytes(blockID crypto.Signature) ([]byte, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	transactions, err := s.rw.readTransactionsBlock(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	blockBytes, err := proto.AppendHeaderBytesToTransactions(headerBytes, transactions)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	return blockBytes, nil
}

func (s *stateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.Block(blockID)
}

func (s *stateManager) BlockBytesByHeight(height uint64) ([]byte, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.BlockBytes(blockID)
}

func (s *stateManager) Height() (uint64, error) {
	height, err := s.rw.currentHeight()
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	height, err := s.rw.heightByBlockID(blockID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return crypto.Signature{}, StateError{errorType: RetrievalError, originalError: err}
	}
	if height < 1 || height > maxHeight {
		return crypto.Signature{}, StateError{errorType: InvalidInputError, originalError: errors.New("height out of valid range")}
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return crypto.Signature{}, StateError{errorType: RetrievalError, originalError: err}
	}
	return blockID, nil
}

func (s *stateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	if asset == nil {
		profile, err := s.stor.balances.wavesBalance(addr, true)
		if err != nil {
			return 0, StateError{errorType: RetrievalError, originalError: err}
		}
		return profile.balance, nil
	}
	balance, err := s.stor.balances.assetBalance(addr, asset, true)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return balance, nil
}

func (s *stateManager) WavesAddressesNumber() (uint64, error) {
	res, err := s.stor.balances.wavesAddressesNumber()
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
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
		if err := s.stor.features.addVote(featureID, block.BlockSignature); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addNewBlock(tv *transactionValidator, block, parent *proto.Block, initialisation bool, chans *verifierChans) error {
	if err := s.rb.addNewBlockID(block.BlockSignature); err != nil {
		return err
	}
	if err := s.stateDB.addBlock(block.BlockSignature); err != nil {
		return err
	}
	// Indicate new block for storage.
	if err := s.rw.startBlock(block.BlockSignature); err != nil {
		return err
	}
	// Save block header to storage.
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		return err
	}
	if err := s.rw.writeBlockHeader(block.BlockSignature, headerBytes); err != nil {
		return err
	}
	transactions := block.Transactions
	// Validate transactions.
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transactions[0:4]))
		if n+4 > len(transactions) {
			return errors.New("invalid tx size: exceeds bytes slice bounds")
		}
		txBytes := transactions[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			return err
		}
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
		if err := s.rw.writeTransaction(tx.GetID(), transactions[:n+4]); err != nil {
			return err
		}
		parentTimestamp := uint64(0)
		if parent != nil {
			parentTimestamp = parent.Timestamp
		}
		// Validate transaction against state.
		info := &txValidationInfo{
			perform:          true,
			initialisation:   initialisation,
			validate:         false,
			currentTimestamp: block.Timestamp,
			parentTimestamp:  parentTimestamp,
			minerPK:          block.GenPublicKey,
			blockID:          block.BlockSignature,
		}
		if err = tv.addTxForValidation(tx, info); err != nil {
			return err
		}
		transactions = transactions[4+n:]
	}
	if err := s.rw.finishBlock(block.BlockSignature); err != nil {
		return err
	}
	if err := s.addFeaturesVotes(block); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) reset() error {
	s.rw.reset()
	s.stor.reset()
	s.stateDB.reset()
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
	s.rb.flush()
	return nil
}

func (s *stateManager) undoBlockAddition() error {
	if err := s.reset(); err != nil {
		return err
	}
	if err := s.stateDB.syncRw(s.rw); err != nil {
		return err
	}
	s.rb.reset()
	return nil
}

func (s *stateManager) AddBlock(block []byte) error {
	blocks := make([][]byte, 1)
	blocks[0] = block
	if err := s.addBlocks(blocks, false); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			panic("Failed to add blocks and can not rollback to previous state after failure.")
		}
		return err
	}
	return nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) error {
	blockBytes, err := block.MarshalBinary()
	if err != nil {
		return StateError{errorType: SerializationError, originalError: err}
	}
	return s.AddBlock(blockBytes)
}

func (s *stateManager) AddNewBlocks(blocks [][]byte) error {
	if err := s.addBlocks(blocks, false); err != nil {
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
		return StateError{errorType: SerializationError, originalError: err}
	}
	return s.AddNewBlocks(blocksBytes)
}

func (s *stateManager) AddOldBlocks(blocks [][]byte) error {
	if err := s.addBlocks(blocks, true); err != nil {
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
		return StateError{errorType: SerializationError, originalError: err}
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

func (s *stateManager) needToCancelLeases(height uint64) bool {
	switch height {
	case s.settings.ResetEffectiveBalanceAtHeight:
		return !s.leasesCl0
	case s.settings.BlockVersion3AfterHeight:
		return !s.leasesCl1
	default:
		return false
	}
}

type breakerTask struct {
	// Indicates that the task to perform before calling addBlocks() is to cancel leases.
	cancelLeases bool
	// Indicates that the task to perform before calling addBlocks() is to finish features voting period.
	finishVotingPeriod bool
}

func (s *stateManager) needToBreakAddingBlocks(curHeight uint64, task *breakerTask) bool {
	if s.needToCancelLeases(curHeight) {
		task.cancelLeases = true
	}
	if s.needToFinishVotingPeriod(curHeight) {
		task.finishVotingPeriod = true
	}
	return task.cancelLeases || task.finishVotingPeriod
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

func (s *stateManager) handleBreak(blocksToFinish [][]byte, initialisation bool, task *breakerTask) error {
	if task == nil {
		return StateError{errorType: Other, originalError: errors.New("handleBreak received empty task")}
	}
	if task.finishVotingPeriod {
		if err := s.finishVoting(); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
	}
	if task.cancelLeases {
		// Need to cancel leases due to bugs in historical blockchain.
		if err := s.cancelLeases(); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
	}
	return s.addBlocks(blocksToFinish, initialisation)
}

func (s *stateManager) addBlocks(blocks [][]byte, initialisation bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	blocksNumber := len(blocks)
	parent, err := s.topBlock()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.stor, s.settings)
	if err != nil {
		return StateError{errorType: Other, originalError: err}
	}
	height, err := s.Height()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	prevScore, err := s.stor.scores.score(height)
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	headers := make([]proto.BlockHeader, blocksNumber)
	var blocksToFinish [][]byte
	breakerInfo := &breakerTask{}
	chans := newVerifierChans()
	go launchVerifier(ctx, chans, s.verificationGoroutinesNum)
	for i, blockBytes := range blocks {
		curHeight := height + uint64(i)
		if s.needToBreakAddingBlocks(curHeight, breakerInfo) {
			// Need to break at this height, so we split block batch in order to cancel and finish with the rest blocks after.
			blocksToFinish = blocks[i:]
			break
		}
		var block proto.Block
		if err := block.UnmarshalBinary(blockBytes); err != nil {
			return StateError{errorType: DeserializationError, originalError: err}
		}
		task := &verifyTask{
			taskType:   verifyBlock,
			parentSig:  parent.BlockSignature,
			block:      &block,
			blockBytes: blockBytes[:len(blockBytes)-crypto.SignatureSize],
		}
		select {
		case verifyError := <-chans.errChan:
			return StateError{errorType: ValidationError, originalError: verifyError}
		case chans.tasksChan <- task:
		}
		// Add score.
		score, err := calculateScore(block.BaseTarget)
		if err != nil {
			return StateError{errorType: Other, originalError: err}
		}
		if err := s.stor.scores.addScore(prevScore, score, s.rw.recentHeight()); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
		prevScore = score
		if err := s.addNewBlock(tv, &block, parent, initialisation, chans); err != nil {
			return StateError{errorType: TxValidationError, originalError: err}
		}
		headers[i] = block.BlockHeader
		parent = &block
	}
	close(chans.tasksChan)
	if err := tv.validateTransactions(initialisation, true); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	if err := s.cv.ValidateHeaders(headers[:len(headers)-len(blocksToFinish)], height); err != nil {
		return StateError{errorType: BlockValidationError, originalError: err}
	}
	verifyError := <-chans.errChan
	if verifyError != nil {
		return StateError{errorType: ValidationError, originalError: err}
	}
	if err := s.flush(initialisation); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.reset(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if blocksToFinish != nil {
		return s.handleBreak(blocksToFinish, initialisation, breakerInfo)
	}
	log.Printf("State: blocks to height %d added.\n", height+uint64(blocksNumber))
	return nil
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
		return StateError{errorType: RetrievalError, originalError: err}
	}
	if err := s.checkRollbackInput(blockID); err != nil {
		return StateError{errorType: InvalidInputError, originalError: err}
	}
	if err := s.RollbackTo(blockID); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
	}
	return nil
}

func (s *stateManager) rollbackToImpl(removalEdge crypto.Signature) error {
	if err := s.checkRollbackInput(removalEdge); err != nil {
		return StateError{errorType: InvalidInputError, originalError: err}
	}
	curHeight, err := s.rw.currentHeight()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	for height := curHeight; height > 0; height-- {
		blockID, err := s.rw.blockIDByHeight(height)
		if err != nil {
			return StateError{errorType: RetrievalError, originalError: err}
		}
		if bytes.Equal(blockID[:], removalEdge[:]) {
			break
		}
		if err := s.stateDB.rollbackBlock(blockID); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	}
	// Remove blocks from block storage.
	if err := s.rw.rollback(removalEdge, true); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
	}
	// Remove scores of deleted blocks.
	newHeight, err := s.Height()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	oldHeight := curHeight + 1
	if err := s.stor.scores.rollback(newHeight, oldHeight); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
	}
	// Reset recent block IDs storage.
	s.rb.reset()
	return nil
}

func (s *stateManager) RollbackTo(removalEdge crypto.Signature) error {
	if err := s.rollbackToImpl(removalEdge); err != nil {
		if err1 := s.stateDB.syncRw(s.rw); err1 != nil {
			panic("Failed to rollback and can not sync state components after failure.")
		}
		s.rb.reset()
		return err
	}
	return nil
}

func (s *stateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	if height < 1 || height > maxHeight {
		return nil, StateError{errorType: InvalidInputError, originalError: errors.New("height out of valid range")}
	}
	score, err := s.stor.scores.score(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return score, nil
}

func (s *stateManager) CurrentScore() (*big.Int, error) {
	height, err := s.Height()
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.ScoreAtHeight(height)
}

func (s *stateManager) EffectiveBalance(addr proto.Address, startHeight, endHeight uint64) (uint64, error) {
	effectiveBalance, err := s.stor.balances.minEffectiveBalanceInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
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
	if err := checkTx(tx); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	info := &txValidationInfo{
		perform:          false,
		initialisation:   false,
		validate:         true,
		currentTimestamp: currentTimestamp,
		parentTimestamp:  parentTimestamp,
	}
	if err := s.standaloneTv.addTxForValidation(tx, info); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	s.standaloneTv.reset()
	return nil
}

func (s *stateManager) ResetValidationList() {
	s.multiTxTv.reset()
}

func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	if err := checkTx(tx); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	info := &txValidationInfo{
		perform:          false,
		initialisation:   false,
		validate:         true,
		currentTimestamp: currentTimestamp,
		parentTimestamp:  parentTimestamp,
	}
	if err := s.multiTxTv.addTxForValidation(tx, info); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	return nil
}

func (s *stateManager) IsActivated(featureID int16) (bool, error) {
	activated, err := s.stor.features.isActivated(featureID)
	if err != nil {
		return false, StateError{errorType: RetrievalError, originalError: err}
	}
	return activated, nil
}

func (s *stateManager) ActivationHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.activationHeight(featureID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) IsApproved(featureID int16) (bool, error) {
	approved, err := s.stor.features.isApproved(featureID)
	if err != nil {
		return false, StateError{errorType: RetrievalError, originalError: err}
	}
	return approved, nil
}

func (s *stateManager) ApprovalHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.approvalHeight(featureID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) Close() error {
	if err := s.rw.close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	if err := s.stateDB.close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	return nil
}

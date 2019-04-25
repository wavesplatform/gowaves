package state

import (
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
	rollbackMaxBlocks = 2000
	blocksStorDir     = "blocks_storage"
	keyvalueDir       = "keyvalue"
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

type stateManager struct {
	genesis proto.Block
	stateDB *stateDB

	assets   *assets
	leases   *leases
	scores   *scores
	balances *balances
	rw       *blockReadWriter
	peers    *peerStorage

	settings *settings.BlockchainSettings
	cv       *consensus.ConsensusValidator

	// Indicates whether lease cancellations were performed.
	leasesCl0, leasesCl1, leasesCl2 bool
}

func (s *stateManager) Peers() ([]proto.TCPAddr, error) {
	return s.peers.peers()
}

func newStateManager(dataDir string, params BlockStorageParams, settings *settings.BlockchainSettings) (*stateManager, error) {
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create blocks directory: %v\n", err)}
		}
	}
	// Initialize database.
	dbDir := filepath.Join(dataDir, keyvalueDir)
	db, err := keyvalue.NewKeyVal(dbDir)
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
	// scores is storage for blocks score.
	scores, err := newScores(db, dbBatch)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create scores: %v\n", err)}
	}
	state := &stateManager{
		stateDB:  stateDB,
		scores:   scores,
		settings: settings,
		peers:    newPeerStorage(db),
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db, dbBatch)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create block storage: %v\n", err)}
	}
	// balances is storage for balances of accounts.
	balances, err := newBalances(db, dbBatch, state, state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create balances storage: %v\n", err)}
	}
	if err := stateDB.syncRw(rw); err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to sync block storage and DB: %v\n", err)}
	}
	// assets is storage for assets info.
	assets, err := newAssets(db, dbBatch, state, state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create assets storage: %v\n", err)}
	}
	// leases is storage for leases info.
	leases, err := newLeases(db, dbBatch, state, state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create leases storage: %v\n", err)}
	}
	// Consensus validator is needed to check block headers.
	cv, err := consensus.NewConsensusValidator(state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	// Set fields which depend on state.
	state.assets = assets
	state.leases = leases
	state.cv = cv
	state.balances = balances
	state.rw = rw
	// Handle genesis block.
	genesisPath, err := genesisFilePath(settings)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	if err := state.handleGenesisBlock(genesisPath); err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	return state, nil
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
	genesisScore, err := calculateScore(s.genesis.BaseTarget)
	if err != nil {
		return err
	}
	if err := s.scores.addScore(&big.Int{}, genesisScore, 1); err != nil {
		return err
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.balances, s.assets, s.leases, s.settings)
	if err != nil {
		return err
	}
	if err := s.addNewBlock(tv, &s.genesis, nil, true); err != nil {
		return err
	}
	if err := tv.performTransactions(); err != nil {
		return err
	}
	if err := s.flush(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.reset(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
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
		if err := s.addGenesisBlock(); err != nil {
			return errors.Errorf("failed to apply/save genesis: %v\n", err)
		}
	}
	return nil
}

func (s *stateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	headerBytes, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	transactions, err := s.rw.readTransactionsBlock(blockID)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	var block proto.Block
	if err := block.UnmarshalHeaderFromBinary(headerBytes); err != nil {
		return nil, StateError{errorType: DeserializationError, originalError: err}
	}
	block.Transactions = make([]byte, len(transactions))
	copy(block.Transactions, transactions)
	return &block, nil
}

func (s *stateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	if height < 1 || height > maxHeight {
		return nil, StateError{errorType: InvalidInputError, originalError: errors.New("height out of valid range")}
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	return s.Block(blockID)
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

func (s *stateManager) NewBlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	height, err := s.rw.heightByNewBlockID(blockID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	id, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return crypto.Signature{}, StateError{errorType: RetrievalError, originalError: err}
	}
	return id, nil
}

func (s *stateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	if asset == nil {
		profile, err := s.balances.wavesBalance(addr)
		if err != nil {
			return 0, StateError{errorType: RetrievalError, originalError: err}
		}
		return profile.balance, nil
	}
	balance, err := s.balances.assetBalance(addr, asset)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return balance, nil
}

func (s *stateManager) WavesAddressesNumber() (uint64, error) {
	res, err := s.balances.wavesAddressesNumber()
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

func (s *stateManager) addNewBlock(tv *transactionValidator, block, parent *proto.Block, initialisation bool) error {
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
		txBytes := transactions[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			return err
		}
		// Save transaction to storage.
		if err := s.rw.writeTransaction(tx.GetID(), transactions[:n+4]); err != nil {
			return err
		}
		// Validate transaction against state.
		if err = tv.validateTransaction(block, parent, tx, initialisation); err != nil {
			return err
		}
		transactions = transactions[4+n:]
	}
	if err := s.rw.finishBlock(block.BlockSignature); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) reset() error {
	s.rw.reset()
	s.assets.reset()
	s.leases.reset()
	s.balances.reset()
	s.stateDB.reset()
	return nil
}

func (s *stateManager) flush() error {
	if err := s.rw.flush(); err != nil {
		return err
	}
	if err := s.assets.flush(); err != nil {
		return err
	}
	if err := s.leases.flush(); err != nil {
		return err
	}
	if err := s.balances.flush(); err != nil {
		return err
	}
	if err := s.stateDB.flush(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) unmarshalAndCheck(blockBytes []byte, parentSig crypto.Signature, initialisation bool) (*proto.Block, error) {
	var block proto.Block
	if err := block.UnmarshalBinary(blockBytes); err != nil {
		return nil, err
	}
	// Check block signature.
	if !crypto.Verify(block.GenPublicKey, block.BlockSignature, blockBytes[:len(blockBytes)-crypto.SignatureSize]) {
		return nil, errors.New("invalid block signature")
	}
	// Check parent.
	if parentSig != block.Parent {
		return nil, errors.New("incorrect parent")
	}
	return &block, nil
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

func (s *stateManager) AddNewBlocks(blocks [][]byte) error {
	if err := s.addBlocks(blocks, false); err != nil {
		if err := s.undoBlockAddition(); err != nil {
			panic("Failed to add blocks and can not rollback to previous state after failure.")
		}
		return err
	}
	return nil
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

func (s *stateManager) cancelLeases() error {
	height, err := s.Height()
	if err != nil {
		return err
	}
	switch height {
	case s.settings.ResetEffectiveBalanceAtHeight:
		if err := s.leases.cancelLeases(nil); err != nil {
			return err
		}
		if err := s.balances.cancelAllLeases(); err != nil {
			return err
		}
		s.leasesCl0 = true
	case s.settings.BlockVersion3AfterHeight:
		overflowAddrs, err := s.balances.cancelLeaseOverflows()
		if err != nil {
			return err
		}
		if err := s.leases.cancelLeases(overflowAddrs); err != nil {
			return err
		}
		s.leasesCl1 = true

		//TODO
		//case blockchainFeatures.DataTransactionHeight:
		//leaseIns, err := s.leases.validLeaseIns()
		//if err != nil {
		//	return err
		//}
		//if err := s.balances.cancelInvalidLeaseIns(leaseIns); err != nil {
		//	return err
		//}
		//s.leasesCl2 = true
	}
	if err := s.flush(); err != nil {
		return err
	}
	if err := s.reset(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) addBlocks(blocks [][]byte, initialisation bool) error {
	blocksNumber := len(blocks)
	parent, err := s.topBlock()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.balances, s.assets, s.leases, s.settings)
	if err != nil {
		return StateError{errorType: Other, originalError: err}
	}
	height, err := s.Height()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	prevScore, err := s.scores.score(height)
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	var blocksToFinish [][]byte
	headers := make([]proto.BlockHeader, blocksNumber)
	for i, blockBytes := range blocks {
		curHeight := height + uint64(i)
		if s.needToCancelLeases(curHeight) {
			// Need to cancel something, so we split block batch in order to cancel and finish with the rest blocks after.
			blocksToFinish = blocks[i:]
			break
		}
		block, err := s.unmarshalAndCheck(blockBytes, parent.BlockSignature, initialisation)
		if err != nil {
			return StateError{errorType: DeserializationError, originalError: err}
		}
		// Add score.
		score, err := calculateScore(block.BaseTarget)
		if err != nil {
			return StateError{errorType: Other, originalError: err}
		}
		if err := s.scores.addScore(prevScore, score, s.rw.recentHeight()); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
		prevScore = score
		if err := s.addNewBlock(tv, block, parent, initialisation); err != nil {
			return StateError{errorType: TxValidationError, originalError: err}
		}
		headers[i] = block.BlockHeader
		parent = block
	}
	if err := tv.performTransactions(); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	if err := s.cv.ValidateHeaders(headers[:len(headers)-len(blocksToFinish)], height); err != nil {
		return StateError{errorType: BlockValidationError, originalError: err}
	}
	if err := s.flush(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.reset(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if blocksToFinish != nil {
		// Need to cancel leases due to bugs in historical blockchain.
		if err := s.cancelLeases(); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
		return s.addBlocks(blocksToFinish, initialisation)
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
		return errors.New("invalid height")
	}
	return nil
}

func (s *stateManager) RollbackToHeight(height uint64) error {
	maxHeight, err := s.Height()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	if height < 1 || height > maxHeight {
		return StateError{errorType: InvalidInputError, originalError: errors.New("height out of valid range")}
	}
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

func (s *stateManager) RollbackTo(removalEdge crypto.Signature) error {
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
		if blockID == removalEdge {
			break
		}
		if err := s.stateDB.rollbackBlock(blockID); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	}
	// Remove scores of deleted blocks.
	newHeight, err := s.Height()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	oldHeight := curHeight + 1
	if err := s.scores.rollback(newHeight, oldHeight); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
	}
	// Remove blocks from block storage.
	if err := s.rw.rollback(removalEdge, true); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
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
	score, err := s.scores.score(height)
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
	effectiveBalance, err := s.balances.minEffectiveBalanceInRange(addr, startHeight, endHeight)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return effectiveBalance, nil
}

func (s *stateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	return s.settings, nil
}

func (s *stateManager) RollbackMax() uint64 {
	return rollbackMaxBlocks
}

func (s *stateManager) IsValidBlock(blockID crypto.Signature) (bool, error) {
	return s.stateDB.isValidBlock(blockID)
}

func (s *stateManager) SavePeers(peers []proto.TCPAddr) error {
	return s.peers.savePeers(peers)

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

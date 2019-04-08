package state

import (
	"encoding/binary"
	"math/big"
	"os"
	"path/filepath"

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

type stateManager struct {
	genesis proto.Block
	stateDB *stateDB

	assets   *assets
	scores   *scores
	balances *balances
	rw       *blockReadWriter

	settings *settings.BlockchainSettings
	cv       *consensus.ConsensusValidator
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
	// Consensus validator is needed to check block headers.
	cv, err := consensus.NewConsensusValidator(state)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: err}
	}
	state.assets = assets
	state.cv = cv
	state.balances = balances
	state.rw = rw
	// If the storage is new (data dir does not contain any data), genesis block must be applied.
	height, err := state.Height()
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to get genesis signature from string: %v\n", err)}
	}
	state.setGenesis(genesisSig)
	if height == 1 {
		if err := state.applyGenesis(genesisSig); err != nil {
			return nil, StateError{errorType: ModificationError, originalError: errors.Errorf("failed to apply genesis: %v\n", err)}
		}
	}
	return state, nil
}

func (s *stateManager) setGenesis(genesisSig crypto.Signature) {
	// Set genesis block itself.
	// TODO: MainNet's genesis is hard coded for now, support settings.BlockchainSettings.
	s.genesis = proto.Block{
		BlockHeader: proto.BlockHeader{
			Version:        1,
			Timestamp:      1460678400000,
			BaseTarget:     153722867,
			BlockSignature: genesisSig,
			Height:         1,
		},
	}
}

func (s *stateManager) applyGenesis(genesisSig crypto.Signature) error {
	// Add genesis to list of valid blocks, so DB will know about it.
	if err := s.stateDB.addBlock(genesisSig); err != nil {
		return err
	}
	// Add score of genesis block.
	genesisScore, err := calculateScore(s.genesis.BaseTarget)
	if err != nil {
		return err
	}
	if err := s.scores.addScore(&big.Int{}, genesisScore, 1); err != nil {
		return err
	}
	// Perform and validate genesis transactions.
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.balances, s.assets, s.settings)
	if err != nil {
		return err
	}
	genesisTx, err := generateGenesisTransactions()
	if err != nil {
		return err
	}
	for _, tx := range genesisTx {
		if err := tv.validateTransaction(&s.genesis, nil, &tx, true); err != nil {
			return err
		}
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

func (s *stateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	if blockID == s.genesis.BlockSignature {
		return &s.genesis, nil
	}
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
	if height == 1 {
		return &s.genesis, nil
	}
	blockID, err := s.rw.blockIDByHeight(height - 2)
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
	return height + 1, nil
}

func (s *stateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if blockID == s.genesis.BlockSignature {
		return 1, nil
	}
	height, err := s.rw.heightByBlockID(blockID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) NewBlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if blockID == s.genesis.BlockSignature {
		return 1, nil
	}
	height, err := s.rw.heightByNewBlockID(blockID)
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	if height == 1 {
		return s.genesis.BlockSignature, nil
	}
	id, err := s.rw.blockIDByHeight(height - 2)
	if err != nil {
		return crypto.Signature{}, StateError{errorType: RetrievalError, originalError: err}
	}
	return id, nil
}

func (s *stateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	key := balanceKey{address: addr, asset: asset}
	balance, err := s.balances.accountBalance(key.bytes())
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return balance, nil
}

func (s *stateManager) AddressesNumber(wavesOnly bool) (uint64, error) {
	res, err := s.balances.addressesNumber(wavesOnly)
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

func (s *stateManager) addBlocks(blocks [][]byte, initialisation bool) error {
	blocksNumber := len(blocks)
	parent, err := s.topBlock()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.balances, s.assets, s.settings)
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
	headers := make([]proto.BlockHeader, blocksNumber)
	for i, blockBytes := range blocks {
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
	if err := s.cv.ValidateHeaders(headers, height); err != nil {
		return StateError{errorType: BlockValidationError, originalError: err}
	}
	if err := s.flush(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.reset(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
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
		blockID, err := s.rw.blockIDByHeight(height - 1)
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
	if removalEdge == s.genesis.BlockSignature {
		// Remove blocks from block storage.
		if err := s.rw.rollbackToGenesis(true); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	} else {
		// Remove blocks from block storage.
		if err := s.rw.rollback(removalEdge, true); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	}
	return nil
}

func (s *stateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
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
	key := balanceKey{address: addr}
	effectiveBalance, err := s.balances.minBalanceInRange(key.bytes(), startHeight, endHeight)
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

func (s *stateManager) Close() error {
	if err := s.rw.close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	if err := s.stateDB.close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	return nil
}

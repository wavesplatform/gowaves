package state

import (
	"encoding/binary"
	"math/big"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	rollbackMaxBlocks = 4000
	blocksStorDir     = "blocks_storage"
	keyvalueDir       = "keyvalue"
)

type stateManager struct {
	genesis  proto.Block
	db       keyvalue.KeyValue
	scores   *scores
	accounts *accountsStorage
	rw       *blockReadWriter
}

func syncDbAndStorage(db keyvalue.KeyValue, stor *accountsStorage, rw *blockReadWriter) error {
	dbHeightBytes, err := db.Get([]byte{dbHeightKeyPrefix})
	if err != nil {
		return err
	}
	dbHeight := binary.LittleEndian.Uint64(dbHeightBytes)
	rwHeighBytes, err := db.Get([]byte{rwHeightKeyPrefix})
	if err != nil {
		return err
	}
	rwHeight := binary.LittleEndian.Uint64(rwHeighBytes)
	if rwHeight < dbHeight {
		// This should never happen, because we update block storage before writing changes into DB.
		panic("Impossible to sync: DB is ahead of block storage; remove data dir and restart the node.")
	}
	if dbHeight == 0 {
		if err := rw.reset(false); err != nil {
			return errors.Errorf("failed to reset block storage: %v", err)
		}
	} else {
		last, err := rw.blockIDByHeight(dbHeight - 1)
		if err != nil {
			return err
		}
		if err := rw.rollback(last, false); err != nil {
			return errors.Errorf("failed to remove blocks from block storage: %v", err)
		}
	}
	return nil
}

func newStateManager(dataDir string, params BlockStorageParams) (*stateManager, error) {
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to get genesis signature from string: %v\n", err)}
	}
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create blocks directory: %v\n", err)}
		}
	}
	dbDir := filepath.Join(dataDir, keyvalueDir)
	db, err := keyvalue.NewKeyVal(dbDir, true)
	scores, err := newScores(db)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create scores: %v\n", err)}
	}
	rw, err := newBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create block storage: %v\n", err)}
	}
	accountsStor, err := newAccountsStorage(genesisSig, db)
	if err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to create accounts storage: %v\n", err)}
	}
	accountsStor.setRollbackMax(rollbackMaxBlocks, rw)
	if err := syncDbAndStorage(db, accountsStor, rw); err != nil {
		return nil, StateError{errorType: Other, originalError: errors.Errorf("failed to sync block storage and DB: %v\n", err)}
	}
	genesis := proto.Block{
		BlockHeader: proto.BlockHeader{
			Version:        1,
			Timestamp:      1460678400000,
			BaseTarget:     153722867,
			BlockSignature: genesisSig,
			Height:         1,
		},
	}
	state := &stateManager{genesis: genesis, db: db, scores: scores, accounts: accountsStor, rw: rw}
	height, err := state.Height()
	if err != nil {
		return nil, StateError{errorType: RetrievalError, originalError: err}
	}
	if height == 1 {
		if err := state.applyGenesis(); err != nil {
			return nil, StateError{errorType: ModificationError, originalError: errors.Errorf("failed to apply genesis: %v\n", err)}
		}
	}
	return state, nil
}

func (s *stateManager) applyGenesis() error {
	// Add score of genesis block.
	genesisScore, err := calculateScore(s.genesis.BaseTarget)
	if err != nil {
		return err
	}
	if err := s.scores.addScore(&big.Int{}, genesisScore, 1); err != nil {
		return err
	}
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.accounts, proto.MainNetScheme)
	if err != nil {
		return err
	}
	genesisTx, err := generateGenesisTransactions()
	if err != nil {
		return err
	}
	for _, tx := range genesisTx {
		if err := tv.validateTransaction(&s.genesis, &tx, true); err != nil {
			return err
		}
	}
	if err := tv.performTransactions(); err != nil {
		return err
	}
	if err := s.accounts.flush(); err != nil {
		return err
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
	return height + 2, nil
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
	balance, err := s.accounts.accountBalance(key.bytes())
	if err != nil {
		return 0, StateError{errorType: RetrievalError, originalError: err}
	}
	return balance, nil
}

func (s *stateManager) AddressesNumber() (uint64, error) {
	res, err := s.accounts.addressesNumber()
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

func (s *stateManager) addNewBlock(tv *transactionValidator, block *proto.Block, initialisation bool) error {
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
		if tv.isSupported(tx) {
			// Genesis, Payment, TransferV1 and TransferV2 Waves-only for now.
			if err = tv.validateTransaction(block, tx, initialisation); err != nil {
				return err
			}
		}
		transactions = transactions[4+n:]
	}
	if err := s.rw.finishBlock(block.BlockSignature); err != nil {
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

func (s *stateManager) AddBlock(block []byte) error {
	blocks := make([][]byte, 1)
	blocks[0] = block
	return s.addBlocks(blocks, false)
}

func (s *stateManager) AddNewBlocks(blocks [][]byte) error {
	return s.addBlocks(blocks, false)
}

func (s *stateManager) AddOldBlocks(blocks [][]byte) error {
	return s.addBlocks(blocks, true)
}

func (s *stateManager) addBlocks(blocks [][]byte, initialisation bool) error {
	blocksNumber := len(blocks)
	parent, err := s.topBlock()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	parentSig := parent.BlockSignature
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.accounts, proto.MainNetScheme)
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
	for _, blockBytes := range blocks {
		block, err := s.unmarshalAndCheck(blockBytes, parentSig, initialisation)
		if err != nil {
			return StateError{errorType: DeserializationError, originalError: err}
		}
		// Add score.
		score, err := calculateScore(block.BaseTarget)
		if err != nil {
			return StateError{errorType: Other, originalError: err}
		}
		if err := s.scores.addScore(prevScore, score, s.rw.recentHeight()+2); err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
		prevScore = score
		if err := s.addNewBlock(tv, block, initialisation); err != nil {
			return StateError{errorType: TxValidationError, originalError: err}
		}
		parentSig = block.BlockSignature
	}
	if err := tv.performTransactions(); err != nil {
		return StateError{errorType: TxValidationError, originalError: err}
	}
	if err := s.rw.updateHeight(blocksNumber); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.rw.flush(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.accounts.updateHeight(blocksNumber); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	if err := s.accounts.flush(); err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}
	return nil
}

func (s *stateManager) RollbackToHeight(height uint64) error {
	// Rollback accounts storage.
	curHeight, err := s.rw.currentHeight()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	oldHeight := curHeight + 1
	if height < 1 {
		return StateError{errorType: RollbackError, originalError: errors.New("minimum block to rollback to is the first block")}
	} else if height == 1 {
		for h := curHeight; h > 0; h-- {
			blockID, err := s.rw.blockIDByHeight(h - 1)
			if err != nil {
				return StateError{errorType: RetrievalError, originalError: err}
			}
			if err := s.accounts.rollbackBlock(blockID); err != nil {
				return StateError{errorType: RollbackError, originalError: err}
			}
		}
		// Remove blocks from block storage.
		if err := s.rw.reset(true); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	} else {
		blockID, err := s.rw.blockIDByHeight(height - 2)
		if err != nil {
			return StateError{errorType: RetrievalError, originalError: err}
		}
		if err := s.RollbackTo(blockID); err != nil {
			return StateError{errorType: RollbackError, originalError: err}
		}
	}
	// Remove scores of deleted blocks.
	if err := s.scores.rollback(height, oldHeight); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
	}
	return nil
}

func (s *stateManager) RollbackTo(removalEdge crypto.Signature) error {
	// Rollback accounts storage.
	curHeight, err := s.rw.currentHeight()
	if err != nil {
		return StateError{errorType: RetrievalError, originalError: err}
	}
	oldHeight := curHeight + 1
	for height := curHeight; height > 0; height-- {
		blockID, err := s.rw.blockIDByHeight(height - 1)
		if err != nil {
			return StateError{errorType: RetrievalError, originalError: err}
		}
		if blockID == removalEdge {
			break
		}
		if err := s.accounts.rollbackBlock(blockID); err != nil {
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
	if err := s.scores.rollback(newHeight, oldHeight); err != nil {
		return StateError{errorType: RollbackError, originalError: err}
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

func (s *stateManager) Close() error {
	if err := s.rw.close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	if err := s.db.Close(); err != nil {
		return StateError{errorType: ClosureError, originalError: err}
	}
	return nil
}

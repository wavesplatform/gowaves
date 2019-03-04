package state

import (
	"encoding/binary"
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

type StateManager struct {
	genesis         proto.Block
	db              keyvalue.KeyValue
	accountsStorage *AccountsStorage
	rw              *BlockReadWriter
}

type BlockStorageParams struct {
	OffsetLen, HeaderOffsetLen int
}

func DefaultBlockStorageParams() BlockStorageParams {
	return BlockStorageParams{OffsetLen: 8, HeaderOffsetLen: 8}
}

func syncDbAndStorage(db keyvalue.KeyValue, stor *AccountsStorage, rw *BlockReadWriter) error {
	dbHeightBytes, err := db.Get([]byte{DbHeightKeyPrefix})
	if err != nil {
		return err
	}
	dbHeight := binary.LittleEndian.Uint64(dbHeightBytes)
	rwHeighBytes, err := db.Get([]byte{RwHeightKeyPrefix})
	if err != nil {
		return err
	}
	rwHeight := binary.LittleEndian.Uint64(rwHeighBytes)
	if rwHeight < dbHeight {
		// This should never happen, because we update block storage before writing changes into DB.
		panic("Impossible to sync: DB is ahead of block storage; remove data dir and restart the node.")
	}
	if dbHeight == 0 {
		if err := rw.Reset(false); err != nil {
			return errors.Errorf("failed to reset block storage: %v", err)
		}
	} else {
		last, err := rw.BlockIDByHeight(dbHeight - 1)
		if err != nil {
			return err
		}
		if err := rw.Rollback(last, false); err != nil {
			return errors.Errorf("failed to remove blocks from block storage: %v", err)
		}
	}
	return nil
}

func NewStateManager(dataDir string, params BlockStorageParams) (*StateManager, error) {
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	if err != nil {
		return nil, errors.Errorf("failed to get genesis signature from string: %v\n", err)
	}
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); os.IsNotExist(err) {
		if err := os.Mkdir(blockStorageDir, 0755); err != nil {
			return nil, errors.Errorf("failed to create blocks directory: %v\n", err)
		}
	}
	dbDir := filepath.Join(dataDir, keyvalueDir)
	db, err := keyvalue.NewKeyVal(dbDir, true)
	rw, err := NewBlockReadWriter(blockStorageDir, params.OffsetLen, params.HeaderOffsetLen, db)
	if err != nil {
		return nil, errors.Errorf("failed to create block storage: %v\n", err)
	}
	accountsStor, err := NewAccountsStorage(genesisSig, db)
	if err != nil {
		return nil, errors.Errorf("failed to create accounts storage: %v\n", err)
	}
	accountsStor.SetRollbackMax(rollbackMaxBlocks, rw)
	if err := syncDbAndStorage(db, accountsStor, rw); err != nil {
		return nil, errors.Errorf("failed to sync block storage and DB: %v\n", err)
	}
	genesis := proto.Block{
		BlockHeader: proto.BlockHeader{
			Version:        1,
			Timestamp:      1460678400000,
			BlockSignature: genesisSig,
			Height:         1,
		},
	}
	state := &StateManager{genesis: genesis, db: db, accountsStorage: accountsStor, rw: rw}
	height, err := state.Height()
	if err != nil {
		return nil, errors.Errorf("failed to get height: %v\n", err)
	}
	if height == 1 {
		if err := state.applyGenesis(); err != nil {
			return nil, errors.Errorf("failed to apply genesis: %v\n", err)
		}
	}
	return state, nil
}

func (s *StateManager) applyGenesis() error {
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.accountsStorage, proto.MainNetScheme)
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
	if err := s.accountsStorage.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) GetBlock(blockID crypto.Signature) (*proto.Block, error) {
	if blockID == s.genesis.BlockSignature {
		return &s.genesis, nil
	}
	headerBytes, err := s.rw.ReadBlockHeader(blockID)
	if err != nil {
		return nil, err
	}
	transactions, err := s.rw.ReadTransactionsBlock(blockID)
	if err != nil {
		return nil, err
	}
	var block proto.Block
	if err := block.UnmarshalHeaderFromBinary(headerBytes); err != nil {
		return nil, err
	}
	block.Transactions = make([]byte, len(transactions))
	copy(block.Transactions, transactions)
	return &block, nil
}

func (s *StateManager) GetBlockByHeight(height uint64) (*proto.Block, error) {
	if height == 1 {
		return &s.genesis, nil
	}
	blockID, err := s.rw.BlockIDByHeight(height - 2)
	if err != nil {
		return nil, err
	}
	return s.GetBlock(blockID)
}

func (s *StateManager) Height() (uint64, error) {
	height, err := s.rw.CurrentHeight()
	if err != nil {
		return 0, err
	}
	return height + 1, nil
}

func (s *StateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if blockID == s.genesis.BlockSignature {
		return 1, nil
	}
	height, err := s.rw.HeightByBlockID(blockID)
	if err != nil {
		return 0, err
	}
	return height + 2, nil
}

func (s *StateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	if height == 1 {
		return s.genesis.BlockSignature, nil
	}
	return s.rw.BlockIDByHeight(height - 2)
}

func (s *StateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	key := BalanceKey{Address: addr, Asset: asset}
	return s.accountsStorage.AccountBalance(key.Bytes())
}

func (s *StateManager) AddressesNumber() (uint64, error) {
	return s.accountsStorage.AddressesNumber()
}

func (s *StateManager) topBlock() (*proto.Block, error) {
	height, err := s.Height()
	if err != nil {
		return nil, err
	}
	// Heights start from 1.
	return s.GetBlockByHeight(height)
}

func (s *StateManager) addNewBlock(tv *transactionValidator, block *proto.Block, initialisation bool) error {
	// Indicate new block for storage.
	if err := s.rw.StartBlock(block.BlockSignature); err != nil {
		return err
	}
	// Save block header to storage.
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		return err
	}
	if err := s.rw.WriteBlockHeader(block.BlockSignature, headerBytes); err != nil {
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
		if err := s.rw.WriteTransaction(tx.GetID(), transactions[:n+4]); err != nil {
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
	if err := s.rw.FinishBlock(block.BlockSignature); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) unmarshalAndCheck(blockBytes []byte, parentSig crypto.Signature, initialisation bool) (*proto.Block, error) {
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

func (s *StateManager) AddBlocks(blocks [][]byte, initialisation bool) error {
	blocksNumber := len(blocks)
	parent, err := s.topBlock()
	if err != nil {
		return err
	}
	parentSig := parent.BlockSignature
	tv, err := newTransactionValidator(s.genesis.BlockSignature, s.accountsStorage, proto.MainNetScheme)
	if err != nil {
		return err
	}
	for _, blockBytes := range blocks {
		block, err := s.unmarshalAndCheck(blockBytes, parentSig, initialisation)
		if err != nil {
			return err
		}
		if err := s.addNewBlock(tv, block, initialisation); err != nil {
			return err
		}
		parentSig = block.BlockSignature
	}
	if err := tv.performTransactions(); err != nil {
		return err
	}
	if err := s.rw.UpdateHeight(blocksNumber); err != nil {
		return err
	}
	if err := s.rw.Flush(); err != nil {
		return err
	}
	if err := s.accountsStorage.UpdateHeight(blocksNumber); err != nil {
		return err
	}
	if err := s.accountsStorage.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) RollbackToHeight(height uint64) error {
	if height < 1 {
		return errors.New("minimum block to rollback to is the first block")
	} else if height == 1 {
		// Rollback accounts storage.
		curHeight, err := s.rw.CurrentHeight()
		if err != nil {
			return err
		}
		for h := curHeight; h > 0; h-- {
			blockID, err := s.rw.BlockIDByHeight(h - 1)
			if err != nil {
				return errors.Errorf("failed to get block ID by height: %v\n", err)
			}
			if err := s.accountsStorage.RollbackBlock(blockID); err != nil {
				return errors.Errorf("failed to rollback accounts storage: %v", err)
			}
		}
		// Remove blocks from block storage.
		return s.rw.Reset(true)
	} else {
		blockID, err := s.rw.BlockIDByHeight(height - 2)
		if err != nil {
			return err
		}
		return s.RollbackTo(blockID)
	}
}

func (s *StateManager) RollbackTo(removalEdge crypto.Signature) error {
	// Rollback accounts storage.
	curHeight, err := s.rw.CurrentHeight()
	if err != nil {
		return err
	}
	for height := curHeight; height > 0; height-- {
		blockID, err := s.rw.BlockIDByHeight(height - 1)
		if err != nil {
			return errors.Errorf("failed to get block ID by height: %v\n", err)
		}
		if blockID == removalEdge {
			break
		}
		if err := s.accountsStorage.RollbackBlock(blockID); err != nil {
			return errors.Errorf("failed to rollback accounts storage: %v", err)
		}
	}
	// Remove blocks from block storage.
	if err := s.rw.Rollback(removalEdge, true); err != nil {
		return errors.Errorf("failed to remove blocks from block storage: %v", err)
	}
	return nil
}

func (s *StateManager) Close() error {
	if err := s.rw.Close(); err != nil {
		return err
	}
	if err := s.db.Close(); err != nil {
		return err
	}
	return nil
}

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

type WavesBalanceKey [1 + proto.AddressSize]byte
type AssetBalanceKey [1 + proto.AddressSize + crypto.DigestSize]byte

type BalancesStorage struct {
	global *AccountsStorage
	assets map[AssetBalanceKey]uint64
	waves  map[WavesBalanceKey]uint64
}

func NewBalancesStorage(global *AccountsStorage) (*BalancesStorage, error) {
	return &BalancesStorage{
		global: global,
		assets: make(map[AssetBalanceKey]uint64),
		waves:  make(map[WavesBalanceKey]uint64),
	}, nil
}

func (stor *BalancesStorage) AccountBalance(key []byte) (uint64, error) {
	size := len(key)
	if size == 1+proto.AddressSize {
		var wavesKey WavesBalanceKey
		copy(wavesKey[:], key)
		_, ok := stor.waves[wavesKey]
		if !ok {
			balance, err := stor.global.AccountBalance(key)
			if err != nil {
				return 0, err
			}
			stor.waves[wavesKey] = balance
		}
		return stor.waves[wavesKey], nil
	} else if size == 1+proto.AddressSize+crypto.DigestSize {
		var assetKey AssetBalanceKey
		copy(assetKey[:], key)
		_, ok := stor.assets[assetKey]
		if !ok {
			balance, err := stor.global.AccountBalance(key)
			if err != nil {
				return 0, err
			}
			stor.assets[assetKey] = balance
		}
		return stor.assets[assetKey], nil
	}
	return 0, errors.New("invalid key size")
}

func (stor *BalancesStorage) SetAccountBalance(key []byte, balance uint64) error {
	size := len(key)
	if size == 1+proto.AddressSize {
		var wavesKey WavesBalanceKey
		copy(wavesKey[:], key)
		stor.waves[wavesKey] = balance
	} else if size == 1+proto.AddressSize+crypto.DigestSize {
		var assetKey AssetBalanceKey
		copy(assetKey[:], key)
		stor.assets[assetKey] = balance
	} else {
		return errors.New("invalid key size")
	}
	return nil
}

type StateManager struct {
	genesis         crypto.Signature
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
	if dbHeight > 0 {
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
	genesis, err := crypto.NewSignatureFromBase58(genesisSignature)
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
	accountsStor, err := NewAccountsStorage(genesis, db)
	if err != nil {
		return nil, errors.Errorf("failed to create accounts storage: %v\n", err)
	}
	accountsStor.SetRollbackMax(rollbackMaxBlocks, rw)
	if err := syncDbAndStorage(db, accountsStor, rw); err != nil {
		return nil, errors.Errorf("failed to sync block storage and DB: %v\n", err)
	}
	state := &StateManager{genesis: genesis, db: db, accountsStorage: accountsStor, rw: rw}
	return state, nil
}

func (s *StateManager) applyGenesis() error {
	balancesStor, err := NewBalancesStorage(s.accountsStorage)
	if err != nil {
		return err
	}
	tv, err := NewTransactionValidator(s.genesis, balancesStor)
	if err != nil {
		return err
	}
	genesisTx, err := generateGenesisTransactions()
	if err != nil {
		return err
	}
	for _, tx := range genesisTx {
		if err := tv.ValidateTransaction(s.genesis, &tx, true); err != nil {
			return errors.Wrap(err, "invalid genesis transaction")
		}
		if err := s.performGenesisTransaction(tx, balancesStor); err != nil {
			return errors.Wrap(err, "failed to perform genesis transaction")
		}
	}
	// Write transactions from local balances storage into DB batch.
	if err := s.addChangesToBatch(balancesStor, s.genesis); err != nil {
		return err
	}
	// Write batch to DB.
	if err := s.db.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) GetBlock(blockID crypto.Signature) (*proto.Block, error) {
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
	blockID, err := s.rw.BlockIDByHeight(height)
	if err != nil {
		return nil, err
	}
	return s.GetBlock(blockID)
}

func (s *StateManager) Height() (uint64, error) {
	return s.rw.CurrentHeight()
}

func (s *StateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	return s.rw.HeightByBlockID(blockID)
}

func (s *StateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	return s.rw.BlockIDByHeight(height)
}

func (s *StateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	key := BalanceKey{Address: addr, Asset: asset}
	return s.accountsStorage.AccountBalance(key.Bytes())
}

func (s *StateManager) AddressesNumber() (uint64, error) {
	return s.accountsStorage.AddressesNumber()
}

func (s *StateManager) performGenesisTransaction(tx proto.Genesis, stor *BalancesStorage) error {
	key := BalanceKey{Address: tx.Recipient}
	receiverBalance, err := stor.AccountBalance(key.Bytes())
	if err != nil {
		return err
	}
	newReceiverBalance := receiverBalance + tx.Amount
	if err := stor.SetAccountBalance(key.Bytes(), newReceiverBalance); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) performTransaction(block *proto.Block, tx proto.Transaction, stor *BalancesStorage) error {
	switch v := tx.(type) {
	case *proto.Payment:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderKey := BalanceKey{Address: senderAddr}
		senderBalance, err := stor.AccountBalance(senderKey.Bytes())
		if err != nil {
			return err
		}
		newSenderBalance := senderBalance - v.Amount - v.Fee
		if newSenderBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := stor.SetAccountBalance(senderKey.Bytes(), newSenderBalance); err != nil {
			return err
		}
		receiverKey := BalanceKey{Address: v.Recipient}
		receiverBalance, err := stor.AccountBalance(receiverKey.Bytes())
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := stor.SetAccountBalance(receiverKey.Bytes(), newReceiverBalance); err != nil {
			return err
		}
		minerKey := BalanceKey{Address: minerAddr}
		minerBalance, err := stor.AccountBalance(minerKey.Bytes())
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := stor.SetAccountBalance(minerKey.Bytes(), newMinerBalance); err != nil {
			return err
		}
		return nil
	case *proto.TransferV1:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("alias without address is not supported yet")
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderFeeKey := BalanceKey{Address: senderAddr, Asset: v.FeeAsset.ToID()}
		senderAmountKey := BalanceKey{Address: senderAddr, Asset: v.AmountAsset.ToID()}
		senderFeeBalance, err := stor.AccountBalance(senderFeeKey.Bytes())
		if err != nil {
			return err
		}
		newSenderFeeBalance := senderFeeBalance - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		senderAmountBalance, err := stor.AccountBalance(senderAmountKey.Bytes())
		if err != nil {
			return err
		}
		newSenderAmountBalance := senderAmountBalance - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := stor.SetAccountBalance(senderFeeKey.Bytes(), newSenderFeeBalance); err != nil {
			return err
		}
		if err := stor.SetAccountBalance(senderAmountKey.Bytes(), newSenderAmountBalance); err != nil {
			return err
		}
		receiverKey := BalanceKey{Address: *v.Recipient.Address, Asset: v.AmountAsset.ToID()}
		receiverBalance, err := stor.AccountBalance(receiverKey.Bytes())
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := stor.SetAccountBalance(receiverKey.Bytes(), newReceiverBalance); err != nil {
			return err
		}
		minerKey := BalanceKey{Address: minerAddr, Asset: v.FeeAsset.ToID()}
		minerBalance, err := stor.AccountBalance(minerKey.Bytes())
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := stor.SetAccountBalance(minerKey.Bytes(), newMinerBalance); err != nil {
			return err
		}
		return nil
	case *proto.TransferV2:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("alias without address is not supported yet")
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderFeeKey := BalanceKey{Address: senderAddr, Asset: v.FeeAsset.ToID()}
		senderAmountKey := BalanceKey{Address: senderAddr, Asset: v.AmountAsset.ToID()}
		senderFeeBalance, err := stor.AccountBalance(senderFeeKey.Bytes())
		if err != nil {
			return err
		}
		newSenderFeeBalance := senderFeeBalance - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		senderAmountBalance, err := stor.AccountBalance(senderAmountKey.Bytes())
		if err != nil {
			return err
		}
		newSenderAmountBalance := senderAmountBalance - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := stor.SetAccountBalance(senderFeeKey.Bytes(), newSenderFeeBalance); err != nil {
			return err
		}
		if err := stor.SetAccountBalance(senderAmountKey.Bytes(), newSenderAmountBalance); err != nil {
			return err
		}
		receiverKey := BalanceKey{Address: *v.Recipient.Address, Asset: v.AmountAsset.ToID()}
		receiverBalance, err := stor.AccountBalance(receiverKey.Bytes())
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := stor.SetAccountBalance(receiverKey.Bytes(), newReceiverBalance); err != nil {
			return err
		}
		minerKey := BalanceKey{Address: minerAddr, Asset: v.FeeAsset.ToID()}
		minerBalance, err := stor.AccountBalance(minerKey.Bytes())
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := stor.SetAccountBalance(minerKey.Bytes(), newMinerBalance); err != nil {
			return err
		}
		return nil
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
}

func (s *StateManager) addChangesToBatch(stor *BalancesStorage, blockID crypto.Signature) error {
	for key, balance := range stor.waves {
		if err := s.accountsStorage.SetAccountBalance(key[:], balance, blockID); err != nil {
			return err
		}
	}
	for key, balance := range stor.assets {
		if err := s.accountsStorage.SetAccountBalance(key[:], balance, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *StateManager) addNewBlock(block *proto.Block, initialisation bool) error {
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
	balancesStor, err := NewBalancesStorage(s.accountsStorage)
	if err != nil {
		return err
	}
	tv, err := NewTransactionValidator(s.genesis, balancesStor)
	if err != nil {
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
		if tv.IsSupported(tx) && (s.accountsStorage != nil) {
			// Genesis, Payment, TransferV1 and TransferV2 Waves-only for now.
			if err = tv.ValidateTransaction(block.BlockSignature, tx, initialisation); err != nil {
				return errors.Wrap(err, "incorrect transaction inside of the block")
			}
			if err = s.performTransaction(block, tx, balancesStor); err != nil {
				return errors.Wrap(err, "failed to perform the transaction")
			}
		}
		transactions = transactions[4+n:]
	}
	// Write transactions from local balances storage into DB batch.
	if err := s.addChangesToBatch(balancesStor, block.BlockSignature); err != nil {
		return err
	}
	// Flush all buffers in BlockReadWriter.
	if err := s.rw.FinishBlock(block.BlockSignature); err != nil {
		return err
	}
	// Write batch to DB.
	if err := s.accountsStorage.FinishBlock(); err != nil {
		return err
	}
	return nil
}

func (s *StateManager) AcceptAndVerifyBlockBinary(data []byte, initialisation bool) error {
	var block proto.Block
	if err := block.UnmarshalBinary(data); err != nil {
		return err
	}
	// Check block signature.
	if !crypto.Verify(block.GenPublicKey, block.BlockSignature, data[:len(data)-crypto.SignatureSize]) {
		return errors.New("invalid block signature")
	}
	// Check parent.
	height, err := s.rw.CurrentHeight()
	if err != nil {
		return err
	}
	if height == 0 {
		if initialisation {
			if err := s.applyGenesis(); err != nil {
				return err
			}
		} else {
			return errors.New("zero height in non-initialisation mode")
		}
		// First block.
		if block.Parent != s.genesis {
			return errors.New("incorrect parent")
		}
	} else {
		parent, err := s.GetBlockByHeight(height - 1)
		if err != nil {
			return err
		}
		if parent.BlockSignature != block.Parent {
			return errors.New("incorrect parent")
		}
	}
	return s.addNewBlock(&block, initialisation)
}

func (s *StateManager) RollbackToHeight(height uint64) error {
	blockID, err := s.rw.BlockIDByHeight(height)
	if err != nil {
		return err
	}
	return s.RollbackTo(blockID)
}

func (s *StateManager) RollbackTo(removalEdge crypto.Signature) error {
	// Rollback accounts storage.
	curHeight, err := s.rw.CurrentHeight()
	if err != nil {
		return err
	}
	for height := curHeight - 1; height > 0; height-- {
		blockID, err := s.rw.BlockIDByHeight(height)
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

package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/storage"
)

var ErrNotFound = errors.New("Not found")

type TaskType byte

const (
	AddBlock TaskType = iota
	RemoveBlocks
)

type StateManagerTask struct {
	Type TaskType
	// For block addition.
	Block *proto.Block
	// For blocks rollback.
	RemovalEdge crypto.Signature
}

type BlockReadWriter interface {
	StartBlock(blockID crypto.Signature) error
	FinishBlock(blockID crypto.Signature) error
	WriteTransaction(txID []byte, tx []byte) error
	WriteBlockHeader(blockID crypto.Signature, header []byte) error
	ReadTransaction(txID []byte) ([]byte, error)
	ReadBlockHeader(blockID crypto.Signature) ([]byte, error)
	ReadTransactionsBlock(blockID crypto.Signature) ([]byte, error)
	RemoveBlocks(removalEdge crypto.Signature) error
	BlockIDByHeight(height uint64) (crypto.Signature, error)
	CurrentHeight() uint64
}

type StateManager struct {
	genesis         crypto.Signature
	accountsStorage *storage.AccountsStorage
	rw              BlockReadWriter
}

func NewStateManager(genesis crypto.Signature, accountsStor *storage.AccountsStorage, rw BlockReadWriter) (*StateManager, error) {
	stor := &StateManager{genesis: genesis, accountsStorage: accountsStor, rw: rw}
	return stor, nil
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
	block.Transactions = make([]byte, block.TransactionBlockLength)
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

func (s *StateManager) performTransaction(block *proto.Block, tx proto.Transaction) error {
	switch v := tx.(type) {
	case proto.Genesis:
		receiverBalance, err := s.accountsStorage.AccountBalance(v.Recipient, nil)
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := s.accountsStorage.SetAccountBalance(v.Recipient, nil, newReceiverBalance); err != nil {
			return err
		}
		return nil
	case proto.Payment:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderBalance, err := s.accountsStorage.AccountBalance(senderAddr, nil)
		if err != nil {
			return err
		}
		newSenderBalance := senderBalance - v.Amount - v.Fee
		if newSenderBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := s.accountsStorage.SetAccountBalance(senderAddr, nil, newSenderBalance); err != nil {
			return err
		}
		receiverBalance, err := s.accountsStorage.AccountBalance(v.Recipient, nil)
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := s.accountsStorage.SetAccountBalance(v.Recipient, nil, newReceiverBalance); err != nil {
			return err
		}
		minerBalance, err := s.accountsStorage.AccountBalance(minerAddr, nil)
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := s.accountsStorage.SetAccountBalance(minerAddr, nil, newMinerBalance); err != nil {
			return err
		}
		return nil
	case proto.TransferV1:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("Alias without address is not supported yet")
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderFeeBalance, err := s.accountsStorage.AccountBalance(senderAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		newSenderFeeBalance := senderFeeBalance - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		senderAmountBalance, err := s.accountsStorage.AccountBalance(senderAddr, v.AmountAsset.ToID())
		if err != nil {
			return err
		}
		newSenderAmountBalance := senderAmountBalance - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := s.accountsStorage.SetAccountBalance(senderAddr, v.FeeAsset.ToID(), newSenderFeeBalance); err != nil {
			return err
		}
		if err := s.accountsStorage.SetAccountBalance(senderAddr, v.AmountAsset.ToID(), newSenderAmountBalance); err != nil {
			return err
		}
		receiverBalance, err := s.accountsStorage.AccountBalance(*v.Recipient.Address, v.AmountAsset.ToID())
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := s.accountsStorage.SetAccountBalance(*v.Recipient.Address, v.AmountAsset.ToID(), newReceiverBalance); err != nil {
			return err
		}
		minerBalance, err := s.accountsStorage.AccountBalance(minerAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := s.accountsStorage.SetAccountBalance(minerAddr, v.FeeAsset.ToID(), newMinerBalance); err != nil {
			return err
		}
		return nil
	case proto.TransferV2:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("Alias without address is not supported yet")
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		senderFeeBalance, err := s.accountsStorage.AccountBalance(senderAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		newSenderFeeBalance := senderFeeBalance - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		senderAmountBalance, err := s.accountsStorage.AccountBalance(senderAddr, v.AmountAsset.ToID())
		if err != nil {
			return err
		}
		newSenderAmountBalance := senderAmountBalance - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		if err := s.accountsStorage.SetAccountBalance(senderAddr, v.FeeAsset.ToID(), newSenderFeeBalance); err != nil {
			return err
		}
		if err := s.accountsStorage.SetAccountBalance(senderAddr, v.AmountAsset.ToID(), newSenderAmountBalance); err != nil {
			return err
		}
		receiverBalance, err := s.accountsStorage.AccountBalance(*v.Recipient.Address, v.AmountAsset.ToID())
		if err != nil {
			return err
		}
		newReceiverBalance := receiverBalance + v.Amount
		if err := s.accountsStorage.SetAccountBalance(*v.Recipient.Address, v.AmountAsset.ToID(), newReceiverBalance); err != nil {
			return err
		}
		minerBalance, err := s.accountsStorage.AccountBalance(minerAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		newMinerBalance := minerBalance + v.Fee
		if err := s.accountsStorage.SetAccountBalance(minerAddr, v.FeeAsset.ToID(), newMinerBalance); err != nil {
			return err
		}
		return nil
	default:
		return errors.Errorf("Transaction type %T is not supported\n", v)
	}
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
	tv, err := proto.NewTransactionValidator(s.genesis, s.accountsStorage)
	if err != nil {
		return err
	}
	transactions := block.Transactions
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
			if err = tv.ValidateTransaction(block, tx, initialisation); err != nil {
				return errors.Wrap(err, "Incorrect transaction inside of the block")
			}
			if err = s.performTransaction(block, tx); err != nil {
				return errors.Wrap(err, "Failed to perform the transaction")
			}
		}
		transactions = transactions[4+n:]
	}
	if err := s.rw.FinishBlock(block.BlockSignature); err != nil {
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
		return errors.New("Invalid block signature.")
	}
	// Check parent.
	height := s.rw.CurrentHeight()
	parent, err := s.GetBlockByHeight(height)
	if err != nil {
		return err
	}
	if parent.BlockSignature != block.Parent {
		return errors.New("Incorrect parent.")
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
	// Remove blocks.
	s.rw.RemoveBlocks(removalEdge)
	if s.accountsStorage != nil {
		// Rollback accounts storage.
		if err := s.accountsStorage.RollbackTo(removalEdge); err != nil {
			return err
		}
	}
	return nil
}

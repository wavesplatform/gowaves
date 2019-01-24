package state

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNotFound = errors.New("Not found")

type TaskType byte

const (
	AddBlock TaskType = iota
	RemoveBlocks
)

type BlockManagerTask struct {
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
}

type BlockManager struct {
	genesis       crypto.Signature
	accountsState proto.AccountsState
	rw            BlockReadWriter
	cancel        context.CancelFunc
}

func NewBlockManager(genesis crypto.Signature, state proto.AccountsState, rw BlockReadWriter) (*BlockManager, error) {
	stor := &BlockManager{genesis: genesis, accountsState: state, rw: rw}
	return stor, nil
}

func (s *BlockManager) GetBlock(blockID crypto.Signature) (*proto.Block, error) {
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

func (s *BlockManager) performTransaction(block *proto.Block, tx proto.Transaction) error {
	wavesAsset, err := proto.NewOptionalAssetFromString(proto.WavesAssetName)
	if err != nil {
		return err
	}
	switch v := tx.(type) {
	case proto.Genesis:
		receiver, err := s.accountsState.Account(v.Recipient)
		if err != nil {
			return err
		}
		newReceiverBalance := receiver.AssetBalance(wavesAsset) + v.Amount
		receiver.SetAssetBalance(wavesAsset, newReceiverBalance)
		return nil
	case proto.Payment:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		sender, err := s.accountsState.Account(senderAddr)
		if err != nil {
			return err
		}
		receiver, err := s.accountsState.Account(v.Recipient)
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		miner, err := s.accountsState.Account(minerAddr)
		if err != nil {
			return err
		}
		newSenderBalance := sender.AssetBalance(wavesAsset) - v.Amount - v.Fee
		if newSenderBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		sender.SetAssetBalance(wavesAsset, newSenderBalance)
		newReceiverBalance := receiver.AssetBalance(wavesAsset) + v.Amount
		receiver.SetAssetBalance(wavesAsset, newReceiverBalance)
		newMinerBalance := miner.AssetBalance(wavesAsset) + v.Fee
		miner.SetAssetBalance(wavesAsset, newMinerBalance)
		return nil
	case proto.TransferV1:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		sender, err := s.accountsState.Account(senderAddr)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("Alias without address is not supported yet")
		}
		receiver, err := s.accountsState.Account(*v.Recipient.Address)
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		miner, err := s.accountsState.Account(minerAddr)
		if err != nil {
			return err
		}
		newSenderFeeBalance := sender.AssetBalance(&v.FeeAsset) - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		newSenderAmountBalance := sender.AssetBalance(&v.AmountAsset) - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		sender.SetAssetBalance(&v.FeeAsset, newSenderFeeBalance)
		sender.SetAssetBalance(&v.AmountAsset, newSenderAmountBalance)
		newReceiverBalance := receiver.AssetBalance(&v.AmountAsset) + v.Amount
		receiver.SetAssetBalance(&v.AmountAsset, newReceiverBalance)
		newMinerBalance := miner.AssetBalance(&v.FeeAsset) + v.Fee
		miner.SetAssetBalance(&v.FeeAsset, newMinerBalance)
		return nil
	case proto.TransferV2:
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return err
		}
		sender, err := s.accountsState.Account(senderAddr)
		if err != nil {
			return err
		}
		if v.Recipient.Address == nil {
			// TODO implement
			return errors.New("Alias without address is not supported yet")
		}
		receiver, err := s.accountsState.Account(*v.Recipient.Address)
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		miner, err := s.accountsState.Account(minerAddr)
		if err != nil {
			return err
		}
		newSenderFeeBalance := sender.AssetBalance(&v.FeeAsset) - v.Fee
		if newSenderFeeBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		newSenderAmountBalance := sender.AssetBalance(&v.AmountAsset) - v.Amount
		if newSenderAmountBalance < 0 {
			panic("Transaction results in negative balance after validation")
		}
		sender.SetAssetBalance(&v.FeeAsset, newSenderFeeBalance)
		sender.SetAssetBalance(&v.AmountAsset, newSenderAmountBalance)
		newReceiverBalance := receiver.AssetBalance(&v.AmountAsset) + v.Amount
		receiver.SetAssetBalance(&v.AmountAsset, newReceiverBalance)
		newMinerBalance := miner.AssetBalance(&v.FeeAsset) + v.Fee
		miner.SetAssetBalance(&v.FeeAsset, newMinerBalance)
		return nil
	default:
		return errors.Errorf("Transaction type %T is not supported\n", v)
	}
}

func (s *BlockManager) AddNewBlock(block *proto.Block, initialisation bool) error {
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
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transactions[0:4]))
		txBytes := transactions[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		// Save transaction to storage.
		s.rw.WriteTransaction(tx.GetID(), txBytes)
		if err != nil {
			return err
		}
		tv, err := proto.NewTransactionValidator(s.genesis, s.accountsState)
		if err != nil {
			return err
		}
		if err = tv.ValidateTransaction(block, tx, initialisation); err != nil {
			return errors.Wrap(err, "Incorrect transaction inside of the block")
		}
		if err = s.performTransaction(block, tx); err != nil {
			return errors.Wrap(err, "Failed to perform the transaction")
		}
	}
	if err := s.rw.FinishBlock(block.BlockSignature); err != nil {
		return err
	}
	return nil
}

func (s *BlockManager) RollbackTo(removalEdge crypto.Signature) error {
	// Remove blocks.
	s.rw.RemoveBlocks(removalEdge)
	// Rollback accounts state.
	return s.accountsState.RollbackTo(removalEdge)
}

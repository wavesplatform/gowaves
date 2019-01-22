package state

import (
	"context"
	"encoding/binary"
	"reflect"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var bytesToTransactionsV2 = map[proto.TransactionType]reflect.Type{
	proto.IssueTransaction:          reflect.TypeOf(proto.IssueV2{}),
	proto.TransferTransaction:       reflect.TypeOf(proto.TransferV2{}),
	proto.ReissueTransaction:        reflect.TypeOf(proto.ReissueV2{}),
	proto.BurnTransaction:           reflect.TypeOf(proto.BurnV2{}),
	proto.ExchangeTransaction:       reflect.TypeOf(proto.ExchangeV2{}),
	proto.LeaseTransaction:          reflect.TypeOf(proto.LeaseV2{}),
	proto.LeaseCancelTransaction:    reflect.TypeOf(proto.LeaseCancelV2{}),
	proto.CreateAliasTransaction:    reflect.TypeOf(proto.CreateAliasV2{}),
	proto.SetScriptTransaction:      reflect.TypeOf(proto.SetScriptV1{}),
	proto.SponsorshipTransaction:    reflect.TypeOf(proto.SponsorshipV1{}),
	proto.SetAssetScriptTransaction: reflect.TypeOf(proto.SetAssetScriptV1{}),
}
var bytesToTransactionsV1 = map[proto.TransactionType]reflect.Type{
	proto.GenesisTransaction:      reflect.TypeOf(proto.Genesis{}),
	proto.PaymentTransaction:      reflect.TypeOf(proto.Payment{}),
	proto.IssueTransaction:        reflect.TypeOf(proto.IssueV1{}),
	proto.TransferTransaction:     reflect.TypeOf(proto.TransferV1{}),
	proto.ReissueTransaction:      reflect.TypeOf(proto.ReissueV1{}),
	proto.BurnTransaction:         reflect.TypeOf(proto.BurnV1{}),
	proto.ExchangeTransaction:     reflect.TypeOf(proto.ExchangeV1{}),
	proto.LeaseTransaction:        reflect.TypeOf(proto.LeaseV1{}),
	proto.LeaseCancelTransaction:  reflect.TypeOf(proto.LeaseCancelV1{}),
	proto.CreateAliasTransaction:  reflect.TypeOf(proto.CreateAliasV1{}),
	proto.MassTransferTransaction: reflect.TypeOf(proto.MassTransferV1{}),
}

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
	WriteTransaction(transactionID []byte, tx []byte) error
	WriteBlockHeader(blockID crypto.Signature, header []byte) error
	ReadTransaction(transactionID crypto.Signature) ([]byte, error)
	ReadBlockHeader(blockID crypto.Signature) ([]byte, error)
	ReadTransactionsBlock(blockID crypto.Signature) ([]byte, error)
	RemoveBlocks(removalEdge crypto.Signature) error
}

type AccountsState interface {
	Account(proto.Recipient) (AccountManipulator, error)
	SetAccount(AccountManipulator) error
	RollbackTo(crypto.Signature) error
}

type AccountManipulator interface {
	SetAssetBalance(*proto.OptionalAsset, uint64)
	AssetBalance(*proto.OptionalAsset) uint64
	Address() proto.Address
}

type BlockManager struct {
	genesis       crypto.Signature
	accountsState AccountsState
	rw            BlockReadWriter
	cancel        context.CancelFunc
}

func BytesToTransaction(tx []byte) (proto.Transaction, error) {
	if len(tx) < 2 {
		return nil, errors.New("Invalid size of transation's bytes slice")
	}
	if tx[0] == 0 {
		transactionType, ok := bytesToTransactionsV2[proto.TransactionType(tx[1])]
		if !ok {
			return nil, errors.New("Invalid transaction type")
		}
		transaction, ok := reflect.New(transactionType).Interface().(proto.TransactionExtended)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx); err != nil {
			return nil, errors.Wrap(err, "Failed to unmarshal transaction")
		}
		return proto.Transaction(transaction), nil
	} else {
		transactionType, ok := bytesToTransactionsV1[proto.TransactionType(tx[0])]
		if !ok {
			return nil, errors.New("Invalid transaction type")
		}
		transaction, ok := reflect.New(transactionType).Interface().(proto.TransactionExtended)
		if !ok {
			panic("This transaction type does not implement marshal/unmarshal functions")
		}
		if err := transaction.UnmarshalBinary(tx); err != nil {
			return nil, errors.Wrap(err, "Failed to unmarshal transaction")
		}
		return proto.Transaction(transaction), nil
	}
}

func NewBlockManager(genesis crypto.Signature, state AccountsState, rw BlockReadWriter) (*BlockManager, error) {
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

func (s *BlockManager) checkTransaction(block *proto.Block, tx proto.Transaction, initialisation bool) error {
	switch v := tx.(type) {
	case proto.Genesis:
		if block.BlockSignature == s.genesis {
			if !initialisation {
				return errors.New("Trying to add genesis transaction in new block")
			}
			// TODO: what to check here?
			return nil
		} else {
			return errors.New("Tried to add genesis transaction inside of non-genesis block")
		}
	case proto.Payment:
		if !initialisation {
			return errors.New("Trying to add payment transaction in new block")
		}
		// Verify the signature first.
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Failed to verify transaction signature")
		}
		if !ok {
			return errors.New("Invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount < 0 {
			return errors.New("Negative amount in transaction")
		}
		if v.Fee < 0 {
			return errors.New("Negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		totalAmount := v.Fee + v.Amount
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
		if err != nil {
			return err
		}
		wavesAsset, err := proto.NewOptionalAssetFromString(proto.WavesAssetName)
		if err != nil {
			return err
		}
		balance := sender.AssetBalance(wavesAsset)
		if balance < totalAmount {
			return errors.New("Transaction verification failed: spending more than current balance.")
		}
		return nil
	case proto.TransferV1:
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Failed to verify transaction signature")
		}
		if !ok {
			return errors.New("Invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount < 0 {
			return errors.New("Negative amount in transaction")
		}
		if v.Fee < 0 {
			return errors.New("Negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
		if err != nil {
			return err
		}
		feeBalance := sender.AssetBalance(&v.FeeAsset)
		amountBalance := sender.AssetBalance(&v.AmountAsset)
		if amountBalance < v.Amount {
			return errors.New("Invalid transaction: not enough to pay the amount provided")
		}
		if feeBalance < v.Fee {
			return errors.New("Invalid transaction: not eough to pay the fee provided")
		}
		return nil
	case proto.TransferV2:
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Failed to verify transaction signature")
		}
		if !ok {
			return errors.New("Invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount < 0 {
			return errors.New("Negative amount in transaction")
		}
		if v.Fee < 0 {
			return errors.New("Negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
		if err != nil {
			return err
		}
		feeBalance := sender.AssetBalance(&v.FeeAsset)
		amountBalance := sender.AssetBalance(&v.AmountAsset)
		if amountBalance < v.Amount {
			return errors.New("Invalid transaction: not enough to pay the amount provided")
		}
		if feeBalance < v.Fee {
			return errors.New("Invalid transaction: not eough to pay the fee provided")
		}
		return nil
	default:
		return errors.Errorf("Transaction type %T is not supported\n", v)
	}
}

func (s *BlockManager) performTransaction(block *proto.Block, tx proto.Transaction) error {
	wavesAsset, err := proto.NewOptionalAssetFromString(proto.WavesAssetName)
	if err != nil {
		return err
	}
	switch v := tx.(type) {
	case proto.Genesis:
		receiver, err := s.accountsState.Account(proto.NewRecipientFromAddress(v.Recipient))
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
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
		if err != nil {
			return err
		}
		receiver, err := s.accountsState.Account(proto.NewRecipientFromAddress(v.Recipient))
		if err != nil {
			return err
		}
		minerAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, block.GenPublicKey)
		if err != nil {
			return err
		}
		miner, err := s.accountsState.Account(proto.NewRecipientFromAddress(minerAddr))
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
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
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
		miner, err := s.accountsState.Account(proto.NewRecipientFromAddress(minerAddr))
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
		sender, err := s.accountsState.Account(proto.NewRecipientFromAddress(senderAddr))
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
		miner, err := s.accountsState.Account(proto.NewRecipientFromAddress(minerAddr))
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
	// Save block header to storage.
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		return err
	}
	s.rw.WriteBlockHeader(block.BlockSignature, headerBytes)
	transactions := block.Transactions
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transactions[0:4]))
		txBytes := transactions[4 : n+4]
		tx, err := BytesToTransaction(txBytes)
		// Save transaction to storage.
		s.rw.WriteTransaction(tx.GetID(), txBytes)
		if err != nil {
			return err
		}
		if err = s.checkTransaction(block, tx, initialisation); err != nil {
			return errors.Wrap(err, "Incorrect transaction inside of the block")
		}
		if err = s.performTransaction(block, tx); err != nil {
			return errors.Wrap(err, "Failed to perform the transaction")
		}
	}
	return nil
}

func (s *BlockManager) RollbackTo(removalEdge crypto.Signature) error {
	// Remove blocks.
	s.rw.RemoveBlocks(removalEdge)
	// Rollback accounts state.
	return s.accountsState.RollbackTo(removalEdge)
}

package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type AccountsState interface {
	// nil asset means Waves.
	AccountBalance(key []byte) (uint64, error)
}

type TransactionValidator struct {
	genesis crypto.Signature
	state   AccountsState
}

func NewTransactionValidator(genesis crypto.Signature, state AccountsState) (*TransactionValidator, error) {
	return &TransactionValidator{genesis: genesis, state: state}, nil
}

func (tv *TransactionValidator) IsSupported(tx proto.Transaction) bool {
	switch v := tx.(type) {
	case *proto.Genesis:
		return true
	case *proto.Payment:
		return true
	case *proto.TransferV1:
		if v.FeeAsset.Present || v.AmountAsset.Present {
			// Only Waves for now.
			return false
		}
		if v.Recipient.Address == nil {
			// Aliases without specified address are not supported yet.
			return false
		}
		return true
	case *proto.TransferV2:
		if v.FeeAsset.Present || v.AmountAsset.Present {
			// Only Waves for now.
			return false
		}
		if v.Recipient.Address == nil {
			// Aliases without specified address are not supported yet.
			return false
		}
		return true
	default:
		// Other types of transactions are not supported.
		return false
	}
}

func (tv *TransactionValidator) ValidateTransaction(blockID crypto.Signature, tx proto.Transaction, initialisation bool) error {
	switch v := tx.(type) {
	case *proto.Genesis:
		if blockID == tv.genesis {
			if !initialisation {
				return errors.New("trying to add genesis transaction in new block")
			}
			return nil
		} else {
			return errors.New("tried to add genesis transaction inside of non-genesis block")
		}
	case *proto.Payment:
		if !initialisation {
			return errors.New("trying to add payment transaction in new block")
		}
		// Verify the signature first.
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "failed to verify transaction signature")
		}
		if !ok {
			return errors.New("invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount <= 0 {
			return errors.New("negative amount in transaction")
		}
		if v.Fee <= 0 {
			return errors.New("negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		totalAmount := v.Fee + v.Amount
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "could not get address from public key")
		}
		senderKey := BalanceKey{Address: senderAddr}
		balance, err := tv.state.AccountBalance(senderKey.Bytes())
		if err != nil {
			return err
		}
		if balance < totalAmount {
			return errors.Errorf("transaction verification failed: balance is %d, trying to spend %d", balance, totalAmount)
		}
		return nil
	case *proto.TransferV1:
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "failed to verify transaction signature")
		}
		if !ok {
			return errors.New("invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount <= 0 {
			return errors.New("negative amount in transaction")
		}
		if v.Fee <= 0 {
			return errors.New("negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		senderFeeKey := BalanceKey{Address: senderAddr, Asset: v.FeeAsset.ToID()}
		feeBalance, err := tv.state.AccountBalance(senderFeeKey.Bytes())
		if err != nil {
			return err
		}
		senderAmountKey := BalanceKey{Address: senderAddr, Asset: v.AmountAsset.ToID()}
		amountBalance, err := tv.state.AccountBalance(senderAmountKey.Bytes())
		if err != nil {
			return err
		}
		if amountBalance < v.Amount {
			return errors.New("invalid transaction: not enough to pay the amount provided")
		}
		if feeBalance < v.Fee {
			return errors.New("invalid transaction: not eough to pay the fee provided")
		}
		return nil
	case *proto.TransferV2:
		ok, err := v.Verify(v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "failed to verify transaction signature")
		}
		if !ok {
			return errors.New("invalid transaction signature")
		}
		// Check amount and fee lower bound.
		if v.Amount <= 0 {
			return errors.New("negative amount in transaction")
		}
		if v.Fee <= 0 {
			return errors.New("negative fee in transaction")
		}
		// Verify the amount spent (amount and fee upper bound).
		senderAddr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "could not get address from public key")
		}
		senderFeeKey := BalanceKey{Address: senderAddr, Asset: v.FeeAsset.ToID()}
		feeBalance, err := tv.state.AccountBalance(senderFeeKey.Bytes())
		if err != nil {
			return err
		}
		senderAmountKey := BalanceKey{Address: senderAddr, Asset: v.AmountAsset.ToID()}
		amountBalance, err := tv.state.AccountBalance(senderAmountKey.Bytes())
		if err != nil {
			return err
		}
		if amountBalance < v.Amount {
			return errors.New("invalid transaction: not enough to pay the amount provided")
		}
		if feeBalance < v.Fee {
			return errors.New("invalid transaction: not eough to pay the fee provided")
		}
		return nil
	default:
		return errors.Errorf("transaction type %T is not supported\n", v)
	}
}

package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AccountsState interface {
	// nil asset means Waves.
	AccountBalance(addr Address, asset []byte) (uint64, error)
}

type TransactionValidator struct {
	genesis crypto.Signature
	state   AccountsState
}

func NewTransactionValidator(genesis crypto.Signature, state AccountsState) (*TransactionValidator, error) {
	return &TransactionValidator{genesis: genesis, state: state}, nil
}

func (tv *TransactionValidator) IsSupported(tx Transaction) bool {
	switch v := tx.(type) {
	case *Genesis:
		return true
	case *Payment:
		return true
	case *TransferV1:
		if v.FeeAsset.Present || v.AmountAsset.Present {
			// Only Waves for now.
			return false
		}
		if v.Recipient.Address == nil {
			// Aliases without specified address are not supported yet.
			return false
		}
		return true
	case *TransferV2:
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

func (tv *TransactionValidator) ValidateTransaction(blockID crypto.Signature, tx Transaction, initialisation bool) error {
	switch v := tx.(type) {
	case *Genesis:
		if blockID == tv.genesis {
			if !initialisation {
				return errors.New("trying to add genesis transaction in new block")
			}
			return nil
		} else {
			return errors.New("tried to add genesis transaction inside of non-genesis block")
		}
	case *Payment:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "could not get address from public key")
		}
		balance, err := tv.state.AccountBalance(senderAddr, nil)
		if err != nil {
			return err
		}
		if balance < totalAmount {
			return errors.Errorf("transaction verification failed: balance is %d, trying to spend %d", balance, totalAmount)
		}
		return nil
	case *TransferV1:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		feeBalance, err := tv.state.AccountBalance(senderAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		amountBalance, err := tv.state.AccountBalance(senderAddr, v.AmountAsset.ToID())
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
	case *TransferV2:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "could not get address from public key")
		}
		feeBalance, err := tv.state.AccountBalance(senderAddr, v.FeeAsset.ToID())
		if err != nil {
			return err
		}
		amountBalance, err := tv.state.AccountBalance(senderAddr, v.AmountAsset.ToID())
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

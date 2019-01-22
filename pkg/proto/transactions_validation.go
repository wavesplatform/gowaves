package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AccountsState interface {
	Account(Recipient) (AccountManipulator, error)
	SetAccount(AccountManipulator) error
	RollbackTo(crypto.Signature) error
}

type AccountManipulator interface {
	SetAssetBalance(*OptionalAsset, uint64)
	AssetBalance(*OptionalAsset) uint64
	Address() Address
}

type TransactionValidator struct {
	genesis crypto.Signature
	state   AccountsState
}

func NewTransactionValidator(genesis crypto.Signature, state AccountsState) (*TransactionValidator, error) {
	return &TransactionValidator{genesis: genesis, state: state}, nil
}

func (tv *TransactionValidator) ValidateTransaction(block *Block, tx Transaction, initialisation bool) error {
	switch v := tx.(type) {
	case Genesis:
		if block.BlockSignature == tv.genesis {
			if !initialisation {
				return errors.New("Trying to add genesis transaction in new block")
			}
			// TODO: what to check here?
			return nil
		} else {
			return errors.New("Tried to add genesis transaction inside of non-genesis block")
		}
	case Payment:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := tv.state.Account(NewRecipientFromAddress(senderAddr))
		if err != nil {
			return err
		}
		wavesAsset, err := NewOptionalAssetFromString(WavesAssetName)
		if err != nil {
			return err
		}
		balance := sender.AssetBalance(wavesAsset)
		if balance < totalAmount {
			return errors.New("Transaction verification failed: spending more than current balance.")
		}
		return nil
	case TransferV1:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := tv.state.Account(NewRecipientFromAddress(senderAddr))
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
	case TransferV2:
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
		senderAddr, err := NewAddressFromPublicKey(MainNetScheme, v.SenderPK)
		if err != nil {
			return errors.Wrap(err, "Could not get address from public key")
		}
		sender, err := tv.state.Account(NewRecipientFromAddress(senderAddr))
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

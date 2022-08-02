package errors

import (
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type transactionError struct {
	genericError
}

// TODO(nickeskov): IMPLEMENT ME
type order struct{}

type (
	TransactionDoesNotExistError    transactionError
	UnsupportedTransactionTypeError transactionError
	AssetDoesNotExistError          transactionError
	NegativeAmount                  transactionError
	InsufficientFeeError            transactionError
	NegativeMinFeeError             transactionError
	NonPositiveAmountError          transactionError
	AlreadyInStateError             transactionError
	AccountBalanceErrorsError       struct {
		transactionError
		Details map[proto.WavesAddress]string `json:"details"`
	}
	OrderInvalidError struct {
		transactionError
		Order order `json:"order"`
	}
	InvalidChainIdError       transactionError
	InvalidProofsError        transactionError
	InvalidTransactionIdError transactionError
	InvalidBlockIdError       transactionError
	InvalidAssetIdError       transactionError
)

var (
	TransactionDoesNotExist = &TransactionDoesNotExistError{
		genericError: genericError{
			ID:       TransactionDoesNotExistErrorID,
			HttpCode: http.StatusNotFound,
			Message:  "transactions does not exist",
		},
	}
	UnsupportedTransactionType = &UnsupportedTransactionTypeError{
		genericError: genericError{
			ID:       UnsupportedTransactionTypeErrorID,
			HttpCode: http.StatusNotImplemented,
			Message:  "transaction type not supported",
		},
	}
	InvalidAssetId = &InvalidAssetIdError{
		genericError: genericError{
			ID:       InvalidAssetIdErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Invalid asset id",
		},
	}
)

func NewInvalidBlockIDError(message string) *InvalidBlockIdError {
	return &InvalidBlockIdError{
		genericError: genericError{
			ID:       InvalidBlockIdErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  message,
		},
	}
}

func NewInvalidTransactionIDError(message string) *InvalidTransactionIdError {
	return &InvalidTransactionIdError{
		genericError: genericError{
			ID:       InvalidTransactionIdErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  message,
		},
	}
}

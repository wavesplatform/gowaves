package errors

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/crypto"

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
	AssetsDoesNotExistError         struct {
		transactionError
		IDs []string `json:"ids"`
	}
	NegativeAmount            transactionError
	InsufficientFeeError      transactionError
	NegativeMinFeeError       transactionError
	NonPositiveAmountError    transactionError
	AlreadyInStateError       transactionError
	AccountBalanceErrorsError struct {
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
	AssetIdNotSpecifiedError  transactionError
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
	AssetIdNotSpecified = &AssetIdNotSpecifiedError{
		genericError: genericError{
			ID:       AssetIdNotSpecifiedErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Asset ID was not specified",
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

func NewAssetDoesNotExistError(digest crypto.Digest) *AssetDoesNotExistError {
	return &AssetDoesNotExistError{
		genericError: genericError{
			ID:       AssetDoesNotExistErrorID,
			HttpCode: http.StatusNotFound,
			Message:  fmt.Sprintf("Asset does not exist: %s", digest.String()),
		},
	}
}

func NewAssetsDoesNotExistError(ids []string) *AssetsDoesNotExistError {
	return &AssetsDoesNotExistError{
		transactionError: transactionError{
			genericError: genericError{
				ID:       AssetsDoesNotExistErrorID,
				HttpCode: http.StatusNotFound,
				Message:  fmt.Sprintf("Asset does not exist. %s", strings.Join(ids, ", ")),
			},
		},
		IDs: ids,
	}
}

package errors

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type validationError struct {
	genericError
}

// TODO(nickeskov): IMPLEMENT ME
type transaction interface{}

type validationErrorWithTransaction struct {
	validationError
	Transaction transaction `json:"transaction,omitempty"`
}

type (
	InvalidSignatureError validationError
	InvalidAddressError   validationError
	InvalidPublicKeyError validationError
	InvalidMessageError   validationError
	InvalidNameError      validationError
	StateCheckFailedError struct {
		validationErrorWithTransaction
		// TODO(nickeskov): implement more optimized way for fields embedding
		// for converting any structs to map[string]interface{}
		// in a convenient way use "github.com/mitchellh/mapstructure"
		embeddedFields map[string]interface{}
	}
	OverflowError                validationError
	ToSelfError                  validationError
	MissingSenderPrivateKeyError validationError
	InvalidIdsError              struct {
		validationError
		Ids []string `json:"ids"`
	}
	CustomValidationError                     validationError
	BlockDoesNotExistError                    validationError
	AliasDoesNotExistError                    validationError
	MistimingError                            validationError
	DataKeyDoesNotExistError                  validationError
	ScriptCompilerError                       validationError
	ScriptExecutionError                      validationErrorWithTransaction
	TransactionNotAllowedByAccountScriptError validationErrorWithTransaction
)

func (e StateCheckFailedError) MarshalJSON() ([]byte, error) {
	errorJson, err := json.Marshal(e.validationErrorWithTransaction)
	if err != nil {
		return nil, errors.Wrap(err, "StateCheckFailedError.MarshalJSON")
	}

	if len(e.embeddedFields) == 0 {
		return errorJson, nil
	}

	embedded, err := json.Marshal(e.embeddedFields)
	if err != nil {
		return nil, errors.Wrap(err, "StateCheckFailedError.MarshalJSON")
	}

	// `{"extra":"somevalue"}` -> `"extra":"somevalue"`
	extraFields := embedded[1 : len(embedded)-1]

	reservedLen := len(errorJson) + len(extraFields) + 1
	if reservedLen > 1024*1024 {
		return nil, errors.New("too big value, 1MB limit exceeded")
	}

	buffer := make([]byte, 0, reservedLen)

	// errorJson=`{"somekey":"somevalue"}`, buffer = `"{"somekey":"somevalue"`
	buffer = append(buffer, errorJson[:len(errorJson)-1]...)
	// buffer = `"{"somekey":"somevalue",`
	buffer = append(buffer, ',')
	// buffer = `"{"somekey":"somevalue","extra":"somevalue"`
	buffer = append(buffer, extraFields...)
	// buffer = `"{"somekey":"somevalue","extra":"somevalue"}`
	buffer = append(buffer, '}')

	return buffer, nil
}

var (
	InvalidSignature = &InvalidSignatureError{
		genericError: genericError{
			ID:       InvalidSignatureErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "invalid signature",
		},
	}
	InvalidAddress = &InvalidAddressError{
		genericError: genericError{
			ID:       InvalidAddressErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "invalid address",
		},
	}
	InvalidPublicKey = &InvalidPublicKeyError{
		genericError: genericError{
			ID:       InvalidPublicKeyErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "invalid public key",
		},
	}
	InvalidMessage = &InvalidMessageError{
		genericError: genericError{
			ID:       InvalidMessageErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "invalid message",
		},
	}
	InvalidName = &InvalidNameError{
		genericError: genericError{
			ID:       InvalidNameErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "invalid name",
		},
	}
	Overflow = &OverflowError{
		genericError: genericError{
			ID:       OverflowErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "overflow error",
		},
	}
	ToSelf = &ToSelfError{
		genericError: genericError{
			ID:       ToSelfErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Transaction to yourself",
		},
	}
	MissingSenderPrivateKey = &MissingSenderPrivateKeyError{
		genericError: genericError{
			ID:       MissingSenderPrivateKeyErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "no private key for sender address in wallet",
		},
	}
	BlockDoesNotExist = &BlockDoesNotExistError{
		genericError: genericError{
			ID:       BlockDoesNotExistErrorID,
			HttpCode: http.StatusNotFound,
			Message:  "block does not exist",
		},
	}
	DataKeyDoesNotExist = &DataKeyDoesNotExistError{
		genericError: genericError{
			ID:       DataKeyDoesNotExistErrorID,
			HttpCode: http.StatusNotFound,
			Message:  "no data for this key",
		},
	}
)

func NewCustomValidationError(message string) *CustomValidationError {
	return &CustomValidationError{
		genericError: genericError{
			ID:       CustomValidationErrorErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  message,
		},
	}
}

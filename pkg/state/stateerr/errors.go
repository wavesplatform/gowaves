package stateerr

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ErrorType byte

const (
	// DeserializationError indicates a failure to unmarshal data (e.g., block or transaction).
	DeserializationError ErrorType = iota + 1

	// NotFoundError is returned when a requested entity cannot be located.
	NotFoundError

	// TxValidationError indicates a transaction failed validation checks.
	TxValidationError

	// TxCommitmentError indicates a failure while committing a transaction.
	TxCommitmentError

	// ValidationError is a generic validation failure.
	ValidationError

	// RollbackError occurs when rolling back a state change fails.
	RollbackError

	// RetrievalError covers failures when reading data from the database.
	RetrievalError

	// ModificationError covers failures when updating or modifying state data.
	ModificationError

	// InvalidInputError indicates that input data is malformed or otherwise invalid.
	InvalidInputError

	// IncompatibilityError indicates mismatched or incompatible data or versions.
	IncompatibilityError

	// ClosureError indicates a failure to properly close the database or block storage.
	ClosureError

	// Other is used for miscellaneous technical errors that should not normally occur.
	Other
)

type StateError struct {
	errorType     ErrorType
	originalError error
}

func NewStateError(errorType ErrorType, originalError error) StateError {
	return StateError{errorType: errorType, originalError: originalError}
}

func (err StateError) Type() ErrorType {
	return err.errorType
}

func (err StateError) Error() string {
	return err.originalError.Error()
}

func (err StateError) Unwrap() error {
	return err.originalError
}

func IsTxCommitmentError(err error) bool {
	var stateErr StateError
	if errors.As(err, &stateErr) {
		return stateErr.Type() == TxCommitmentError
	}
	return false
}

func IsNotFound(err error) bool {
	var stateErr StateError
	switch {
	case errors.Is(err, proto.ErrNotFound):
		// Special case: sometimes proto.ErrNotFound might be used as well.
		return true
	case errors.Is(err, keyvalue.ErrNotFound):
		// the same as above, but for keyvalue.ErrNotFound
		return true
	case errors.As(err, &stateErr):
		errType := stateErr.Type()
		return (errType == NotFoundError) || (errType == RetrievalError)
	default:
		return false
	}
}

func IsInvalidInput(err error) bool {
	var stateErr StateError
	switch {
	case errors.As(err, &stateErr):
		return stateErr.Type() == InvalidInputError
	default:
		return false
	}
}

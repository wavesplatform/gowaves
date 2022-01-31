package state

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ErrorType byte

const (
	// Unmarshal error (for example, of block or transaction).
	DeserializationError ErrorType = iota + 1
	NotFoundError
	SerializationError
	TxValidationError
	TxCommitmentError
	ValidationError
	RollbackError
	// Errors occurring while getting data from database.
	RetrievalError
	// Errors occurring while updating/modifying state data.
	ModificationError
	InvalidInputError
	IncompatibilityError
	// DB or block storage Close() error.
	ClosureError
	// Minor technical errors which shouldn't ever happen.
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
	switch {
	case err == nil:
		return false
	case errors.As(err, &stateErr):
		return stateErr.Type() == TxCommitmentError
	default:
		return false
	}
}

func IsNotFound(err error) bool {
	var stateErr StateError
	switch {
	case err == nil:
		return false
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
	case err == nil:
		return false
	case errors.As(err, &stateErr):
		return stateErr.Type() == InvalidInputError
	default:
		return false
	}
}

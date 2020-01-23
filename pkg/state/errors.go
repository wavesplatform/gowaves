package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ErrorType byte

const (
	// Unmarshal error of block or transaction.
	DeserializationError ErrorType = iota + 1
	NotFoundError
	SerializationError
	TxValidationError
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

func (err StateError) Error() string {
	return err.originalError.Error()
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if err == proto.ErrNotFound {
		// Special case: sometimes proto.ErrNotFound might be used as well.
		return true
	}
	se, ok := err.(StateError)
	if !ok {
		return false
	}
	return (se.errorType == NotFoundError) || (se.errorType == RetrievalError)
}

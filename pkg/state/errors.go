package state

import "github.com/wavesplatform/gowaves/pkg/keyvalue"

type StateErrorType byte

const (
	// Unmarshal error of block or transaction.
	DeserializationError StateErrorType = iota + 1
	TxValidationError
	BlockValidationError
	RollbackError
	// Errors occurring while getting data from database.
	RetrievalError
	// Errors occurring while updating/modifying state data.
	ModificationError
	InvalidInputError
	// DB or block storage Close() error.
	ClosureError
	// Minor technical errors which shouldn't ever happen.
	Other
)

type StateError struct {
	errorType     StateErrorType
	originalError error
}

func NewStateError(errorType StateErrorType, originalError error) StateError {
	return StateError{errorType: errorType, originalError: originalError}
}

func (err StateError) Error() string {
	return err.originalError.Error()
}

func ErrorType(err error) StateErrorType {
	switch e := err.(type) {
	case StateError:
		return e.errorType
	default:
		return 0
	}
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	s, ok := err.(StateError)
	if !ok {
		return false
	}
	return keyvalue.ErrNotFound == s.originalError
}

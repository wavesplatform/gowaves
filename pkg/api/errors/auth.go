package errors

import (
	"fmt"
	"net/http"
)

// API Auth
type authError struct {
	genericError
}

type (
	APIKeyNotValidError        authError
	APIKeyDisabledError        authError
	TooBigArrayAllocationError authError
)

var (
	ErrAPIKeyNotValid = &APIKeyNotValidError{
		genericError: genericError{
			ID:       APIKeyNotValidErrorID,
			HttpCode: http.StatusForbidden,
			Message:  "Provided API key is not correct",
		},
	}
	ErrAPIKeyDisabled = &APIKeyDisabledError{
		genericError: genericError{
			ID:       APIKeyDisabledErrorID,
			HttpCode: http.StatusForbidden,
			Message:  "API key disabled",
		},
	}
	TooBigArrayAllocation = &TooBigArrayAllocationError{
		genericError: genericError{
			ID:       TooBigArrayAllocationErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Too big sequence requested",
		},
	}
)

func NewTooBigArrayAllocationError(limit int) *TooBigArrayAllocationError {
	return &TooBigArrayAllocationError{
		genericError: genericError{
			ID:       TooBigArrayAllocationErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  fmt.Sprintf("Too big sequence requested: max limit is %d entries", limit),
		},
	}
}

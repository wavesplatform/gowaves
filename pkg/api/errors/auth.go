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
	ApiKeyNotValidError        authError
	TooBigArrayAllocationError authError
)

var (
	ApiKeyNotValid = &ApiKeyNotValidError{
		genericError: genericError{
			ID:       ApiKeyNotValidErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Provided API key is not correct",
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

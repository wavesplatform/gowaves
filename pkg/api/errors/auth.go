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
		ID:       APIKeyNotValidErrorID,
		HttpCode: http.StatusBadRequest,
		Message:  "Provided API key is not correct",
	}
	TooBigArrayAllocation = &TooBigArrayAllocationError{
		ID:       TooBigArrayAllocationErrorID,
		HttpCode: http.StatusBadRequest,
		Message:  "Too big sequence requested",
	}
)

func NewTooBigArrayAllocationError(limit int) *TooBigArrayAllocationError {
	return &TooBigArrayAllocationError{
		ID:       TooBigArrayAllocationErrorID,
		HttpCode: http.StatusBadRequest,
		Message:  fmt.Sprintf("Too big sequence requested: max limit is %d entries", limit),
	}
}

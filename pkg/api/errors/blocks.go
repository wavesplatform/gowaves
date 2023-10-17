package errors

import (
	"fmt"
	"net/http"
)

type blocksError struct {
	genericError
}

type (
	InvalidHeightError   blocksError
	NoBlockAtHeightError blocksError
)

var (
	InvalidHeight = &InvalidHeightError{
		genericError: genericError{
			ID:       InvalidHeightErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "Invalid height",
		},
	}
)

func NewNoBlockAtHeightError(inner error) *NoBlockAtHeightError {
	return &NoBlockAtHeightError{
		genericError: genericError{
			ID:       NoBlockAtHeightErrorID,
			HttpCode: http.StatusNotFound,
			Message:  fmt.Sprintf("No block at height: %v", inner),
		},
	}
}

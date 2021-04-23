package types

import (
	"fmt"
	"net/http"
)

const (
	UnknownApiErrorID   = 0
	WrongJsonApiErrorID = 1
)

type genericApiError struct {
	ID       int    `json:"error"`
	HttpCode int    `json:"-"`
	Message  string `json:"message"`
}

func (g *genericApiError) Error() string {
	return fmt.Sprintf("ApiError #%d: %s", g.ID, g.Message)
}

type UnknownApiError struct {
	genericApiError
	Err error
}

func (u *UnknownApiError) Cause() error {
	return u.Err
}

type WrongJsonApiError struct {
	genericApiError
}

type AuthApiError struct {
	genericApiError
}

type ValidationApiError struct {
	genericApiError
}

type TransactionsApiError struct {
	genericApiError
}

var (
	UnknownApiErrorDefault   = NewUnknownApiError("Error is unknown", nil)
	WrongJsonApiErrorDefault = WrongJsonApiError{
		genericApiError: genericApiError{
			ID:       WrongJsonApiErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "failed to parse json message",
		},
	}
)

func NewUnknownApiError(message string, inner error) UnknownApiError {
	return UnknownApiError{
		genericApiError: genericApiError{
			ID:       UnknownApiErrorID,
			HttpCode: http.StatusInternalServerError,
			Message:  message,
		},
		Err: inner,
	}
}

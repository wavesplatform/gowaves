package errors

import (
	"fmt"
	"net/http"
)

type Identifier interface {
	IntCode() int
}

// ApiError is a basic interface for node HTTP API.
// Type which implements this interface MUST be serializable to JSON.
type ApiError interface {
	error
	GetID() Identifier
	GetName() string
	GetHttpCode() int
	GetMessage() string
}

type ErrorID int
type ApiAuthErrorID ErrorID
type ValidationErrorID ErrorID
type TransactionErrorID ErrorID

func (e ErrorID) IntCode() int {
	return int(e)
}
func (e ApiAuthErrorID) IntCode() int {
	return int(e)
}
func (e ValidationErrorID) IntCode() int {
	return int(e)
}
func (e TransactionErrorID) IntCode() int {
	return int(e)
}

// generic error

type genericError struct {
	ID       Identifier `json:"error"`
	HttpCode int        `json:"-"`
	Message  string     `json:"message"`
}

func (g *genericError) GetID() Identifier {
	return g.ID
}

func (g *genericError) GetName() string {
	return errorNames[g.ID]
}

func (g *genericError) GetHttpCode() int {
	return g.HttpCode
}

func (g *genericError) GetMessage() string {
	return g.Message
}

func (g *genericError) Error() string {
	return fmt.Sprintf("%s #%d: %s", g.GetName(), g.ID.IntCode(), g.Message)
}

// --generic error

// UnknownError is a wrapper for any unknown internal error
type UnknownError struct {
	genericError
	inner error
}

func (u *UnknownError) Unwrap() error {
	return u.inner
}

func (u *UnknownError) Error() string {
	if u.Unwrap() != nil {
		return fmt.Sprintf(
			"%s; inner error (%T): %s",
			u.genericError.Error(),
			u.Unwrap(), u.Unwrap().Error(),
		)
	}
	return u.genericError.Error()
}

func NewUnknownError(inner error) *UnknownError {
	return NewUnknownErrorWithMsg("Error is unknown", inner)
}

func NewUnknownErrorWithMsg(message string, inner error) *UnknownError {
	return &UnknownError{
		genericError: genericError{
			ID:       UnknownErrorID,
			HttpCode: http.StatusInternalServerError,
			Message:  message,
		},
		inner: inner,
	}
}

// --UnknownError

type WrongJsonError struct {
	genericError
	Cause            string  `json:"cause,omitempty"`
	ValidationErrors []error `json:"validationErrors,omitempty"`
}

func NewWrongJsonError(cause string, validationErrors []error) *WrongJsonError {
	return &WrongJsonError{
		genericError: genericError{
			ID:       WrongJSONErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  "failed to parse json message",
		},
		Cause:            cause,
		ValidationErrors: validationErrors,
	}
}

// UnavailableError UnknownError is a wrapper for any error related to service unavailability.
type UnavailableError struct {
	genericError
	inner error
}

func (u *UnavailableError) Unwrap() error {
	return u.inner
}

func (u *UnavailableError) Error() string {
	if u.Unwrap() != nil {
		return fmt.Sprintf(
			"%s; inner error (%T): %s",
			u.genericError.Error(),
			u.Unwrap(), u.Unwrap().Error(),
		)
	}
	return u.genericError.Error()
}

func NewUnavailableError(inner error) *UnavailableError {
	return NewUnavailableErrorWithMsg("Service is unavailable", inner)
}

func NewUnavailableErrorWithMsg(message string, inner error) *UnavailableError {
	return &UnavailableError{
		genericError: genericError{
			ID:       ServiceUnavailableErrorID,
			HttpCode: http.StatusServiceUnavailable,
			Message:  message,
		},
		inner: inner,
	}
}

// BadRequestError is a wrapper for any bad request error.
type BadRequestError struct {
	genericError
	inner error
}

func (u *BadRequestError) Unwrap() error {
	return u.inner
}

func (u *BadRequestError) Error() string {
	if u.Unwrap() != nil {
		return fmt.Sprintf(
			"%s; inner error (%T): %s",
			u.genericError.Error(),
			u.Unwrap(), u.Unwrap().Error(),
		)
	}
	return u.genericError.Error()
}

func NewBadRequestError(inner error) *BadRequestError {
	return NewBadRequestErrorWithMsg("Bad request", inner)
}

func NewBadRequestErrorWithMsg(message string, inner error) *BadRequestError {
	return &BadRequestError{
		genericError: genericError{
			ID:       BadRequestErrorID,
			HttpCode: http.StatusBadRequest,
			Message:  message,
		},
		inner: inner,
	}
}

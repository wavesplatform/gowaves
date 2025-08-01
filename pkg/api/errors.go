package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/errors"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/logging"
)

// internal node api errors
var (
	notFound = errors.New("not found")
)

// BadRequestError represents a bad request error.
// Deprecated: don't use this error type in new code. Create a new error type or value in 'pkg/api/errors' package.
type BadRequestError struct {
	inner error
}

func wrapToBadRequestError(err error) *BadRequestError {
	return &BadRequestError{inner: err}
}

func (e *BadRequestError) Error() string {
	return e.inner.Error()
}

// AuthError represents an authentication error or problem.
// Deprecated: don't use this error type in new code. Create a new error type or value in 'pkg/api/errors' package.
type AuthError struct {
	inner error
}

func wrapToAuthError(err error) *AuthError {
	return &AuthError{inner: err}
}

func (e *AuthError) Error() string {
	return e.inner.Error()
}

type ErrorHandler struct {
	logger *slog.Logger
}

func NewErrorHandler(logger *slog.Logger) ErrorHandler {
	return ErrorHandler{
		logger: logger,
	}
}

func (eh *ErrorHandler) Handle(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	// target errors
	var (
		badRequestError *BadRequestError
		authError       *AuthError
		unknownError    *apiErrs.UnknownError
		apiError        apiErrs.ApiError
		// check that all targets implement the error interface
		_, _, _, _ = error(badRequestError), error(authError), error(unknownError), error(apiError)
	)
	switch {
	case errors.As(err, &badRequestError):
		// nickeskov: this error type will be removed in future
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", badRequestError.Error()), http.StatusBadRequest)
	case errors.As(err, &authError):
		// nickeskov: this error type will be removed in future
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", authError.Error()), http.StatusForbidden)
	case errors.As(err, &unknownError):
		eh.logger.Error("UnknownError",
			slog.String("proto", r.Proto),
			slog.String("path", r.URL.Path),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("remote_addr", r.RemoteAddr),
			logging.Error(err))
		eh.sendApiErrJSON(w, r, unknownError)
	case errors.As(err, &apiError):
		eh.sendApiErrJSON(w, r, apiError)
	default:
		eh.logger.Error("InternalServerError",
			slog.String("proto", r.Proto),
			slog.String("path", r.URL.Path),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("remote_addr", r.RemoteAddr),
			logging.Error(err))
		unknownErrWrapper := apiErrs.NewUnknownError(err)
		eh.sendApiErrJSON(w, r, unknownErrWrapper)
	}
}

func (eh *ErrorHandler) sendApiErrJSON(w http.ResponseWriter, r *http.Request, apiErr apiErrs.ApiError) {
	w.WriteHeader(apiErr.GetHttpCode())
	if encodeErr := json.NewEncoder(w).Encode(apiErr); encodeErr != nil {
		eh.logger.Error("Failed to marshal API Error to JSON",
			slog.String("proto", r.Proto),
			slog.String("path", r.URL.Path),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("remote_addr", r.RemoteAddr),
			logging.Error(encodeErr),
			slog.String("api_error", apiErr.Error()),
		)
		// nickeskov: Type which implements ApiError interface MUST be serializable to JSON.
		panic(errors.Errorf("BUG, CREATE REPORT: %s", encodeErr.Error()))
	}
}

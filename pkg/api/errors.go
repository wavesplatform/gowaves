package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"go.uber.org/zap"
)

// internal node api errors
var (
	notFound = errors.New("not found")
)

type BadRequestError struct {
	error
}

type AuthError struct {
	error
}

type ErrorHandler struct {
	logger *zap.Logger
}

func NewErrorHandler(logger *zap.Logger) ErrorHandler {
	return ErrorHandler{
		logger: logger,
	}
}

func (eh *ErrorHandler) Handle(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	switch innerErr := errors.Cause(err).(type) {
	case BadRequestError, *BadRequestError:
		// nickeskov: this error type will be removed in future
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", innerErr.Error()), http.StatusBadRequest)
	case AuthError, *AuthError:
		// nickeskov: this error type will be removed in future
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", innerErr.Error()), http.StatusForbidden)
	case *apiErrs.UnknownError:
		eh.logger.Error("UnknownError",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("reqId", middleware.GetReqID(r.Context())),
			zap.Error(err),
		)
		eh.sendApiErrJSON(w, r, innerErr)
	case apiErrs.ApiError:
		eh.sendApiErrJSON(w, r, innerErr)
	default:
		eh.logger.Error("InternalServerError",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("reqId", middleware.GetReqID(r.Context())),
			zap.Error(err),
		)
		unknownErrWrapper := apiErrs.NewUnknownError(innerErr)
		eh.sendApiErrJSON(w, r, unknownErrWrapper)
	}
}

func (eh *ErrorHandler) sendApiErrJSON(w http.ResponseWriter, r *http.Request, apiErr apiErrs.ApiError) {
	w.WriteHeader(apiErr.GetHttpCode())
	if encodeErr := json.NewEncoder(w).Encode(apiErr); encodeErr != nil {
		eh.logger.Error("Failed to marshal API Error to JSON",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("request_id", middleware.GetReqID(r.Context())),
			zap.Error(encodeErr),
			zap.String("api_error", apiErr.Error()),
		)
		// nickeskov: Type which implements ApiError interface MUST be serializable to JSON.
		panic(errors.Errorf("BUG, CREATE REPORT: %s", encodeErr.Error()))
	}
}

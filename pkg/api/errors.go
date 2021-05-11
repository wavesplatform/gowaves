package api

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"go.uber.org/zap"
	"net/http"
)

type BadRequestError struct {
	error
}

type AuthError struct {
	error
}

type InternalError struct {
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
		// TODO(nickeskov): remove this
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", innerErr.Error()), http.StatusForbidden)
	case AuthError, *AuthError:
		// TODO(nickeskov): remove this
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", innerErr.Error()), http.StatusBadRequest)
	case InternalError, *InternalError:
		// TODO(nickeskov): remove this
		eh.logger.Error("LegacyInternalError",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("reqId", middleware.GetReqID(r.Context())),
			zap.Error(err),
		)
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", innerErr.Error()), http.StatusInternalServerError)
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
		panic(errors.Errorf("BUG, CREATE REPORT: %s", encodeErr.Error()))
	}
}

package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
)

// internal node api errors
var (
	notFound = errors.New("not found")
)

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
	// target errors
	var (
		unknownError = &apiErrs.UnknownError{}
		apiError     = apiErrs.ApiError(nil)
		// check that all targets implement the error interface
		_, _ = error(unknownError), error(apiError)
	)
	switch {
	case errors.As(err, &unknownError):
		eh.logger.Error("UnknownError",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("request_id", middleware.GetReqID(r.Context())),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Error(err),
		)
		eh.sendApiErrJSON(w, r, unknownError)
	case errors.As(err, &apiError):
		eh.sendApiErrJSON(w, r, apiError)
	default:
		eh.logger.Error("InternalServerError",
			zap.String("proto", r.Proto),
			zap.String("path", r.URL.Path),
			zap.String("request_id", middleware.GetReqID(r.Context())),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Error(err),
		)
		unknownErrWrapper := apiErrs.NewUnknownError(err)
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
			zap.String("remote_addr", r.RemoteAddr),
			zap.Error(encodeErr),
			zap.String("api_error", apiErr.Error()),
		)
		// nickeskov: Type which implements ApiError interface MUST be serializable to JSON.
		panic(errors.Errorf("BUG, CREATE REPORT: %s", encodeErr.Error()))
	}
}

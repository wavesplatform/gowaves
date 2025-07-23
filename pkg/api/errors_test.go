package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
)

func TestErrorHandler_Handle(t *testing.T) {
	var (
		mustJSON = func(err error) string {
			data, err := json.Marshal(err)
			require.NoError(t, err)
			return string(data)
		}
		badReqErr  = &BadRequestError{errors.New("bad-request")}
		unknownErr = apiErrs.NewUnknownError(errors.New("unknown"))
		defaultErr = errors.New("default")
	)
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedBody string
	}{
		{
			name:         "BadRequestErrorCase",
			err:          errors.WithStack(errors.WithStack(badReqErr)),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to complete request: bad-request\n",
		},
		{
			name:         "ErrorWithMultipleWraps",
			err:          errors.Wrap(errors.Wrap(badReqErr, "wrap1"), "wrap2"),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to complete request: bad-request\n",
		},
		{
			name:         "AuthErrorCase",
			err:          &AuthError{errors.New("auth")},
			expectedCode: http.StatusForbidden,
			expectedBody: "Failed to complete request: auth\n",
		},
		{
			name:         "ApiErrorCase",
			err:          apiErrs.InvalidAddress,
			expectedCode: apiErrs.InvalidAddress.GetHttpCode(),
			expectedBody: mustJSON(apiErrs.InvalidAddress) + "\n",
		},
		{
			name:         "UnknownErrorCase",
			err:          unknownErr,
			expectedCode: unknownErr.GetHttpCode(),
			expectedBody: mustJSON(unknownErr) + "\n",
		},
		{
			name:         "DefaultCase",
			err:          defaultErr,
			expectedCode: http.StatusInternalServerError,
			expectedBody: mustJSON(apiErrs.NewUnknownError(defaultErr)) + "\n",
		},
		{
			name:         "NilCase",
			err:          nil,
			expectedCode: 200,
			expectedBody: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
			)
			h := NewErrorHandler(slog.New(slog.DiscardHandler))
			h.Handle(w, r, test.err)
			assert.Equal(t, test.expectedCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}

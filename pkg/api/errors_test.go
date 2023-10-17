package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
)

func TestErrorHandler_Handle(t *testing.T) {
	var (
		mustJSON = func(err error) string {
			data, err := json.Marshal(err)
			require.NoError(t, err)
			return string(data)
		}
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
			name:         "AuthErrorCase",
			err:          apiErrs.APIKeyDisabled,
			expectedCode: http.StatusForbidden,
			expectedBody: "API key disabled\n",
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
			h := NewErrorHandler(zap.NewNop())
			h.Handle(w, r, test.err)
			assert.Equal(t, test.expectedCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}

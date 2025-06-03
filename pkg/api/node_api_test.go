package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

const apiKey = "X-API-Key"

type walletLoadKeysTest struct {
	apiKey   string
	password []byte
}

func (a *walletLoadKeysTest) LoadKeys(apiKey string, password []byte) error {
	a.apiKey = apiKey
	a.password = password
	return nil
}

func TestWalletLoadKeys(t *testing.T) {
	r := &walletLoadKeysTest{}
	f := WalletLoadKeys(r)

	req := httptest.NewRequest("POST", "/wallet/load", strings.NewReader(`{"password": "password"}`))
	req.Header.Add(apiKey, "apikey")
	resp := httptest.NewRecorder()
	err := f(resp, req)
	assert.NoError(t, err)

	assert.Equal(t, "apikey", r.apiKey)
	assert.EqualValues(t, "password", r.password)
}

func TestNodeApi_FindFirstInvalidRuneInBase58String(t *testing.T) {
	invalidData := []struct {
		str       string
		isInvalid bool
		expected  rune
	}{
		{"234234😀$32@", true, '😀'},
		{"234234$32@", true, '$'},
		{"2@3423432", true, '@'},
		{"42354", false, 0},
	}

	for _, testCase := range invalidData {
		actual, isInvalid := findFirstInvalidRuneInBase58String(testCase.str)
		assert.Equal(t, testCase.isInvalid, isInvalid)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestNodeApi_WavesRegularBalanceByAddress(t *testing.T) {
	createRequest := func(addrStr string) *http.Request {
		chiCtx := chi.NewRouteContext()
		chiCtx.URLParams.Add("address", addrStr)

		req := httptest.NewRequest("", "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
		return req
	}

	t.Run("success", func(t *testing.T) {
		const (
			addrStr = "3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7"
			balance = 1000
		)
		addr, err := proto.NewAddressFromString(addrStr)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		st := mock.NewMockState(ctrl)
		st.EXPECT().WavesBalance(proto.NewRecipientFromAddress(addr)).Return(uint64(1000), nil).Times(1)

		resp := httptest.NewRecorder()
		req := createRequest(addrStr)

		a, err := NewApp("", nil, services.Services{
			State:  st,
			Scheme: proto.TestNetScheme,
		})
		require.NoError(t, err)

		aErr := NewNodeAPI(a, nil).WavesRegularBalanceByAddress(resp, req)
		require.NoError(t, aErr)

		exp := WavesRegularBalance{
			Address:       addr,
			Balance:       balance,
			Confirmations: 0,
		}
		expJS, err := json.Marshal(exp)
		require.NoError(t, err)

		assert.JSONEq(t, string(expJS), resp.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		doTest := func(t *testing.T, addrStr string) {
			ctrl := gomock.NewController(t)

			resp := httptest.NewRecorder()
			req := createRequest(addrStr)

			a, err := NewApp("", nil, services.Services{
				State:  mock.NewMockState(ctrl),
				Scheme: proto.TestNetScheme,
			})
			require.NoError(t, err)

			aErr := NewNodeAPI(a, nil).WavesRegularBalanceByAddress(resp, req)
			assert.Error(t, aErr)
			assert.ErrorIs(t, aErr, apiErrs.InvalidAddress)
		}

		t.Run("empty-address", func(t *testing.T) {
			doTest(t, "")
		})
		t.Run("invalid-address", func(t *testing.T) {
			doTest(t, "3Myqjf1D44wR8Vko4Tr5CwSzRNo2Vg9S7u7$32@")
		})
		t.Run("invalid-address-scheme", func(t *testing.T) {
			const mainnetAddr = "3PQ9hZ36dyXGcqabcrHXsjP9PaQMqy69yeE"
			doTest(t, mainnetAddr)
		})
	})
}

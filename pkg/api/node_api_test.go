package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
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

	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{
			GenerationPeriod: 0,
		},
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
		}, cfg)
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
			}, cfg)
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

func TestNodeApi_TransactionSignCommitToGeneration(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mock.NewMockState(ctrl)
	st.EXPECT().IsActivated(int16(settings.DeterministicFinality)).Return(true, nil)
	st.EXPECT().Height().Return(proto.Height(252), nil)
	st.EXPECT().ActivationHeight(int16(settings.DeterministicFinality)).Return(proto.Height(1), nil)

	w := newTestWallet(t)

	cfg := &settings.BlockchainSettings{
		FunctionalitySettings: settings.FunctionalitySettings{GenerationPeriod: 100},
	}

	app, err := NewApp("", nil, services.Services{
		State:  st,
		Scheme: proto.MainNetScheme,
		Wallet: w,
	}, cfg)
	require.NoError(t, err)

	api := NewNodeAPI(app, st)

	body := `{"periodHeight":252,"type":19,"sender":"3JbGqxNqwBfwnCbzLbo4HwjA9NR1wDjrRTr","version":1}`
	req := httptest.NewRequest(http.MethodPost, "/transactions/sign", strings.NewReader(body))
	resp := httptest.NewRecorder()

	aErr := api.transactionSign(resp, req)
	require.NoError(t, aErr)
	assert.Equal(t, http.StatusOK, resp.Code)

	var signed proto.CommitToGenerationWithProofs
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &signed))

	assert.Equal(t, proto.CommitToGenerationTransaction, signed.Type)
	assert.EqualValues(t, 1, signed.Version)
	assert.Equal(t, w.blsPk, signed.EndorserPublicKey)

	expectedPeriodStart, err := state.CurrentGenerationPeriodStart(1, 252, cfg.GenerationPeriod)
	require.NoError(t, err)
	assert.Equal(t, expectedPeriodStart, signed.GenerationPeriodStart)
}

type testWallet struct {
	pk    crypto.PublicKey
	blsPk bls.PublicKey
	blsSk bls.SecretKey
}

func newTestWallet(t *testing.T) *testWallet {
	t.Helper()

	_, pk, err := crypto.GenerateKeyPair([]byte("commit-wallet"))
	require.NoError(t, err)

	blsSk, err := bls.GenerateSecretKey([]byte("commit-wallet"))
	require.NoError(t, err)

	blsPk, err := blsSk.PublicKey()
	require.NoError(t, err)

	return &testWallet{
		pk:    pk,
		blsPk: blsPk,
		blsSk: blsSk,
	}
}

func (w *testWallet) SignTransactionWith(_ crypto.PublicKey, _ proto.Transaction) error {
	return nil
}

func (w *testWallet) FindPublicKeyByAddress(_ proto.WavesAddress, _ proto.Scheme) (crypto.PublicKey, error) {
	return w.pk, nil
}

func (w *testWallet) BLSPairByWavesPK(_ crypto.PublicKey) (bls.SecretKey, bls.PublicKey, error) {
	return w.blsSk, w.blsPk, nil
}

func (w *testWallet) Load(_ []byte) error {
	return nil
}

func (w *testWallet) AccountSeeds() [][]byte {
	return nil
}

func (w *testWallet) KeyPairsBLS() ([]bls.PublicKey, []bls.SecretKey, error) {
	return []bls.PublicKey{w.blsPk}, []bls.SecretKey{w.blsSk}, nil
}

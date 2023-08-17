package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var addressesBalanceDetailsJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "regular": 37983028703592,
  "generating": 70035776023130,
  "available": 37983028700423,
  "effective": 70035887443130
}`

func TestAddresses_BalanceDetails(t *testing.T) {
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressesBalanceDetailsJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.BalanceDetails(context.Background(), address)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesBalanceDetails{
		Address:    address,
		Regular:    37983028703592,
		Generating: 70035776023130,
		Available:  37983028700423,
		Effective:  70035887443130,
	}, body)
	assert.Equal(t,
		"https://testnode1.wavesnodes.com/addresses/balance/details/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
		resp.Request.URL.String())
}

var addressesScriptInfoJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "complexity": 0,
  "extraFee": 0
}`

func TestAddresses_ScriptInfo(t *testing.T) {
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressesScriptInfoJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.ScriptInfo(context.Background(), address)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.Equal(t, &AddressesScriptInfo{Address: address}, body)
	assert.Equal(t,
		"https://testnode1.wavesnodes.com/addresses/scriptInfo/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
		resp.Request.URL.String())
}

var addressesAddressesJson = `
[
  "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp"
]
`

func TestAddresses_Addresses(t *testing.T) {
	address, _ := proto.NewAddressFromString("3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressesAddressesJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.Addresses(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []proto.WavesAddress{address}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/addresses", resp.Request.URL.String())
}

var addressesValidateJson = `
{
  "address": "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS",
  "valid": true
}`

func TestAddresses_Validate(t *testing.T) {
	address, _ := proto.NewAddressFromString("3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressesValidateJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.Validate(context.Background(), address)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesValidate{
		Address: address,
		Valid:   true,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/addresses/validate/3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS", resp.Request.URL.String())
}

var addressesBalanceJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "confirmations": 0,
  "balance": 37983033403592
}`

func TestAddresses_Balance(t *testing.T) {
	address, _ := proto.NewAddressFromString("3MTsJTRzVZ6bmJ5dh4sp1U3Dr5iQmVtZ6Em")
	client := client(t, NewMockHttpRequestFromString(addressesBalanceJson, 200))
	body, resp, err :=
		client.Addresses.Balance(context.Background(), address)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &AddressesBalance{}, body)
	assert.Contains(t, resp.Request.URL.String(), "/addresses/balance/"+address.String())
}

var addressesEffectiveBalance = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "confirmations": 10,
  "balance": 70035901443130
}`

func TestAddresses_EffectiveBalance(t *testing.T) {
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressesEffectiveBalance, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.EffectiveBalance(context.Background(), address)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesEffectiveBalance{
		Address:       address,
		Balance:       70035901443130,
		Confirmations: 10,
	}, body)
	assert.Equal(t, address, body.Address)
	assert.Equal(t, "https://testnode1.wavesnodes.com/addresses/effectiveBalance/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8", resp.Request.URL.String())
}

var addressPublicKeyJson = `
{
"address": "3Mr5af3Y7r7gQej3tRtugYbKaPr5qYps2ei"
}`

func TestAddresses_PublicKey(t *testing.T) {
	pubKey := "AF9HLq2Rsv2fVfLPtsWxT7Y3S9ZTv6Mw4ZTp8K8LNdEp"
	address, _ := proto.NewAddressFromString("3Mr5af3Y7r7gQej3tRtugYbKaPr5qYps2ei")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressPublicKeyJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.PublicKey(context.Background(), pubKey)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &address, body)
	assert.Equal(t,
		"https://testnode1.wavesnodes.com/addresses/publicKey/AF9HLq2Rsv2fVfLPtsWxT7Y3S9ZTv6Mw4ZTp8K8LNdEp",
		resp.Request.URL.String())
}

var addressSignTextJson = `
{
  "message": "some-text",
  "publicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "signature": "RP742bUjfrzWcXhnmkbim2dWk9mSUcPcmn77EcsD5t2TBUZqZGe8Vk211hAYbW4FNxWtWqcCmR1Trv8gUXKN6if"
}`

func TestAddresses_SignText(t *testing.T) {
	address, _ := proto.NewAddressFromString("3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp")
	pubKey, _ := crypto.NewPublicKeyFromBase58("CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw")
	sign, _ := crypto.NewSignatureFromBase58("RP742bUjfrzWcXhnmkbim2dWk9mSUcPcmn77EcsD5t2TBUZqZGe8Vk211hAYbW4FNxWtWqcCmR1Trv8gUXKN6if")
	text := "some-text"
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		ApiKey:  "ApiKey",
		Client:  NewMockHttpRequestFromString(addressSignTextJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.SignText(context.Background(), address, text)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, text, body.Message)
	assert.Equal(t, &AddressesSignText{
		Message:   text,
		PublicKey: pubKey,
		Signature: sign,
	}, body)
	assert.Equal(t,
		"https://testnode1.wavesnodes.com/addresses/signText/3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp",
		resp.Request.URL.String())
}

var addressVerifyTextJson = `
{
  "valid": true
}`

func TestAddresses_VerifyText(t *testing.T) {
	address, _ := proto.NewAddressFromString("3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp")
	pubKey, _ := crypto.NewPublicKeyFromBase58("J26nL27BBmTgCRye1MdzkFduFDE2aA4agCcuJUyDR2sZ")
	sign, _ := crypto.NewSignatureFromBase58("4Bh3vksvhe55Ej8bwt42HiPgU18MynnKg87Rr1ZhRQUhmJmFiWC7imgaorW5QJRXxXwbK38bvRmZH4dncPzA9grA")
	data := VerifyTextReq{
		Message:   "text",
		PublicKey: pubKey,
		Signature: sign,
	}
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		ApiKey:  "apiKey",
		Client:  NewMockHttpRequestFromString(addressVerifyTextJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.VerifyText(context.Background(), address, data)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/addresses/verifyText/3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp", resp.Request.URL.String())
}

var addressBalanceAfterConfirmationsJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "confirmations": 1,
  "balance": 37983102983592
}`

func TestAddresses_BalanceAfterConfirmations(t *testing.T) {
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	confirmations := uint64(1)
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com/",
		Client:  NewMockHttpRequestFromString(addressBalanceAfterConfirmationsJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Addresses.BalanceAfterConfirmations(context.Background(), address, confirmations)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.Equal(t, confirmations, body.Confirmations)
	assert.EqualValues(t, 37983102983592, body.Balance)
	assert.Equal(t, "https://testnode1.wavesnodes.com/addresses/balance/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8/1", resp.Request.URL.String())
}

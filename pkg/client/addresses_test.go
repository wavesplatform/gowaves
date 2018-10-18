package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestAddresses_BalanceDetails(t *testing.T) {
	address := "3P7qLRU2EZ1BfU3gt2jv6enrEiJ1gQbaWVL"
	client := mustClient()
	body, resp, err :=
		client.Addresses.BalanceDetails(context.Background(), address)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.IsType(t, &AddressesBalanceDetails{}, body)

	bad := mustClient()
	body, resp, err =
		bad.Addresses.BalanceDetails(context.Background(), "3P7qLRU2EZ1BfU3gt2jv6enrE++1gQbaWVL")
	assert.NotNil(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Nil(t, body)
}

func TestAddresses_ScriptInfo(t *testing.T) {
	address := "3P7qLRU2EZ1BfU3gt2jv6enrEiJ1gQbaWVL"
	client := mustClient()
	body, resp, err :=
		client.Addresses.ScriptInfo(context.Background(), address)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.IsType(t, &AddressesScriptInfo{}, body)
}

func TestAddresses_Addresses(t *testing.T) {
	client := mustClient()
	body, resp, err :=
		client.Addresses.Addresses(context.Background())

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, []string{}, body)

}

func TestAddresses_Validate(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := mustClient()
	body, resp, err :=
		client.Addresses.Validate(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesValidate{
		Address: address,
		Valid:   true,
	}, body)
}

func TestAddresses_Balance(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := mustClient()
	body, resp, err :=
		client.Addresses.Balance(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &AddressesBalance{}, body)
}

func TestAddresses_EffectiveBalance(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := mustClient()
	body, resp, err :=
		client.Addresses.EffectiveBalance(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &AddressesEffectiveBalance{}, body)
	assert.Equal(t, address, body.Address)
}

func TestAddresses_PublicKey(t *testing.T) {
	pubKey := "AF9HLq2Rsv2fVfLPtsWxT7Y3S9ZTv6Mw4ZTp8K8LNdEp"
	client := mustClient()
	body, resp, err :=
		client.Addresses.PublicKey(context.Background(), pubKey)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesPublicKey{Address: "3P46PcNRyyf5372U9W9udzy8wHMrfgTxdqT"}, body)
}

func TestAddresses_SignText(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	text := "some-text"
	client := mustClient(Options{BaseUrl: "https://testnode1.wavesnodes.com", ApiKey: apiKey})
	body, resp, err :=
		client.Addresses.SignText(context.Background(), "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp", text)

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, text, body.Message)
	assert.IsType(t, &AddressesSignText{}, body)
}

func TestAddresses_VerifyText(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	data := VerifyTextReq{
		Message:   "text",
		PublicKey: "J26nL27BBmTgCRye1MdzkFduFDE2aA4agCcuJUyDR2sZ",
		Signature: "4Bh3vksvhe55Ej8bwt42HiPgU18MynnKg87Rr1ZhRQUhmJmFiWC7imgaorW5QJRXxXwbK38bvRmZH4dncPzA9grA",
	}

	client := mustClient(Options{BaseUrl: "https://testnode1.wavesnodes.com", ApiKey: apiKey})
	body, resp, err :=
		client.Addresses.VerifyText(context.Background(), "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp", data)

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.True(t, body)
}

func TestAddresses_BalanceAfterConfirmations(t *testing.T) {
	address := "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp"
	confirmations := uint64(1)

	client := mustClient(Options{BaseUrl: "https://testnode1.wavesnodes.com"})
	body, resp, err :=
		client.Addresses.BalanceAfterConfirmations(context.Background(), "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp", confirmations)

	if err != nil {
		t.Fatalf("expected nil, found %+v", err)
	}
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.Equal(t, confirmations, body.Confirmations)
}

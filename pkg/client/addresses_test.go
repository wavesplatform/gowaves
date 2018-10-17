package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient_GetAddressesBalanceDetails(t *testing.T) {
	address := "3P7qLRU2EZ1BfU3gt2jv6enrEiJ1gQbaWVL"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesBalanceDetails(context.Background(), address)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.IsType(t, &AddressesBalanceDetails{}, body)

	bad := NewClient()
	body, resp, err =
		bad.GetAddressesBalanceDetails(context.Background(), "3P7qLRU2EZ1BfU3gt2jv6enrE++1gQbaWVL")
	assert.NotNil(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Nil(t, body)
}

func TestClient_GetAddressesScriptInfo(t *testing.T) {
	address := "3P7qLRU2EZ1BfU3gt2jv6enrEiJ1gQbaWVL"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesScriptInfo(context.Background(), address)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.IsType(t, &AddressesScriptInfo{}, body)
}

func TestClient_GetAddresses(t *testing.T) {
	client := NewClient()
	body, resp, err :=
		client.GetAddresses(context.Background())

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, []string{}, body)

}

func TestClient_GetAddressesValidate(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesValidate(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesValidate{
		Address: address,
		Valid:   true,
	}, body)
}

func TestClient_GetAddressesBalance(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesBalance(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &AddressesBalance{}, body)
}

func TestClient_GetAddressesEffectiveBalance(t *testing.T) {
	address := "3P3oWUH9oXRqiByBG7g9hYSDpCFxcT2wTBS"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesEffectiveBalance(context.Background(), address)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.IsType(t, &AddressesEffectiveBalance{}, body)
	assert.Equal(t, address, body.Address)
}

func TestClient_GetAddressesPublicKey(t *testing.T) {
	pubKey := "AF9HLq2Rsv2fVfLPtsWxT7Y3S9ZTv6Mw4ZTp8K8LNdEp"
	client := NewClient()
	body, resp, err :=
		client.GetAddressesPublicKey(context.Background(), pubKey)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AddressesPublicKey{Address: "3P46PcNRyyf5372U9W9udzy8wHMrfgTxdqT"}, body)
}

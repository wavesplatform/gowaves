package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewAlias(t *testing.T) {
	assert.NotNil(t, NewAlias(defaultOptions))
}

var aliasGetJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8"
}
`

func TestAlias_Get(t *testing.T) {
	addr, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(aliasGetJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.Get(context.Background(), "frozen")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, addr, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/by-alias/frozen", resp.Request.URL.String())
}

var aliasGetByAddressJson = `
["alias:T:v3g4n"]
`

func TestAlias_GetByAddress(t *testing.T) {
	address, err := proto.NewAddressFromString("3MvxxSiCqCA5gvtp3EgjEukpWg4HccdLAMh")
	require.Nil(t, err)
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		Client:  NewMockHttpRequestFromString(aliasGetByAddressJson, 200),
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Alias.GetByAddress(context.Background(), address)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(body))
	assert.Equal(t, "https://testnode1.wavesnodes.com/alias/by-address/3MvxxSiCqCA5gvtp3EgjEukpWg4HccdLAMh", resp.Request.URL.String())
}

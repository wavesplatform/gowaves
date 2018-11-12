package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var walletSeedJson = `
{
  "seed": "Nu2uaGvo5SJXG1aoai9ZGKYbJ7PEZppGVc6ogFkRHAv"
}`

func TestWallet_Seed(t *testing.T) {
	client, err := NewClient(Options{
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKEy",
		Client:  NewMockHttpRequestFromString(walletSeedJson, 200),
	})
	require.NoError(t, err)
	body, resp, err :=
		client.Wallet.Seed(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Nu2uaGvo5SJXG1aoai9ZGKYbJ7PEZppGVc6ogFkRHAv", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/wallet/seed", resp.Request.URL.String())
}

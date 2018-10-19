package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestWallet_Seed(t *testing.T) {
	apiKey := os.Getenv("ApiKey")
	if apiKey == "" {
		t.Skip("no env api key provided")
		return
	}

	client, err := NewClient(Options{BaseUrl: "https://testnode1.wavesnodes.com", ApiKey: apiKey})
	require.Nil(t, err)
	body, resp, err :=
		client.Wallet.Seed(context.Background())

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.NotEqual(t, "", body)
}

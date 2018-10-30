// +build integration

package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransactionIntegration_All(t *testing.T) {
	client, err := NewClient(Options{
		BaseUrl: "https://testnodes.wavesnodes.com",
	})
	require.Nil(t, err)
	_, resp, err :=
		client.Transactions.UnconfirmedSize(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
}

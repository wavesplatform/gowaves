package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestNewConsensus(t *testing.T) {
	assert.NotNil(t, NewConsensus(defaultOptions))
}

var consensusGeneratingBalanceJson = `
{
  "address": "3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp",
  "balance": 10003400001
}`

func TestConsensus_GeneratingBalance(t *testing.T) {
	address, _ := proto.NewAddressFromString("3MzemqBzJ9h844PparHU1EzGC5SQmtH5pNp")
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(consensusGeneratingBalanceJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.GeneratingBalance(context.Background(), address)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.Equal(t, uint64(10003400001), body.Balance)
}

var consensusGenerationSignatureJson = `
{
  "generationSignature": "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW"
}`

func TestConsensus_GenerationSignature(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(consensusGenerationSignatureJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.GenerationSignature(context.Background(), "abdcddd")
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW", body)
}

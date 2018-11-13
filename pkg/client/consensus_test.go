package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

var consensusGenerationSignatureByBlockJson = `
{
  "generationSignature": "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW"
}`

func TestConsensus_GenerationSignatureByBlock(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(consensusGenerationSignatureByBlockJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.GenerationSignatureByBlock(context.Background(), "abdcddd")
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW", body)
}

var consensusBaseTargetBlockJson = `
{
  "baseTarget": 737
}`

func TestConsensus_BaseTargetByBlock(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(consensusBaseTargetBlockJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.BaseTargetByBlock(context.Background(), "abdcddd")
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(737), body)
}

var consensusBaseTargetJson = `
{
  "baseTarget": 840,
  "score": "15057308169423316786914"
}
`

func TestConsensus_BaseTarget(t *testing.T) {
	client, err := NewClient(Options{
		Client: NewMockHttpRequestFromString(consensusBaseTargetJson, 200),
		ApiKey: "ApiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.BaseTarget(context.Background())
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(840), body.BaseTarget)
	assert.Equal(t, "15057308169423316786914", body.Score)
}

var consensusAlgoJson = `
{
  "consensusAlgo": "Fair Proof-of-Stake (FairPoS)"
}`

func TestConsensus_Algo(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(consensusAlgoJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Consensus.Algo(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Fair Proof-of-Stake (FairPoS)", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/consensus/algo", resp.Request.URL.String())
}

var consensusGenerationSignatureJson = `
{
  "generationSignature": "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW"
}`

func TestConsensus_GenerationSignature(t *testing.T) {
	client, _ := NewClient(Options{
		Client:  NewMockHttpRequestFromString(consensusGenerationSignatureJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	body, resp, err :=
		client.Consensus.GenerationSignature(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "EL4TZk4ANnSEsZ7ndgp89BaCDmcrhBNHEJJEwQiKWxdW", body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/consensus/generationsignature", resp.Request.URL.String())
}

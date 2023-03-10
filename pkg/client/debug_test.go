package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewDebug(t *testing.T) {
	assert.NotNil(t, NewDebug(defaultOptions))
}

var debugInfoJson = `
{
  "stateHeight": 376668,
  "extensionLoaderState": "State(Idle,Idle)",
  "historyReplierCacheSizes": {
    "blocks": 20,
    "microBlocks": 50
  },
  "microBlockSynchronizerCacheSizes": {
    "microBlockOwners": 5,
    "nextInvs": 4,
    "awaiting": 0,
    "successfullyReceived": 5
  },
  "scoreObserverStats": {
    "localScore": 1.4625538760279534e+22,
    "currentBestChannel": "Some([id: 0x3d813a03, L:/172.31.28.4:6863 - R:/5.189.150.22:58808])",
    "scoresCacheSize": 17
  },
  "minerState": "mining blocks"
}`

func TestDebug_Info(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(debugInfoJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Debug.Info(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, "1.4625538760279534e+22", body.ScoreObserverStats.LocalScore)
	assert.EqualValues(t, 20, body.HistoryReplierCacheSizes.Blocks)
	assert.Equal(t, "https://testnode1.wavesnodes.com/debug/info", resp.Request.URL.String())
	assert.NotEmpty(t, resp.Request.Header.Get(ApiKeyHeader))
}

var debugMinerInfoJson = `
[
  {
    "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
    "miningBalance": 70038873015279,
    "timestamp": 1542283963811
  }
]`

func TestDebug_MinerInfo(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(debugMinerInfoJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Debug.MinerInfo(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 1, len(body))
	assert.EqualValues(t, "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8", body[0].Address.String())
	assert.Equal(t, "https://testnode1.wavesnodes.com/debug/minerInfo", resp.Request.URL.String())
	assert.NotEmpty(t, resp.Request.Header.Get(ApiKeyHeader))
}

func TestDebug_ConfigInfo(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(`{}`, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Debug.ConfigInfo(context.Background(), false)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.True(t, len(body) > 0)
	assert.Equal(t, "https://testnode1.wavesnodes.com/debug/configInfo?full=false", resp.Request.URL.String())
	assert.NotEmpty(t, resp.Request.Header.Get(ApiKeyHeader))
}

var balancesHistoryJson = `
[
  {
    "height": 435572,
    "balance": 9452947750000
  },
  {
    "height": 435070,
    "balance": 9453947850000
  }
]
`

func TestDebug_BalancesHistory(t *testing.T) {
	client := client(t, NewMockHttpRequestFromString(balancesHistoryJson, 200))
	body, resp, err :=
		client.Debug.BalancesHistory(context.Background(), proto.MustAddressFromString("3MgSuT5FfeMrwwZCbztqLhQpcJNxySaFEiT"))
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.True(t, len(body) > 0)
	assert.Contains(t, resp.Request.URL.String(), "/debug/balances/history")
}

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

var debugBlocksJson = `
[
  {
    "227": "5oXh7pXSmk57GkfehtxpUzuopb4BauSoaoecETgty1Kb"
  },
  {
    "455": "GBeHktFQKnSoni1zgQwRzhPxDKeziUridSaaPpp7mKH"
  }
]`

func TestDebug_Blocks(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(debugBlocksJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Debug.Blocks(context.Background(), 1)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 2, len(body))
	assert.EqualValues(t, "GBeHktFQKnSoni1zgQwRzhPxDKeziUridSaaPpp7mKH", body[1][455])
	assert.Equal(t, "https://testnode1.wavesnodes.com/debug/blocks/1", resp.Request.URL.String())
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

var debugHistoryInfoJson = `
{
  "lastBlockIds": [
    "4sanYTRdcQC2UqWUj8JVQoR8S6vv2QnfDrqbg7zLHij8uSzkurNqxxeEFUfUr42VkeZo5ogQABNZj1tCbHvbkLL",
    "2WCvaAYE6rzpcoTi8YMT1BkH6shcXSfBE7hzhVJSHuTMdeAemLJCwmhjtN65XwSAW9skoDMpzWs7nb8dYnSXcY37",
    "2H5tfhuKTu7JUgcQYHVjBuD4zh5FGxWdnQN9vdNtQeFkcRg1qgfajozLN2jnCy2mYGq4VYx2TboHXSCMdkUGURsB",
    "3iu2ZSWMif6G3vJxzUwFAYEJYgJXh4q3gTDCp1kCFHHRzAVpCCjKrrFQ6FbWjUSvprHsWg4gYQd1AjTBLbKwdWy2",
    "5zsJ7Wt6rvuvLPRH77gG1kQswSXNQtJk2odBxE7HNE7dWs5FnGNhuoaWQNPsmwk1Ny2wfecUAyEnXkH44gu2dHXD",
    "34Q93KETbv5nzDAmEWqnv4B4MXEecYYASynRNxyHtLN3VrSRceBGRu1cSzKXz2mMbxyeMYGqY9Qwo5DCE1cn3EKa",
    "vZRqsPqHpup1Y1QDGQgFkmfE8h3NjYRqX1mkUU8D3c4Vdv6foqYhxpuaaR4bzuNq9bTP3RY3RBn8r3XRKW7hViF",
    "2jRJRiL6hTKNbk7XK1JBTwgLHCaFYh4XbW1xkuznVhN3EqRyXyc7PTFSirJCVtRhurze48d69fjmxkowxJaAFggF",
    "r9hCtXeKKbsRK1USAAuRTJ7qrhQvxXnPv2Zt7nrKciDuiLrv2REVjcjgZGxRsnf9xq6Ps9bzgyLnGiKzfURCbp1",
    "24uG1maWHQpL9piyEFNDx6pyN97NSPAax448GcE7ubxmPWk3ZofUgBoJqg86WRPXKuVSWUahRPPyFb6WawdAGLkr"
  ],
  "microBlockIds": [
    "NtNEPWoaTHkB4Y5qNeLLjZP5EfVfFYpn1iZrC4pK4Kgib5xworBZYG4sRWtq6J3dpTWyjNb9W88Q2WmZRXT2P2v",
    "4m4uGWb2op9TNTjLjRReJH7LzTGPd29fgRKGuVNLgVUhz4ue2mRxYW4R3LHihv1a7ZEaLyB9hLGP4A6cNEDTc6Xk"
  ]
}`

func TestDebug_HistoryInfo(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(debugHistoryInfoJson, 200),
		ApiKey:  "ApiKey",
		BaseUrl: "https://testnode1.wavesnodes.com/",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Debug.HistoryInfo(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(
		t,
		"4sanYTRdcQC2UqWUj8JVQoR8S6vv2QnfDrqbg7zLHij8uSzkurNqxxeEFUfUr42VkeZo5ogQABNZj1tCbHvbkLL",
		body.LastBlockIds[0].String())
	assert.Equal(t, "https://testnode1.wavesnodes.com/debug/historyInfo", resp.Request.URL.String())
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

var stateChangesJson = `
{
  "id": "83fxPJzEaQjEnVNGZ4TB4AdJrzCuFhy3xJkhVxY3dGf7",
  "height": 50,
  "stateChanges": {
    "data": [
      {
        "type": "integer",
        "key": "key",
        "value": 5
      }
    ],
    "transfers": [
      {
        "address": "3MgSuT5FfeMrwwZCbztqLhQpcJNxySaFEiT",
        "asset": "83fxPJzEaQjEnVNGZ4TB4AdJrzCuFhy3xJkhVxY3dGf7",
        "amount": 90
      }
    ]
  }
}

`

func TestDebug_StateChanges(t *testing.T) {
	client := client(t, NewMockHttpRequestFromString(stateChangesJson, 200))
	body, resp, err :=
		client.Debug.StateChanges(context.Background(), crypto.MustDigestFromBase58("83fxPJzEaQjEnVNGZ4TB4AdJrzCuFhy3xJkhVxY3dGf7"))
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, body)
	assert.Contains(t, resp.Request.URL.String(), "/debug/stateChanges/info/")
}

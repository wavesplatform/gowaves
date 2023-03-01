package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewAssets(t *testing.T) {
	assert.NotNil(t, NewAssets(defaultOptions))
}

var assetsBalanceJson = `
{
  "address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "balances": [
    {
      "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
      "balance": 1906756655,
      "reissuable": true,
      "minSponsoredAssetFee": null,
      "sponsorBalance": null,
      "quantity": 1906756655,
      "issueTransaction": {
        "type": 3,
        "id": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
        "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
        "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
        "fee": 104989368,
        "timestamp": 1502208238435,
        "signature": "4GnEfTtR91e8a6FZx89xMTWDhJEFpmNJnoFzBn3yduXcSH6TFcHy2AbXfjdc6ASrVEMjKYjupSCah2G9Pzk7jxSP",
        "proofs": [
          "4GnEfTtR91e8a6FZx89xMTWDhJEFpmNJnoFzBn3yduXcSH6TFcHy2AbXfjdc6ASrVEMjKYjupSCah2G9Pzk7jxSP"
        ],
        "version": 1,
        "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
        "name": "�\\��~V\"�w�",
        "quantity": 1906756655,
        "reissuable": true,
        "decimals": 1,
        "description": "\u001c-�\u0000@��Ï�"
      }
    }]
}
`

func TestAssets_BalanceByAddress(t *testing.T) {
	d, _ := crypto.NewDigestFromBase58("CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf")
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsBalanceJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.BalanceByAddress(context.Background(), address)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	require.Equal(t, 1, len(body.Balances))
	assert.Equal(t, d, body.Balances[0].AssetId)
	assert.EqualValues(t, 3, body.Balances[0].IssueTransaction.Type)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/balance/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8", resp.Request.URL.String())
}

var balanceByAddressAndAssetJson = `
{
	"address": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
	"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
	"balance": 1906756655
}`

func TestAssets_BalanceByAddressAndAsset(t *testing.T) {
	d, _ := crypto.NewDigestFromBase58("CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf")
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(balanceByAddressAndAssetJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.BalanceByAddressAndAsset(context.Background(), address, d)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, address, body.Address)
	assert.EqualValues(t, 1906756655, body.Balance)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/balance/3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8/CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf", resp.Request.URL.String())
}

var assertDetailsJson = `
{
  "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "issueHeight": 109232,
  "issueTimestamp": 1502208238435,
  "issuer": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "name": "�\\��~V\"�w�",
  "description": "\u001c-�\u0000@��Ï�",
  "decimals": 1,
  "reissuable": true,
  "quantity": 1906756656,
  "minSponsoredAssetFee": null
}`

func TestAssets_Details(t *testing.T) {
	assetId, _ := crypto.NewDigestFromBase58("CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assertDetailsJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Details(context.Background(), assetId)
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, assetId, body.AssetId)
	assert.EqualValues(t, 1906756656, body.Quantity)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/details/CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf", resp.Request.URL.String())
}

var assetsDistributionJson = `
{
  "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8": 1906756655
}`

func TestAssets_Distribution(t *testing.T) {
	assetId, _ := crypto.NewDigestFromBase58("CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf")
	address := "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8"
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsDistributionJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Distribution(context.Background(), assetId)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, map[string]uint64{address: 1906756655}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf/distribution", resp.Request.URL.String())
}

func TestAssets_DistributionAtHeight(t *testing.T) {
	const assetDistributionAtHeight = `
{
  "hasNext": true,
  "lastItem": "3PQL81CriMZu5tXjdbS5HqBVnrVpy9eRzp2",
  "items": {
    "3PJCh8EZ1toiXRM2schLUNG3Zy2L1fYvsGF": 172500,
    "3P76TmRjfjhdN9KEmwSnzQHpLrMRuf1qV29": 29198943,
    "3PQL81CriMZu5tXjdbS5HqBVnrVpy9eRzp2": 163275
  }
}`

	assetId := crypto.MustDigestFromBase58("34N9YcEETLWn93qYQ64EsP1x89tSruJU44RrEMSXXEPJ")
	addr := proto.MustAddressFromString("3PExCrMwdm9F7Cd2MW7vLAp2RQSYFSFqiUU")

	expectedLastItem := proto.MustAddressFromString("3PQL81CriMZu5tXjdbS5HqBVnrVpy9eRzp2")
	expectedItems := map[proto.WavesAddress]uint64{
		proto.MustAddressFromString("3PJCh8EZ1toiXRM2schLUNG3Zy2L1fYvsGF"): 172500,
		proto.MustAddressFromString("3P76TmRjfjhdN9KEmwSnzQHpLrMRuf1qV29"): 29198943,
		proto.MustAddressFromString("3PQL81CriMZu5tXjdbS5HqBVnrVpy9eRzp2"): 163275,
	}
	for _, after := range []*proto.WavesAddress{&addr, nil} {
		t.Run(fmt.Sprintf("after=%s", after), func(t *testing.T) {
			client, err := NewClient(Options{
				Client:  NewMockHttpRequestFromString(assetDistributionAtHeight, 200),
				BaseUrl: "https://testnode1.wavesnodes.com",
			})
			require.NoError(t, err)

			body, resp, err := client.Assets.DistributionAtHeight(context.Background(), assetId, 3533881, 3, after)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.True(t, body.HasNext)
			require.Equal(t, expectedLastItem, body.LastItem)
			require.Equal(t, expectedItems, body.Items)
		})
	}
}

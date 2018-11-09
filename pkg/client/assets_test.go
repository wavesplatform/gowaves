package client

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
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

var assetsIssueJson = `
{
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "name": "00kk",
  "description": "string",
  "quantity": 100,
  "decimals": 8,
  "reissuable": false,
  "fee": 100000000,
  "timestamp": 1541669009107
}`

func TestAssets_Issue(t *testing.T) {
	address, _ := proto.NewAddressFromString("3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsIssueJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Issue(context.Background(), AssetsIssueReq{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, &AssetsIssue{
		Sender:      address,
		Name:        "00kk",
		Description: "string",
		Quantity:    100,
		Decimals:    8,
		Reissuable:  false,
		Fee:         100000000,
		Timestamp:   1541669009107,
	}, body)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/issue", resp.Request.URL.String())
}

var assetsMassTransferJson = `
{
  "type": 11,
  "id": "HaNfTNE6FHRZrpTFKkfYLfzq6jT3bD3KiAv6g7KMzKVn",
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "fee": 200000,
  "timestamp": 1541684282576,
  "proofs": [
    "5bEXskGGGoPg5wG8QREg8Vjop6pgm2mihZKgoos83cAC55z6JyRRbmwhRCEuFtdgBcQU6d7sEN1CEAPBTF5gUpFU"
  ],
  "version": 1,
  "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "attachment": "t",
  "transferCount": 1,
  "totalAmount": 100,
  "transfers": [
    {
      "recipient": "3N5yE73RZkcdBC9jL1An3FJWGfMahqQyaQN",
      "amount": 100
    }
  ]
}`

func TestAssets_MassTransfer(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsMassTransferJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.MassTransfer(context.Background(), AssetsMassTransfersReq{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, proto.MassTransferTransaction, body.Type)
	assert.EqualValues(t, 1, body.Version)
	att, _ := proto.NewAttachmentFromBase58("t")
	assert.Equal(t, att, body.Attachment)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/masstransfer", resp.Request.URL.String())
}

var assetsSponsorJson = `
{
  "type": 14,
  "id": "6EjgYrLhWyLtotiYZANA3BRuZHMpszb74Gp3BnXJLjcZ",
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "fee": 100000000,
  "timestamp": 1541691379136,
  "proofs": [
    "g6dcYFR6dVHNwCiKptxW3PWVFzA2GaYMGX8vtWEFXeYkEpSVq9aU1tQzoqtsj4rbbGqcW8Tt1eQxSVExLTsZ3Cg"
  ],
  "version": 1,
  "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "minSponsoredAssetFee": 1
}`

func TestAssets_Sponsor(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsSponsorJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Sponsor(context.Background(), AssetsSponsorReq{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, proto.SponsorshipTransaction, body.Type)
	assert.EqualValues(t, 1, body.Version)
	assert.Equal(t, 1, len(body.Proofs.Proofs))
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/sponsor", resp.Request.URL.String())
}

var assetsTransferJson = `
{
  "type": 4,
  "id": "56X71ws8xgrBu3AF6YT2rQ3Mqvh1XZ9pTcLMsRJBvgoK",
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "fee": 1,
  "timestamp": 1541685898570,
  "proofs": [
    "RxQrXF9kQx2zMTyqf2GxJY2UcKWfnxr65TDzPKCzv2qPtYciV1ZZ93313X7FfrFRLqfREsdR6gtLnuLeZR2bRZ2"
  ],
  "version": 2,
  "recipient": "3N5yE73RZkcdBC9jL1An3FJWGfMahqQyaQN",
  "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "feeAssetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "feeAsset": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "amount": 1,
  "attachment": "T"
}`

func TestAssets_Transfer(t *testing.T) {
	id, _ := crypto.NewDigestFromBase58("56X71ws8xgrBu3AF6YT2rQ3Mqvh1XZ9pTcLMsRJBvgoK")
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsTransferJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Transfer(context.Background(), AssetsTransferReq{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, proto.TransferTransaction, body.Type)
	assert.EqualValues(t, 2, body.Version)
	assert.Equal(t, &id, body.ID)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/transfer", resp.Request.URL.String())
}

var assetsBurnJson = `
{
  "type": 6,
  "id": "C36WStdMDe4EYABXc2LruPCr7MEPketGJvmaiwMvM56G",
  "sender": "3NBVqYXrapgJP9atQccdBPAgJPwHDKkh6A8",
  "senderPublicKey": "CRxqEuxhdZBEHX42MU4FfyJxuHmbDBTaHMhM3Uki7pLw",
  "fee": 100000,
  "timestamp": 1541767435814,
  "signature": "4uY38B47h38HX62YaKLWapUK7ehueHA4iB5HxSsuiuyNvDk32zRwh7ysfpZ5YRgdyrFq5i2EEWB6ppZ3ptAJVCfE",
  "proofs": [
    "4uY38B47h38HX62YaKLWapUK7ehueHA4iB5HxSsuiuyNvDk32zRwh7ysfpZ5YRgdyrFq5i2EEWB6ppZ3ptAJVCfE"
  ],
  "chainId": null,
  "version": 1,
  "assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
  "amount": 1
}`

func TestAssets_Burn(t *testing.T) {
	client, err := NewClient(Options{
		Client:  NewMockHttpRequestFromString(assetsBurnJson, 200),
		BaseUrl: "https://testnode1.wavesnodes.com",
		ApiKey:  "apiKey",
	})
	require.Nil(t, err)
	body, resp, err :=
		client.Assets.Burn(context.Background(), AssetsBurnReq{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, proto.BurnTransaction, body.Type)
	assert.EqualValues(t, 1, body.Version)
	assert.Equal(t, "https://testnode1.wavesnodes.com/assets/burn", resp.Request.URL.String())
}

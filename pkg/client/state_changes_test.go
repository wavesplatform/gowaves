package client

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestUnmarshalStateChangesAllFields(t *testing.T) {
	jsonSrc := `{
		"data": [
			{
				"type": "string",
				"key": "keyHello",
				"value": "hello"
			}
		],
		"transfers": [
			{
				"address": "3N5yE73RZkcdBC9jL1An3FJWGfMahqQyaQN",
				"asset": null,
				"amount": 900000
			}
		],
		"issues": [
			{
				"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
				"name": "CatCoin",
				"description": "Cats are the best",
				"decimals": 6,
				"isReissuable": true,
				"compiledScript": "AQQAAAAQd2hpdGVMaXN0QWNjb3VudAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVy3YfBi6sTVYY0bkC3rJRVVPBcXqnEJojwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMEAAAACXJlY2lwaWVudAkAAlgAAAABCAkABCQAAAABCAUAAAACdHgAAAAJcmVjaXBpZW50AAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAABnNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAACXJlY2lwaWVudAcDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAA9zZWxsT3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAJc2VsbE9yZGVyAAAABnNlbmRlcgAAAAVieXRlcwQAAAAOYnV5T3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAIYnV5T3JkZXIAAAAGc2VuZGVyAAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAAD3NlbGxPcmRlclNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAADmJ1eU9yZGVyU2VuZGVyBwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMJAQAAAAdleHRyYWN0AAAAAQkABBsAAAACBQAAABB3aGl0ZUxpc3RBY2NvdW50BQAAAAZzZW5kZXIGWSftFg=="
			},
			{
				"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
				"name": "VIRES_USDC_LP",
				"description": "USDC liquidity provider token",
				"quantity": 0,
				"decimals": 6,
				"isReissuable": true,
				"compiledScript": null,
				"nonce": 1
			}
		],
		"reissues": [
			{
				"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
				"isReissuable": false,
				"quantity": 10000
			}
		],
		"burns": [
			{
				"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
				"quantity": 10000
			}
		],
		"sponsorFees": [
			{
				"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf",
				"minSponsoredAssetFee": 100
			}
		],
		"leases": [
			{
				"id": "7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL",
				"originTransactionId": "7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL",
				"sender": "3MvhHXWL5TCskTpL3XS2euywQaoFyzLWHCu",
				"recipient": "3MqCsTH9y6nFJEFL81DubDrpKnvreR9M52p",
				"amount": 100000000,
				"height": 10596,
				"status": "active"
			}
		],
		"leaseCancels": [
			{
				"id": "7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL",
				"originTransactionId": "7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL",
				"sender": "3MvhHXWL5TCskTpL3XS2euywQaoFyzLWHCu",
				"recipient": "3MqCsTH9y6nFJEFL81DubDrpKnvreR9M52p",
				"amount": 100000000,
				"height": 10596,
				"status": "canceled"
			}
		],
		"invokes": [
			{
				"dApp": "3My9cBgDYLyeT1YF8ip9XxqwWvJMjj8WdeM",
				"payment": [
					{
						"amount": 100000,
						"assetId": "CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf"
					}
				],
				"call": {
					"function": "printNumber",
					"args": [
						{
							"type": "integer",
							"value": 1000
						}
					]
				}
			}
		]
	}`

	expectedTransferAddr, _ := proto.NewAddressFromString("3N5yE73RZkcdBC9jL1An3FJWGfMahqQyaQN")
	expectedAssetId := crypto.MustDigestFromBase58("CMBHKDtyE8GMbZAZANNeE5n2HU4VDpsQaBLmfCw9ASbf")
	expectedScript, _ := base64.StdEncoding.DecodeString("AQQAAAAQd2hpdGVMaXN0QWNjb3VudAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVy3YfBi6sTVYY0bkC3rJRVVPBcXqnEJojwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMEAAAACXJlY2lwaWVudAkAAlgAAAABCAkABCQAAAABCAUAAAACdHgAAAAJcmVjaXBpZW50AAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAABnNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAACXJlY2lwaWVudAcDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAA9zZWxsT3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAJc2VsbE9yZGVyAAAABnNlbmRlcgAAAAVieXRlcwQAAAAOYnV5T3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAIYnV5T3JkZXIAAAAGc2VuZGVyAAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAAD3NlbGxPcmRlclNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAADmJ1eU9yZGVyU2VuZGVyBwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMJAQAAAAdleHRyYWN0AAAAAQkABBsAAAACBQAAABB3aGl0ZUxpc3RBY2NvdW50BQAAAAZzZW5kZXIGWSftFg==")
	expectedLeaseID := crypto.MustDigestFromBase58("7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL")
	expectedLeaseSender, _ := proto.NewAddressFromString("3MvhHXWL5TCskTpL3XS2euywQaoFyzLWHCu")
	expectedLeaseRecipient, _ := proto.NewRecipientFromString("3MqCsTH9y6nFJEFL81DubDrpKnvreR9M52p")
	expectedInvokeDApp, _ := proto.NewAddressFromString("3My9cBgDYLyeT1YF8ip9XxqwWvJMjj8WdeM")
	expectedMinSponsoredFee := int64(100)

	expectedStateChanges := StateChanges{
		Data: DataEntries{
			&proto.StringDataEntry{
				Key:   "keyHello",
				Value: "hello",
			},
		},
		Transfers: []TransferAction{
			{
				Address: expectedTransferAddr,
				Asset:   proto.NewOptionalAssetWaves(),
				Amount:  900000,
			},
		},
		Issues: []IssueAction{
			{
				AssetID:        expectedAssetId,
				Name:           "CatCoin",
				Description:    "Cats are the best",
				Decimals:       6,
				Reissuable:     true,
				CompiledScript: expectedScript,
			},
			{
				AssetID:        expectedAssetId,
				Name:           "VIRES_USDC_LP",
				Description:    "USDC liquidity provider token",
				Quantity:       0,
				Decimals:       6,
				Reissuable:     true,
				CompiledScript: nil,
				Nonce:          1,
			},
		},
		Reissues: []ReissueAction{
			{
				AssetID:    expectedAssetId,
				Reissuable: false,
				Quantity:   10000,
			},
		},
		Burns: []BurnAction{
			{
				AssetID:  expectedAssetId,
				Quantity: 10000,
			},
		},
		SponsorFees: []SponsorFeeAction{
			{
				AssetID:              expectedAssetId,
				MinSponsoredAssetFee: &expectedMinSponsoredFee,
			},
		},
		Leases: []LeaseAction{
			{
				ID:                  expectedLeaseID,
				OriginTransactionId: expectedLeaseID,
				Sender:              expectedLeaseSender,
				Recipient:           expectedLeaseRecipient,
				Amount:              100000000,
				Height:              10596,
				Status:              LeaseActiveStatus,
			},
		},
		LeaseCancels: []LeaseCancelAction{
			{
				ID:                  expectedLeaseID,
				OriginTransactionId: expectedLeaseID,
				Sender:              expectedLeaseSender,
				Recipient:           expectedLeaseRecipient,
				Amount:              100000000,
				Height:              10596,
				Status:              LeaseCanceledStatus,
			},
		},
		Invokes: []InvokeAction{
			{
				DApp: expectedInvokeDApp,
				Call: proto.FunctionCall{
					Name: "printNumber",
					Arguments: proto.Arguments{
						proto.NewIntegerArgument(1000),
					},
				},
				Payments: []proto.ScriptPayment{
					{
						Asset:  *proto.NewOptionalAssetFromDigest(expectedAssetId),
						Amount: 100000,
					},
				},
			},
		},
	}

	var actualChanges StateChanges
	require.NoError(t, json.Unmarshal([]byte(jsonSrc), &actualChanges), "unmarshal StateChanges error")
	require.Equal(t, expectedStateChanges, actualChanges, "non equal stateChanges")
}

func TestUnmarshalStateChangesLeaseCancelParams(t *testing.T) {
	// mainnet tx 'HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw'
	const jsonSrc = `{
		"data": [
			{
				"key": "leasing_id",
				"type": "binary",
				"value": "base64:mXrKMW0+BGVMMEQLHuT7MYTFWYQr3Bn00uZyns/tGWQ="
			},
			{
				"key": "leasing_amount",
				"type": "integer",
				"value": 9929853366
			}
		],
		"transfers": [],
		"issues": [],
		"reissues": [],
		"burns": [],
		"sponsorFees": [],
		"leases": [
			{
				"id": "BL7yUWMEnCqSpykfRGxcSwnUP7KzVM3kfbXEPGwS9MQF",
				"originTransactionId": "HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw",
				"sender": "3PRFKemXs4rAJYGPccNtP63Kw2UzwEdH7sZ",
				"recipient": "3PK8o5xADFueuBVKX2qdgTf7wV6j5pJMUas",
				"amount": 9929853366,
				"height": 3407048,
				"status": "active",
				"cancelHeight": null,
				"cancelTransactionId": null
			}
		],
		"leaseCancels": [
			{
				"id": "AcFMxxaZJ53HRkSEMdrFxwjrGY74RzBnoEfRzvgQ2h1j",
				"originTransactionId": "6AXnciRNxgE1EK6ScL7zVvshdCT9AaVuFiRi2fkivVzF",
				"sender": "3PRFKemXs4rAJYGPccNtP63Kw2UzwEdH7sZ",
				"recipient": "3PK8o5xADFueuBVKX2qdgTf7wV6j5pJMUas",
				"amount": 9891985036,
				"height": 3406858,
				"status": "canceled",
				"cancelHeight": 3407048,
				"cancelTransactionId": "HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw"
			}
		],
		"invokes": []
	}`

	var sc StateChanges
	err := json.Unmarshal([]byte(jsonSrc), &sc)
	require.NoError(t, err)
	require.Nil(t, sc.Leases[0].CancelHeight)
	require.Nil(t, sc.Leases[0].CancelTransactionId)

	d, err := crypto.NewDigestFromBase58("HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw")
	require.NoError(t, err)
	require.NotNil(t, d, sc.LeaseCancels[0].CancelHeight)
	require.Equal(t, uint32(3407048), *sc.LeaseCancels[0].CancelHeight)
	require.NotNil(t, d, sc.LeaseCancels[0].CancelTransactionId)
	require.Equal(t, d, *sc.LeaseCancels[0].CancelTransactionId)
}

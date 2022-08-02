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
				"status": "canceled"
			}
		],
		"leaseCancel": [
			{
				"leaseId": "7517y2CZZZD6HUVy6bAV3R4EV4Zrd7ZtEW2WVawHiAgL"
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
				MinSponsoredAssetFee: 100,
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
				Status:              LeaseCanceledStatus,
			},
		},
		LeaseCancel: []LeaseCancelAction{
			{
				LeaseID: expectedLeaseID,
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

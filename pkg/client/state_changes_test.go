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

func TestName(t *testing.T) {
	// mainnet tx 'HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw'
	const js = `
{
  "type": 18,
  "id": "HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw",
  "fee": 500000,
  "feeAssetId": null,
  "timestamp": 1669915557263,
  "version": 1,
  "chainId": 87,
  "bytes": "0xf902b2860184ceb93d8f8502540be4008307a120949d0caac61351a96ecb80f7637dc16478e2ef724d80b90244960ec803000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000009e727900000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000001473939393939392f335045654a51524a543476345876535542506d786864574b7a3433396e6165374b74512c73776f7066692c44473278466b506444774b556f426b7a47416851744c7053477a66584c69435950457a654b483241643234702c43316957734b47714c776a48556e64695137695870646d50756d39506543444666795842644a4a6f734452533b33504d76797455796a6f35716d427432535a6d7852585a41466d694669624765625a622c70757a7a6c652c43316957734b47714c776a48556e64695137695870646d50756d39506543444666795842644a4a6f734452532c57415645533b335052464b656d58733472414a59475063634e745036334b7732557a7745644837735a2c73776f7066692c57415645532c48454238516177397872577057733874487369415459474257444274503253376b6350414c724d7534334153000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001b62629c304f5ce5391a40e4b75242f648c51b1fadfaf5429bd48d21d2ab2aad100000000000000000000000000000000000000000000000000000000000f424081d1a0db7bc8f127bdd32d4fc9ea9a2d9323f969ac7659a4a8651c0d1df5f4d3db6feba04e625a0c37e7ca311e834724625ed638680d7278bb3b6ccde9ee7112101d99bd",
  "sender": "3P5gYUQUToz2zgxHbEZfNW8s75DyAdX3SkK",
  "senderPublicKey": "2R2FfZsthzR3XenfzUNMQ6dM5ubc6UUHHZY4PwH4pf4196RnwimucPtuidhBqJ3gSqupPhk9cS5bVwSYjHkXmPyM",
  "height": 3407048,
  "applicationStatus": "succeeded",
  "spentComplexity": 6125,
  "payload": {
    "type": "invocation",
    "dApp": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU",
    "call": {
      "function": "swap",
      "args": [
        {
          "type": "string",
          "value": "999999/3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ,swopfi,DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p,C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS;3PMvytUyjo5qmBt2SZmxRXZAFmiFibGebZb,puzzle,C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS,WAVES;3PRFKemXs4rAJYGPccNtP63Kw2UzwEdH7sZ,swopfi,WAVES,HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS"
        },
        {
          "type": "integer",
          "value": 10383993
        }
      ]
    },
    "payment": [
      {
        "amount": 1000000,
        "assetId": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p"
      }
    ],
    "stateChanges": {
      "data": [],
      "transfers": [
        {
          "address": "3P5gYUQUToz2zgxHbEZfNW8s75DyAdX3SkK",
          "asset": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
          "amount": 10748793
        },
        {
          "address": "3P5gYUQUToz2zgxHbEZfNW8s75DyAdX3SkK",
          "asset": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
          "amount": 1
        }
      ],
      "issues": [],
      "reissues": [],
      "burns": [],
      "sponsorFees": [],
      "leases": [],
      "leaseCancels": [],
      "invokes": [
        {
          "dApp": "3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ",
          "call": {
            "function": "exchange",
            "args": [
              {
                "type": "Int",
                "value": 1
              }
            ]
          },
          "payment": [
            {
              "assetId": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
              "amount": 999999
            }
          ],
          "stateChanges": {
            "data": [
              {
                "key": "A_asset_balance",
                "type": "integer",
                "value": 8193264466
              },
              {
                "key": "B_asset_balance",
                "type": "integer",
                "value": 1011471859
              }
            ],
            "transfers": [
              {
                "address": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU",
                "asset": "C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS",
                "amount": 8059667
              },
              {
                "address": "3P6J84oH51DzY6xk2mT5TheXRbrCwBMxonp",
                "asset": "C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS",
                "amount": 19459
              }
            ],
            "issues": [],
            "reissues": [],
            "burns": [],
            "sponsorFees": [],
            "leases": [],
            "leaseCancels": [],
            "invokes": [
              {
                "dApp": "3PBFHAHS4PZaXpS7gT5SPLnuPh7YPoJgCfE",
                "call": {
                  "function": "exchange",
                  "args": [
                    {
                      "type": "ByteVector",
                      "value": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU"
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "String",
                          "value": "1"
                        }
                      ]
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "Int",
                          "value": 999999
                        }
                      ]
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "ByteVector",
                          "value": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p"
                        }
                      ]
                    }
                  ]
                },
                "payment": [],
                "stateChanges": {
                  "data": [],
                  "transfers": [],
                  "issues": [],
                  "reissues": [],
                  "burns": [],
                  "sponsorFees": [],
                  "leases": [],
                  "leaseCancels": [],
                  "invokes": [
                    {
                      "dApp": "3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ",
                      "call": {
                        "function": "stakeUnstake",
                        "args": [
                          {
                            "type": "Boolean",
                            "value": true
                          },
                          {
                            "type": "Int",
                            "value": 999999
                          },
                          {
                            "type": "String",
                            "value": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p"
                          }
                        ]
                      },
                      "payment": [],
                      "stateChanges": {
                        "data": [],
                        "transfers": [],
                        "issues": [],
                        "reissues": [],
                        "burns": [],
                        "sponsorFees": [],
                        "leases": [],
                        "leaseCancels": [],
                        "invokes": [
                          {
                            "dApp": "3PNikM6yp4NqcSU8guxQtmR5onr2D4e8yTJ",
                            "call": {
                              "function": "lockNeutrino",
                              "args": []
                            },
                            "payment": [
                              {
                                "assetId": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
                                "amount": 999999
                              }
                            ],
                            "stateChanges": {
                              "data": [
                                {
                                  "key": "%s%s%s%s__history__stake__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ__HPsKrSJFFe9duhGbT3jb1YYnTninzmb99JRdRTtRefLw",
                                  "type": "string",
                                  "value": "%s%d%d%d%d__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ__3407048__1669915546569__1010471860__1011471859"
                                },
                                {
                                  "key": "%s%s%s__rwd__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ__DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
                                  "type": "integer",
                                  "value": 193
                                },
                                {
                                  "key": "%s%s%s__userRwdFromDepNum__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ",
                                  "type": "integer",
                                  "value": 0
                                },
                                {
                                  "key": "%s%s%s__paramByUser__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ__amount",
                                  "type": "integer",
                                  "value": 1011471859
                                },
                                {
                                  "key": "%s%s%s__paramByUser__3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ__start",
                                  "type": "integer",
                                  "value": 3356246
                                },
                                {
                                  "key": "rpd_balance_DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p_3PEeJQRJT4v4XvSUBPmxhdWKz439nae7KtQ",
                                  "type": "integer",
                                  "value": 1011471859
                                },
                                {
                                  "key": "%s%s__stats__locksCount",
                                  "type": "integer",
                                  "value": 83843
                                },
                                {
                                  "key": "%s%s__stats__activeUsersCount",
                                  "type": "integer",
                                  "value": 4472
                                },
                                {
                                  "key": "%s%s__stats__activeTotalLocked",
                                  "type": "integer",
                                  "value": 102372194075123
                                },
                                {
                                  "key": "rpd_balance_DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
                                  "type": "integer",
                                  "value": 102372194075123
                                }
                              ],
                              "transfers": [],
                              "issues": [],
                              "reissues": [],
                              "burns": [],
                              "sponsorFees": [],
                              "leases": [],
                              "leaseCancels": [],
                              "invokes": []
                            }
                          }
                        ]
                      }
                    }
                  ]
                }
              }
            ]
          }
        },
        {
          "dApp": "3PMvytUyjo5qmBt2SZmxRXZAFmiFibGebZb",
          "call": {
            "function": "swap",
            "args": [
              {
                "type": "String",
                "value": "WAVES"
              },
              {
                "type": "Int",
                "value": 0
              }
            ]
          },
          "payment": [
            {
              "assetId": "C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS",
              "amount": 8059667
            }
          ],
          "stateChanges": {
            "data": [
              {
                "key": "global_WAVES_balance",
                "type": "integer",
                "value": 9437101256
              },
              {
                "key": "global_C1iWsKGqLwjHUndiQ7iXpdmPum9PeCDFfyXBdJJosDRS_balance",
                "type": "integer",
                "value": 296996019
              },
              {
                "key": "global_HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS_balance",
                "type": "integer",
                "value": 1264990751
              },
              {
                "key": "global_earnedByOwner",
                "type": "integer",
                "value": 2440130573
              },
              {
                "key": "global_volume",
                "type": "integer",
                "value": 135152720688
              }
            ],
            "transfers": [
              {
                "address": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU",
                "asset": null,
                "amount": 37868330
              },
              {
                "address": "3PJ5pJLA8Pae4uEMWksrXpygKChoKbAMayt",
                "asset": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
                "amount": 32529
              }
            ],
            "issues": [],
            "reissues": [],
            "burns": [
              {
                "assetId": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
                "quantity": 130119
              }
            ],
            "sponsorFees": [],
            "leases": [],
            "leaseCancels": [],
            "invokes": []
          }
        },
        {
          "dApp": "3PRFKemXs4rAJYGPccNtP63Kw2UzwEdH7sZ",
          "call": {
            "function": "exchange",
            "args": [
              {
                "type": "Int",
                "value": 1
              }
            ]
          },
          "payment": [
            {
              "assetId": null,
              "amount": 37868330
            }
          ],
          "stateChanges": {
            "data": [
              {
                "key": "A_asset_balance",
                "type": "integer",
                "value": 2833008701
              },
              {
                "key": "B_asset_balance",
                "type": "integer",
                "value": 9929853366
              }
            ],
            "transfers": [
              {
                "address": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU",
                "asset": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
                "amount": 10780055
              },
              {
                "address": "3P6J84oH51DzY6xk2mT5TheXRbrCwBMxonp",
                "asset": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
                "amount": 26028
              }
            ],
            "issues": [],
            "reissues": [],
            "burns": [],
            "sponsorFees": [],
            "leases": [],
            "leaseCancels": [],
            "invokes": [
              {
                "dApp": "3PBFHAHS4PZaXpS7gT5SPLnuPh7YPoJgCfE",
                "call": {
                  "function": "exchange",
                  "args": [
                    {
                      "type": "ByteVector",
                      "value": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU"
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "String",
                          "value": "1"
                        }
                      ]
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "Int",
                          "value": 37868330
                        }
                      ]
                    },
                    {
                      "type": "Array",
                      "value": [
                        {
                          "type": "ByteVector",
                          "value": ""
                        }
                      ]
                    }
                  ]
                },
                "payment": [],
                "stateChanges": {
                  "data": [],
                  "transfers": [],
                  "issues": [],
                  "reissues": [],
                  "burns": [],
                  "sponsorFees": [],
                  "leases": [],
                  "leaseCancels": [],
                  "invokes": [
                    {
                      "dApp": "3PRFKemXs4rAJYGPccNtP63Kw2UzwEdH7sZ",
                      "call": {
                        "function": "stakeUnstake",
                        "args": [
                          {
                            "type": "Boolean",
                            "value": true
                          },
                          {
                            "type": "Int",
                            "value": 37868330
                          },
                          {
                            "type": "String",
                            "value": "WAVES"
                          }
                        ]
                      },
                      "payment": [],
                      "stateChanges": {
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
                      }
                    }
                  ]
                }
              }
            ]
          }
        },
        {
          "dApp": "3PFDgzu1UtswAkCMxqqQjbTeHaX4cMab8Kh",
          "call": {
            "function": "swap",
            "args": [
              {
                "type": "String",
                "value": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p"
              },
              {
                "type": "Int",
                "value": 0
              }
            ]
          },
          "payment": [
            {
              "assetId": "HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS",
              "amount": 31262
            }
          ],
          "stateChanges": {
            "data": [
              {
                "key": "global_lastPuzzlePrice",
                "type": "integer",
                "value": 9148486
              },
              {
                "key": "global_DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p_balance",
                "type": "integer",
                "value": 45234122302
              },
              {
                "key": "global_HEB8Qaw9xrWpWs8tHsiATYGBWDBtP2S7kcPALrMu43AS_balance",
                "type": "integer",
                "value": 3954760620292
              }
            ],
            "transfers": [
              {
                "address": "3PES7MMthaKJx9WMXnNCY3cwTGG9nD9YT8f",
                "asset": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
                "amount": 57
              },
              {
                "address": "3PGFHzVGT4NTigwCKP1NcwoXkodVZwvBuuU",
                "asset": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
                "amount": 2803
              }
            ],
            "issues": [],
            "reissues": [],
            "burns": [],
            "sponsorFees": [],
            "leases": [],
            "leaseCancels": [],
            "invokes": []
          }
        },
        {
          "dApp": "3PKUxbZaSYfsR7wu2HaAgiirHYwAMupDrYW",
          "call": {
            "function": "topUpReward",
            "args": []
          },
          "payment": [
            {
              "assetId": "DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p",
              "amount": 2803
            }
          ],
          "stateChanges": {
            "data": [
              {
                "key": "global_DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p_earnings",
                "type": "integer",
                "value": 73633416463
              },
              {
                "key": "global_lastCheck_DG2xFkPdDwKUoBkzGAhQtLpSGzfXLiCYPEzeKH2Ad24p_interest",
                "type": "integer",
                "value": 98123653964792020
              }
            ],
            "transfers": [],
            "issues": [],
            "reissues": [],
            "burns": [],
            "sponsorFees": [],
            "leases": [],
            "leaseCancels": [],
            "invokes": []
          }
        }
      ]
    }
  }
}`
	ei := new(EthereumTransactionInfo)
	err := ei.UnmarshalJSON([]byte(js))
	require.NoError(t, err)
}

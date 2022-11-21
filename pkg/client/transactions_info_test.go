package client

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestEthereumTransferTransactionInfo(t *testing.T) {
	jsonSrc := `{
		"type": 18,
		"id": "8NJXHtHTSwmb3th98omHdWCmeTrkKH4Q3w1SRC3FyUFK",
		"fee": 100000,
		"feeAssetId": null,
		"timestamp": 1656599340244,
		"version": 1,
		"chainId": 84,
		"bytes": "0xf874860181b503e4d48502540be400830186a094ac2acffa6113399cd85038bd5e28b52d6094db2988016345785d8a00008081cba0f0478e38af7b562ae90e5ea42c545248445360de8c596983327f6df7477d41bca01bd7e9a8410d960b79ebde8f175e79edb31d1ce90464ac9134bcd2671361af9a",
		"sender": "3MpLdCXFukShUXsHXLoiUGZCzzkaBJEnmVh",
		"senderPublicKey": "v5DNa6N7r7Qmssi5LDFVV2kFzDNczCt7L6qubJjoGVrcfeT1Rdwtn5515QdHFztjLibGWRfhsvFv84qoCckU4a1",
		"height": 2119282,
		"applicationStatus": "succeeded",
		"spentComplexity": 0,
		"payload": {
			"type": "transfer",
			"recipient": "3N5cRHaFQTmuJ2sbHrKmgk7WW1jTe5ZnNPy",
			"asset": null,
			"amount": 10000000
		}
	}`

	txInfo := new(EthereumTransactionInfo)
	err := json.Unmarshal([]byte(jsonSrc), txInfo)
	require.NoError(t, err, "unmarshal transfer Ethereum transaction info")

	expectedRecipient, _ := proto.NewRecipientFromString("3N5cRHaFQTmuJ2sbHrKmgk7WW1jTe5ZnNPy")
	var expectedPayload EthereumTransactionPayload = &EthereumTransactionTransferPayload{
		Recipient: expectedRecipient,
		Asset:     proto.NewOptionalAssetWaves(),
		Amount:    10000000,
	}
	require.Equal(t, expectedPayload, txInfo.Payload, "check payload equality")
}

func TestEthereumInvocationTransactionInfo(t *testing.T) {
	tests := []string{
		`
		  {
			"type": 18,
			"id": "2R1979dHWjjQfhdfmNi1v54YRJg8HcZ7zWFxTfretsqY",
			"fee": 500000,
			"feeAssetId": null,
			"timestamp": 1668515141806,
			"version": 1,
			"chainId": 84,
			"bytes": "0xf901f28601847b4098ae8502540be4008307a12094878bbf66de7c60866e9a5fe7f57e4b8571f9dac380b90184483c83ce000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000000865786368616e6765000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000009363631343531343037000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000be420e081cba0fe19650d0d87493513f62fb8bd2823e97a339c92f4fc58465930ddf831d8bff0a035c598fa1256209d79be88d8b63b35ba0dff8caa4ccea265a4842d13268cc000",
			"sender": "3NARGSvSxBMnCCCwSJHxovCGWuSvtGnmSKj",
			"senderPublicKey": "5DBpX6U48n14NEv79DumUvpiP4b1SM3w6K9gfsU44JBQMFZRLhzjvczbHggXJe9mM9C1zsWuDBxK2nQGFQcKhfHB",
			"height": 2318236,
			"applicationStatus": "succeeded",
			"spentComplexity": 1187,
			"payload": {
			  "type": "invocation",
			  "dApp": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
			  "call": {
				"function": "callFunction",
				"args": [
				  {
					"type": "string",
					"value": "exchange"
				  },
				  {
					"type": "list",
					"value": [
					  {
						"type": "string",
						"value": "661451407"
					  }
					]
				  }
				]
			  },
			  "payment": [
				{
				  "amount": 199500000,
				  "assetId": null
				}
			  ],
			  "stateChanges": {
				"data": [
				  {
					"key": "A_asset_balance",
					"type": "integer",
					"value": 299500000
				  },
				  {
					"key": "B_asset_balance",
					"type": "integer",
					"value": 336287815
				  }
				],
				"transfers": [
				  {
					"address": "3NARGSvSxBMnCCCwSJHxovCGWuSvtGnmSKj",
					"asset": "8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS",
					"amount": 662113521
				  },
				  {
					"address": "3N2eueE5vLLKe8jXuBDbdbKcPaH36yG1Had",
					"asset": "8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS",
					"amount": 1598664
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
					"dApp": "3MzqQ3HKdkHmJmk9mDhAeAMxmK5D2ztdAe5",
					"call": {
					  "function": "exchange",
					  "args": [
						{
						  "type": "ByteVector",
						  "value": "3NARGSvSxBMnCCCwSJHxovCGWuSvtGnmSKj"
						},
						{
						  "type": "Array",
						  "value": [
							{
							  "type": "String",
							  "value": "661451407"
							}
						  ]
						},
						{
						  "type": "Array",
						  "value": [
							{
							  "type": "Int",
							  "value": 199500000
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
						  "dApp": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
						  "call": {
							"function": "stakeUnstake",
							"args": [
							  {
								"type": "Boolean",
								"value": true
							  },
							  {
								"type": "Int",
								"value": 199500000
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
								"value": "base64:O+0HPhSuaf6lD6Y4tW2OSdyaa/8BGGQdk1ch/IgVP08="
							  },
							  {
								"key": "leasing_amount",
								"type": "integer",
								"value": 299500000
							  }
							],
							"transfers": [],
							"issues": [],
							"reissues": [],
							"burns": [],
							"sponsorFees": [],
							"leases": [
							  {
								"id": "52vgdFas57Lbfe7ccvtq3bocvteJWRR7uQYeUatWQbxA",
								"originTransactionId": "2R1979dHWjjQfhdfmNi1v54YRJg8HcZ7zWFxTfretsqY",
								"sender": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
								"recipient": "3MvKopk5a1HPTCPgzMixcSanqJ9jskZzjfu",
								"amount": 299500000,
								"height": 2318236,
								"status": "canceled",
								"cancelHeight": 2324157,
								"cancelTransactionId": "E5aFJJWDCbGsWBvJxqEoKVt7kFXytCjhfdfVoKsR9gbE"
							  }
							],
							"leaseCancels": [
							  {
								"id": "C9mzgEDgZr8KBQh67JeaXPFCyaGZQxq5ds1ePgrrpNre",
								"originTransactionId": "ss6FC9Z7P2rLLtURfrCiaLRrHMQKTP2G5fUdQqgPbfi",
								"sender": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
								"recipient": "3MvKopk5a1HPTCPgzMixcSanqJ9jskZzjfu",
								"amount": 100000000,
								"height": 2277595,
								"status": "canceled",
								"cancelHeight": 2318236,
								"cancelTransactionId": "2R1979dHWjjQfhdfmNi1v54YRJg8HcZ7zWFxTfretsqY"
							  }
							],
							"invokes": []
						  }
						},
						{
						  "dApp": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
						  "call": {
							"function": "stakeUnstake",
							"args": [
							  {
								"type": "Boolean",
								"value": false
							  },
							  {
								"type": "Int",
								"value": 663712185
							  },
							  {
								"type": "String",
								"value": "8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS"
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
								"dApp": "3MvKopk5a1HPTCPgzMixcSanqJ9jskZzjfu",
								"call": {
								  "function": "unlockNeutrino",
								  "args": [
									{
									  "type": "Int",
									  "value": 663712185
									},
									{
									  "type": "String",
									  "value": "8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS"
									}
								  ]
								},
								"payment": [],
								"stateChanges": {
								  "data": [
									{
									  "key": "rpd_balance_8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS",
									  "type": "integer",
									  "value": 1740699712526
									},
									{
									  "key": "rpd_balance_8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS_3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
									  "type": "integer",
									  "value": 336287815
									}
								  ],
								  "transfers": [
									{
									  "address": "3N2GnQeySpP2XZMtufCLo34J7QRnfcTkDmD",
									  "asset": "8UrfDVd5GreeUwm7uPk7eYz1eMv376kzR52C6sANPkwS",
									  "amount": 663712185
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
							  }
							]
						  }
						}
					  ]
					}
				  }
				]
			  }
			}
		  }`,
		`
	{
			"type": 18,
			"id": "2Y67uLthNfzEBpEJFyrP7MKqPYTFYjM5nz2NnETZVUYU",
			"fee": 500000,
			"feeAssetId": null,
			"timestamp": 1634881836984,
			"version": 1,
			"chainId": 83,
			"bytes": "0xf9011186017ca68d17b88502540be4008307a120940ea8e14f313237aac31995f9c19a7e0f78c1cc2b80b8a409abf90e0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000064672696461790000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000081caa0ecb7124f915bd366186a6451aabdde3fbf0db94caa78a6b8d6115bb5ce6407d8a077ab1e756d343b9927c3c4add5c797915aef2de112576213d6a30ce5e040ba3c",
			"sender": "3MRejoFLZ6FsXRjVEzBpnQ27s61FDLLDGxh",
			"senderPublicKey": "3nFhfAYDSRS4UrU22HaAuFT4YHZD5Et3vy7fBTcTxefuAVXs8pHRR4pvpAzvMbmskwjWB7PxFKqPNsioRVZ9mxaa",
			"height": 1042032,
			"applicationStatus": "succeeded",
			"payload": {
			  "type": "invocation",
			  "dApp": "3MRuzZVauiiX2DGwNyP8Tv7idDGUy1VG5bJ",
			  "call": {
				"function": "saveString",
				"args": [
				  {
					"type": "string",
					"value": "Friday"
				  },
				  {
					"type": "Boolean",
					"value": true
				  },
				  {
					"type": "Int",
					"value": 100000000
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
					"key": "str_1042032",
					"type": "string",
					"value": "Friday"
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
		  }`,
	}

	for i, jsonSrc := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			txInfo := new(EthereumTransactionInfo)
			err := json.Unmarshal([]byte(jsonSrc), txInfo)
			require.NoError(t, err, "unmarshal invocation Ethereum transaction info")

			_, ok := txInfo.Payload.(*EthereumTransactionInvocationPayload)
			require.True(t, ok, "payload type of ethereum transaction is wrong")
		})
	}
}

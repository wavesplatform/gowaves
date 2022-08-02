package client

import (
	"encoding/json"
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
	jsonSrc := `{
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
	  }`

	txInfo := new(EthereumTransactionInfo)
	err := json.Unmarshal([]byte(jsonSrc), txInfo)
	require.NoError(t, err, "unmarshal invocation Ethereum transaction info")

	_, ok := txInfo.Payload.(*EthereumTransactionInvocationPayload)
	require.True(t, ok, "payload type of ethereum transaction is wrong")
}

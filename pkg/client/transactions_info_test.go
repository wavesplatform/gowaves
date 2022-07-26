package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestEthereumTransactionInfo(t *testing.T) {
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

	require.Equal(t, 0, txInfo.GetSpentComplexity())
}

package abi

import (
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/metamask"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"sort"
	"strings"
	"testing"
)

func TestTransferWithRideTypes(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedSignature := "transfer(address,uint256)"
	expectedName := "transfer"
	expectedFirstArg := "0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c"
	expectedSecondArg := "209470300000000000000000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseRide(data)
	require.NoError(t, err)

	var addr metamask.Address
	addr.SetBytes(callData.Inputs[0].DecodedValue().(ride.RideBytes))
	require.Equal(t, expectedSignature, callData.Signature)
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, addr.String())
	require.Equal(t, expectedSecondArg, callData.Inputs[1].DecodedValue().(ride.RideBigInt).String())
}

func TestJsonAbi(t *testing.T) {
	expectedJson := `
	[
	  {
		"name":"transfer",
		"type":"function",
		"inputs": [
		  {
			"name":"_to",
			"type":"address"
		  },
		  {
			"name":"_value",
			"type":"uint256"
		  },
		  {
			"name":"",
			"type":"tuple[]",
			"components": [
			  {
				"name": "",
				"type": "address"
			  },
			  {
			    "name": "",
			    "type": "uint256"
			  }
            ]
		  }
		]
	  },
	  {
	    "name":"transferFrom",
		"type":"function",
		"inputs": [
		  {
			"name":"_from",
			"type":"address"
		  },
		  {
			"name":"_to",
			"type":"address"
		  },
		  {
			"name":"_value",
			"type":"uint256"
		  },
		  {
			"name":"",
			"type":"tuple[]",
			"components": [
			  {
				"name": "",
				"type": "address"
			  },
			  {
			    "name": "",
			    "type": "uint256"
			  }
            ]
		  }
		]
	  }
	]
`
	var expectedABI []ABI
	err := json.Unmarshal([]byte(expectedJson), &expectedABI)
	require.NoError(t, err)

	resJsonABI, err := getJsonAbi(fourbyte.Erc20Methods)
	require.NoError(t, err)
	var abi []ABI
	err = json.Unmarshal(resJsonABI, &abi)
	require.NoError(t, err)

	sort.Slice(abi, func(i, j int) bool { return abi[i].Name < abi[j].Name })
	sort.Slice(expectedABI, func(i, j int) bool { return expectedABI[i].Name < expectedABI[j].Name })

	require.Equal(t, expectedABI, abi)
}

package abi

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

func TestTransfer(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedSignature := "transfer(address,uint256)"
	expectedName := "transfer"
	expectedFirstArg := "0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c"
	expectedSecondArg := "209470300000000000000000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseNew(data)
	require.NoError(t, err)

	require.Equal(t, expectedSignature, callData.Signature)
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, callData.Inputs[0].DecodedValue().(fmt.Stringer).String())
	require.Equal(t, expectedSecondArg, callData.Inputs[1].DecodedValue().(*big.Int).String())
}

func TestJsonAbi(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedJson := `[{"name":"transfer","type":"function","inputs":[{"type":"address"},{"type":"uint256"}]}]`

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseNew(data)
	require.NoError(t, err)

	resJson, err := getJsonAbi(callData.Signature)
	require.NoError(t, err)
	require.Equal(t, string(resJson), expectedJson)
}

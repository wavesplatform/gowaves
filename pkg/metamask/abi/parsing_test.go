package abi

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/metamask"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
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

	callData, err := parseNew(data, true)
	// nickeskov: no error because we have zero length bytes data for payments
	require.NoError(t, err)

	require.Equal(t, expectedSignature, callData.Signature)
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, callData.Inputs[0].DecodedValue().(fmt.Stringer).String())
	require.Equal(t, expectedSecondArg, callData.Inputs[1].DecodedValue().(*big.Int).String())
}

func TestRandomFunctionABIParsing(t *testing.T) {
	// taken and modified from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const (
		testSignature = fourbyte.Signature("minta(address,uint256,uint256,uint256,uint256)")
		hexData       = "0xe00c88d6000000000000000000000000892555e75350e11f2058d086c72b9c94c9493d7200000000000000000000000000000000000000000000000000000000000000a50000000000000000000000000000000000000000000000056bc75e2d631000000000000000000000000000000000000000000000000000056bc75e2d63100000000000000000000000000000000000000000000000000000000000000000000a"
	)

	var customDB = map[fourbyte.Selector]fourbyte.Method{
		testSignature.Selector(): {
			RawName: "minta",
			Inputs: fourbyte.Arguments{
				{Name: "_token", Type: fourbyte.Type{T: fourbyte.AddressTy}},
				{Name: "_id", Type: fourbyte.Type{T: fourbyte.UintTy, Size: 256}},
				{Name: "_supply", Type: fourbyte.Type{T: fourbyte.UintTy, Size: 256}},
				{Name: "_listPrice", Type: fourbyte.Type{T: fourbyte.UintTy, Size: 256}},
				{Name: "_fee", Type: fourbyte.Type{T: fourbyte.UintTy, Size: 256}},
			},
			Sig: testSignature,
		},
	}

	data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	require.NoError(t, err)
	db := fourbyte.NewCustomDatabase(customDB)
	callData, err := db.ParseCallDataRide(data, true)
	// nickeskov: no error because we have zero length bytes data for payments
	require.NoError(t, err)

	require.Equal(t, "minta", callData.Name)
	var addr metamask.Address
	addr.SetBytes(callData.Inputs[0].Value.(ride.RideBytes))
	require.Equal(t, "0x892555E75350E11f2058d086C72b9C94C9493d72", addr.String())
	require.Equal(t, "165", callData.Inputs[1].Value.(ride.RideBigInt).String())
	require.Equal(t, "100000000000000000000", callData.Inputs[2].Value.(ride.RideBigInt).String())
	require.Equal(t, "100000000000000000000", callData.Inputs[3].Value.(ride.RideBigInt).String())
	require.Equal(t, "10", callData.Inputs[4].Value.(ride.RideBigInt).String())
}

func TestTransferWithRideTypes(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedSignature := "transfer(address,uint256)"
	expectedName := "transfer"
	expectedFirstArg := "0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c"
	expectedSecondArg := "209470300000000000000000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseRide(data, true)
	// nickeskov: no error because we have zero length bytes data for payments
	require.NoError(t, err)

	var addr metamask.Address
	addr.SetBytes(callData.Inputs[0].DecodedValue().(ride.RideBytes))
	require.Equal(t, expectedSignature, callData.Signature)
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, addr.String())
	require.Equal(t, expectedSecondArg, callData.Inputs[1].DecodedValue().(ride.RideBigInt).String())
}

func TestJsonAbi(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedJson := `[{"name":"transfer","type":"function","inputs":[{"type":"address"},{"type":"uint256"}]}]`

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseNew(data, false)
	require.NoError(t, err)

	resJson, err := getJsonAbi(callData.Signature, callData.Payments)
	require.NoError(t, err)
	require.Equal(t, expectedJson, string(resJson))
}

func TestJsonAbiPayments(t *testing.T) {
	expectedJson := `[{"name":"transfer","type":"function","inputs":[{"type":"address"},{"type":"uint256"},{"type":"(address, uint256)[]"}]}]`

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	callData, err := parseNew(data, false)
	require.NoError(t, err)
	callData.Payments = append(callData.Payments, fourbyte.Payment{})

	resJson, err := getJsonAbi(callData.Signature, callData.Payments)
	require.NoError(t, err)
	require.Equal(t, expectedJson, string(resJson))
}

func TestParsingABIUsingRideMeta(t *testing.T) {
	// hexdata created with https://github.com/rust-ethereum/ethabi

	testdata := []struct {
		rideFunctionMeta     meta.Function
		hexdata              string
		expectedResultValues []ride.RideType
	}{
		{
			rideFunctionMeta: meta.Function{
				Name:      "some_test_fn",
				Arguments: []meta.Type{meta.Boolean, meta.String, meta.String},
			},
			hexdata: "0x7afebf3b0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000861736661736466730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000015657468657265756d2061626920746573742e2e2e2e0000000000000000000000",
			expectedResultValues: []ride.RideType{
				ride.RideBoolean(true), ride.RideString("asfasdfs"), ride.RideString("ethereum abi test...."),
			},
		},
	}
	for _, test := range testdata {
		data, err := hex.DecodeString(strings.TrimPrefix(test.hexdata, "0x"))
		require.NoError(t, err)

		dAppMeta := meta.DApp{
			Version:       1,
			Functions:     []meta.Function{test.rideFunctionMeta},
			Abbreviations: meta.Abbreviations{},
		}
		db, err := fourbyte.NewDBFromRideDAppMeta(dAppMeta)
		require.NoError(t, err)

		decodedCallData, err := db.ParseCallDataRide(data, false)
		require.NoError(t, err)

		values := make([]ride.RideType, 0, len(decodedCallData.Inputs))
		for _, arg := range decodedCallData.Inputs {
			values = append(values, arg.Value.(ride.RideType))
		}
		require.Equal(t, test.expectedResultValues, values)
	}
}

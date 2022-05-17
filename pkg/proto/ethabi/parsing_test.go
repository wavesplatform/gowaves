package ethabi

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

func TestERC20EthereumTransfer(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedSignature := "transfer(address,uint256)"
	expectedName := "transfer"
	expectedFirstArg := strings.ToLower("0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c")
	expectedSecondArg := "209470300000000000000000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)

	erc20Db := NewErc20MethodsMap()
	callData, err := erc20Db.ParseCallDataRide(data)
	require.NoError(t, err)

	require.Equal(t, expectedSignature, callData.Signature.String())
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, fmt.Sprintf("0x%x", callData.Inputs[0].Value.(Bytes)))
	require.Equal(t, expectedSecondArg, callData.Inputs[1].Value.(BigInt).V.String())

	_, err = GetERC20TransferArguments(callData)
	require.Error(t, err)
	require.Equal(t,
		"failed to convert BigInt value to int64 (overflow), value is 209470300000000000000000",
		err.Error(),
	)
}

func TestGetERC20TransferArguments(t *testing.T) {
	expectedFirstArg := strings.ToLower("0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c")
	expectedSecondArg := "31650332672000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c00000000000000000000000000000000000000000000000000001cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)

	erc20Db := NewErc20MethodsMap()
	callData, err := erc20Db.ParseCallDataRide(data)
	require.NoError(t, err)

	transferArgs, err := GetERC20TransferArguments(callData)
	require.NoError(t, err)

	require.Equal(t, expectedFirstArg, fmt.Sprintf("0x%x", transferArgs.Recipient))
	require.Equal(t, expectedSecondArg, fmt.Sprintf("%d", transferArgs.Amount))
}

func TestRandomFunctionABIParsing(t *testing.T) {
	// taken and modified from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const (
		testSignature = Signature("minta(address,uint256,uint256,uint256,uint256)")
		hexData       = "0xe00c88d6000000000000000000000000892555e75350e11f2058d086c72b9c94c9493d7200000000000000000000000000000000000000000000000000000000000000a50000000000000000000000000000000000000000000000056bc75e2d631000000000000000000000000000000000000000000000000000056bc75e2d63100000000000000000000000000000000000000000000000000000000000000000000a"
	)

	var customDB = map[Selector]Method{
		testSignature.Selector(): {
			RawName: "minta",
			Inputs: Arguments{
				{Name: "_token", Type: Type{T: AddressType}},
				{Name: "_id", Type: Type{T: UintType, Size: 256}},
				{Name: "_supply", Type: Type{T: UintType, Size: 256}},
				{Name: "_listPrice", Type: Type{T: UintType, Size: 256}},
				{Name: "_fee", Type: Type{T: UintType, Size: 256}},
			},
			Payments: nil,
			Sig:      testSignature,
		},
	}

	data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	require.NoError(t, err)
	db := MethodsMap{
		methods:       customDB,
		parsePayments: false,
	}
	callData, err := db.ParseCallDataRide(data)
	require.NoError(t, err)

	require.Equal(t, "minta", callData.Name)
	require.Equal(t,
		strings.ToLower("0x892555E75350E11f2058d086C72b9C94C9493d72"),
		fmt.Sprintf("0x%x", callData.Inputs[0].Value.(Bytes)),
	)
	require.Equal(t, "165", callData.Inputs[1].Value.(BigInt).V.String())
	require.Equal(t, "100000000000000000000", callData.Inputs[2].Value.(BigInt).V.String())
	require.Equal(t, "100000000000000000000", callData.Inputs[3].Value.(BigInt).V.String())
	require.Equal(t, "10", callData.Inputs[4].Value.(BigInt).V.String())
}

func TestJsonAbi(t *testing.T) {
	expectedJson := `
[
  {
    "name": "transfer",
    "type": "function",
    "constant": false,
    "payable": false,
    "stateMutability": "nonpayable",
    "outputs": [],
    "inputs": [
      {
        "name": "_to",
        "type": "address"
      },
      {
        "name": "_value",
        "type": "uint256"
      }
    ]
  }
]
`
	var expectedABI []abi
	err := json.Unmarshal([]byte(expectedJson), &expectedABI)
	require.NoError(t, err)

	resJsonABI, err := NewErc20MethodsMap().MarshalJSON()
	require.NoError(t, err)
	var abiRes []abi
	err = json.Unmarshal(resJsonABI, &abiRes)
	require.NoError(t, err)

	sort.Slice(abiRes, func(i, j int) bool { return abiRes[i].Name < abiRes[j].Name })
	sort.Slice(expectedABI, func(i, j int) bool { return expectedABI[i].Name < expectedABI[j].Name })

	require.Equal(t, expectedABI, abiRes)
}

func TestJsonAbiWithAllTypes(t *testing.T) {
	testMethodWithAllTypes := []Method{
		{
			RawName: "testFunction",
			Inputs: Arguments{
				{Name: "stringVar", Type: Type{T: StringType}},
				{Name: "intVar", Type: Type{T: IntType, Size: 64}},
				{Name: "bytesVar", Type: Type{T: BytesType}},
				{Name: "boolVar", Type: Type{T: BoolType}},
				{
					Name: "sliceVar",
					Type: Type{
						T:    SliceType,
						Elem: &Type{T: IntType, Size: 64}},
				},
				{
					Name: "tupleSliceVar",
					Type: Type{
						T: TupleType,
						TupleFields: Arguments{
							{Name: "union_index", Type: Type{T: UintType, Size: 8}},
							{Name: "stringVar", Type: Type{T: StringType}},
							{Name: "boolVar", Type: Type{T: BoolType}},
							{Name: "intVar", Type: Type{T: IntType, Size: 64}},
							{Name: "bytesVar", Type: Type{T: BytesType}},
							{Name: "addrVar", Type: Type{T: AddressType}},
						},
					},
				},
			},
			Payments: &Argument{
				Name: "payments",
				Type: paymentsType,
			},
		},
	}
	expectedJson := `
[
  {
    "name": "testFunction",
    "type": "function",
    "constant": false,
    "payable": false,
    "stateMutability": "nonpayable",
    "outputs": [],
    "inputs": [
      {
        "name": "stringVar",
        "type": "string"
      },
      {
        "name": "intVar",
        "type": "int64"
      },
      {
        "name": "bytesVar",
        "type": "bytes"
      },
      {
        "name": "boolVar",
        "type": "bool"
      },
      {
        "name": "sliceVar",
        "type": "int64[]"
      },
      {
        "name": "tupleSliceVar",
        "type": "tuple",
        "components": [
          {
            "name": "union_index",
            "type": "uint8"
          },
          {
            "name": "stringVar",
            "type": "string"
          },
          {
            "name": "boolVar",
            "type": "bool"
          },
          {
            "name": "intVar",
            "type": "int64"
          },
          {
            "name": "bytesVar",
            "type": "bytes"
          },
          {
            "name": "addrVar",
            "type": "address"
          }
        ]
      },
      {
        "name": "payments",
        "type": "tuple[]",
        "components": [
          {
            "name": "id",
            "type": "bytes32"
          },
          {
            "name": "value",
            "type": "int64"
          }
        ]
      }
    ]
  }
]
`
	var expectedABI []abi
	err := json.Unmarshal([]byte(expectedJson), &expectedABI)
	require.NoError(t, err)

	resJsonABI, err := MakeJsonABI(testMethodWithAllTypes)
	require.NoError(t, err)

	require.JSONEq(t, expectedJson, string(resJsonABI))

	var abiRes []abi
	err = json.Unmarshal(resJsonABI, &abiRes)
	require.NoError(t, err)

	sort.Slice(abiRes, func(i, j int) bool { return abiRes[i].Name < abiRes[j].Name })
	sort.Slice(expectedABI, func(i, j int) bool { return expectedABI[i].Name < expectedABI[j].Name })

	require.Equal(t, expectedABI, abiRes)
}

func TestParsingABIUsingRideMeta(t *testing.T) {
	b1, err := base64.StdEncoding.DecodeString("4BD+OLA=")
	require.NoError(t, err)
	b2, err := base64.StdEncoding.DecodeString("AUWrPlBPQlWXsn4=")
	require.NoError(t, err)
	b3, err := base64.StdEncoding.DecodeString("pimpshwamp21312390AA")
	require.NoError(t, err)
	tests := []struct {
		rideFunctionMeta     meta.Function
		hexdata              string
		expectedResultValues []DataType
		parsePayments        bool
		payments             []Payment
	}{
		// first test hexdata created with https://github.com/rust-ethereum/ethabi
		{
			rideFunctionMeta: meta.Function{
				Name:      "some_test_fn",
				Arguments: []meta.Type{meta.Boolean, meta.String, meta.String},
			},
			hexdata:              "0x7afebf3b0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000861736661736466730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000015657468657265756d2061626920746573742e2e2e2e0000000000000000000000",
			expectedResultValues: []DataType{Bool(true), String("asfasdfs"), String("ethereum abi test....")},
			parsePayments:        false,
			payments:             nil,
		},
		{
			rideFunctionMeta: meta.Function{
				Name:      "call",
				Arguments: []meta.Type{meta.String},
			},
			hexdata:              "0x3e08c22800000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000573616664730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			expectedResultValues: []DataType{String("safds")},
			parsePayments:        true,
			payments:             make([]Payment, 0),
		},
		{
			rideFunctionMeta: meta.Function{
				Name:      "call",
				Arguments: []meta.Type{meta.String, meta.Int, meta.Boolean, meta.Bytes, meta.String, meta.Int, meta.Boolean, meta.Bytes, meta.String, meta.Int, meta.Boolean, meta.Bytes, meta.Int, meta.Int},
			},
			hexdata:              "0x90e0fe3f00000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000022000000000000000000000000000000000000000000000000000000000000002600000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a000000000000000000000000000000000000000000000000000000000000002e0000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000320000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000002e064c32f0000000000000000000000000000000000000000000000000000000000000360000000000000000000000000000000000000000000000000000000000000001cd09fd0b5d180d0b2d0b0d18f20d0b8d0b720d181d182d180d0bed0ba000000000000000000000000000000000000000000000000000000000000000000000005e010fe38b0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001cd092d182d0bed180d0b0d18f20d0b8d0b720d181d182d180d0bed0ba00000000000000000000000000000000000000000000000000000000000000000000000b0145ab3e504f425597b27e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001cd0a2d180d0b5d182d18cd18f20d0b8d0b720d181d182d180d0bed0ba00000000000000000000000000000000000000000000000000000000000000000000000fa629a9b21c1a9a9db5df5db7f740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000011e1a300",
			expectedResultValues: []DataType{String("Первая из строк"), Int(1), Bool(true), Bytes(b1), String("Вторая из строк"), Int(3), Bool(false), Bytes(b2), String("Третья из строк"), Int(3), Bool(true), Bytes(b3), Int(1), Int(12354634543)},
			parsePayments:        true,
			payments:             []Payment{{PresentAssetID: false, AssetID: crypto.Digest{}, Amount: 300000000}},
		},
	}
	for _, tc := range tests {
		data, err := hex.DecodeString(strings.TrimPrefix(tc.hexdata, "0x"))
		require.NoError(t, err)

		dAppMeta := meta.DApp{
			Version:       1,
			Functions:     []meta.Function{tc.rideFunctionMeta},
			Abbreviations: meta.Abbreviations{},
		}
		db, err := newMethodsMapFromRideDAppMeta(dAppMeta, tc.parsePayments)
		require.NoError(t, err)

		decodedCallData, err := db.ParseCallDataRide(data)
		require.NoError(t, err)

		values := make([]DataType, 0, len(decodedCallData.Inputs))
		for _, arg := range decodedCallData.Inputs {
			values = append(values, arg.Value)
		}
		require.Equal(t, tc.expectedResultValues, values)

		if tc.parsePayments {
			require.Equal(t, tc.payments, decodedCallData.Payments)
		}
	}
}

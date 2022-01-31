package ethabi

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

func TestSignature_Selector(t *testing.T) {
	// from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const testSignatureMint = Signature("mint(string,string,address,uint256,uint256,uint256,uint256)")
	require.Equal(t, "0xbdc01110", testSignatureMint.Selector().Hex())

	require.Equal(t, "0xa9059cbb", erc20TransferSelector.String())
}

func TestBuildSignatureFromRideFunctionMeta(t *testing.T) {
	testdata := []struct {
		expectedSig Signature
		metadata    meta.Function
		payments    bool
		success     bool
	}{
		{
			expectedSig: "meta()",
			metadata: meta.Function{
				Name: "meta",
			},
			payments: false,
			success:  true,
		},
		{
			expectedSig: "hardMeta(int64,string,bytes,bool,bool[])",
			metadata: meta.Function{
				Name: "hardMeta",
				Arguments: []meta.Type{
					meta.Int,
					meta.String,
					meta.Bytes,
					meta.Boolean,
					meta.ListType{Inner: meta.Boolean},
				},
			},
			payments: false,
			success:  true,
		},
		{
			expectedSig: "hardMeta(int64,string,bytes,bool,bool[],(uint8,bool,string,bytes,int64)[])",
			metadata: meta.Function{
				Name: "hardMeta",
				Arguments: []meta.Type{
					meta.Int,
					meta.String,
					meta.Bytes,
					meta.Boolean,
					meta.ListType{Inner: meta.Boolean},
					meta.ListType{Inner: meta.UnionType{meta.Boolean, meta.String, meta.Bytes, meta.Int}},
				},
			},
			payments: false,
			success:  false,
		},
		{
			expectedSig: "metaPayments(bool,bytes)",
			metadata:    meta.Function{Name: "metaPayments", Arguments: []meta.Type{meta.Boolean, meta.Bytes}},
			payments:    false,
			success:     true,
		},
		{
			expectedSig: "metaPayments(bool,bytes,(bytes32,int64)[])",
			metadata:    meta.Function{Name: "metaPayments", Arguments: []meta.Type{meta.Boolean, meta.Bytes}},
			payments:    true,
			success:     true,
		},
	}

	for _, test := range testdata {
		actualSig, err := NewSignatureFromRideFunctionMeta(test.metadata, test.payments)
		if test.success {
			require.NoError(t, err)
			require.Equal(t, test.expectedSig, actualSig)
		} else {
			require.Error(t, err)
		}
	}
}

func TestAbiTypeFromRideMetaType(t *testing.T) {
	testdata := []struct {
		expected Type
		metaType meta.Type
		success  bool
	}{
		{
			expected: Type{T: IntType, Size: 64, stringKind: "int64"},
			metaType: meta.Int,
			success:  true,
		},
		{
			expected: Type{T: BoolType, stringKind: "bool"},
			metaType: meta.Boolean,
			success:  true,
		},
		{
			expected: Type{T: StringType, stringKind: "string"},
			metaType: meta.String,
			success:  true,
		},
		{
			expected: Type{T: BytesType, stringKind: "bytes"},
			metaType: meta.Bytes,
			success:  true,
		},
		{
			expected: Type{
				Elem: &Type{
					T:          IntType,
					Size:       64,
					stringKind: "int64",
				},
				T:          SliceType,
				stringKind: "int64[]",
			},
			metaType: meta.ListType{Inner: meta.Int},
			success:  true,
		},
		{
			expected: Type{},
			metaType: meta.UnionType{meta.Int, meta.String},
			success:  false,
		},
		{
			expected: Type{},
			metaType: meta.ListType{Inner: meta.UnionType{meta.Boolean, meta.String, meta.Bytes, meta.Int}},
			success:  false,
		},
	}
	for _, test := range testdata {
		actual, err := AbiTypeFromRideTypeMeta(test.metaType)
		if test.success {
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		} else {
			require.Error(t, err)
		}
	}
}

func TestNewDBFromRideDAppMeta(t *testing.T) {
	dAppMeta := meta.DApp{
		Version: 1,
		Functions: []meta.Function{
			{
				Name:      "func1",
				Arguments: []meta.Type{meta.Int, meta.Boolean},
			},
			{
				Name:      "boba8",
				Arguments: []meta.Type{meta.String, meta.Bytes, meta.ListType{Inner: meta.String}},
			},
			{
				Name: "allKind",
				Arguments: []meta.Type{
					meta.String,
					meta.Int,
					meta.Bytes,
					meta.Boolean,
					meta.ListType{Inner: meta.Int},
				},
			},
			{
				Name:      "withUnion",
				Arguments: []meta.Type{meta.String, meta.UnionType{meta.Int, meta.String}},
			},
			{
				Name:      "withListUnion",
				Arguments: []meta.Type{meta.String, meta.ListType{Inner: meta.UnionType{meta.Int, meta.String}}},
			},
			{
				Name:      "afterUnions",
				Arguments: []meta.Type{},
			},
		},
	}
	expectedFuncs := []Method{
		{
			RawName: "func1",
			Sig:     "func1(int64,bool)",
			Inputs: Arguments{
				{Name: "", Type: Type{Size: 64, T: IntType, stringKind: "int64"}},
				{Name: "", Type: Type{T: BoolType, stringKind: "bool"}},
			},
			Payments: nil,
		},
		{
			RawName: "boba8",
			Sig:     "boba8(string,bytes,string[])",
			Inputs: Arguments{
				{Name: "", Type: Type{T: StringType, stringKind: "string"}},
				{Name: "", Type: Type{T: BytesType, stringKind: "bytes"}},
				{
					Name: "",
					Type: Type{
						T:          SliceType,
						stringKind: "string[]",
						Elem:       &Type{T: StringType, stringKind: "string"}},
				},
			},
			Payments: nil,
		},
		{
			RawName: "allKind",
			Sig:     "allKind(string,int64,bytes,bool,int64[])",
			Inputs: Arguments{
				{Name: "", Type: Type{T: StringType, stringKind: "string"}},
				{Name: "", Type: Type{Size: 64, T: IntType, stringKind: "int64"}},
				{Name: "", Type: Type{T: BytesType, stringKind: "bytes"}},
				{Name: "", Type: Type{T: BoolType, stringKind: "bool"}},
				{
					Name: "",
					Type: Type{
						T:          SliceType,
						stringKind: "int64[]",
						Elem:       &Type{Size: 64, T: IntType, stringKind: "int64"}},
				},
			},
			Payments: nil,
		},
		{
			RawName:  "afterUnions",
			Sig:      "afterUnions()",
			Inputs:   Arguments{},
			Payments: nil,
		},
	}

	db, err := newMethodsMapFromRideDAppMeta(dAppMeta, false)
	require.NoError(t, err)

	for _, expectedFunc := range expectedFuncs {
		actualFunc, err := db.MethodBySelector(expectedFunc.Sig.Selector())
		require.NoError(t, err, "failed while looking function %q", expectedFunc.String())
		require.Equal(t, expectedFunc, actualFunc)
	}
}

func TestUnpackPayment(t *testing.T) {
	tests := []struct {
		hexInput        string
		expectedPayment Payment
	}{
		{
			"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
			Payment{PresentAssetID: false, AssetID: crypto.Digest{}, Amount: 1},
		},
		{
			"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a",
			Payment{PresentAssetID: false, AssetID: crypto.Digest{}, Amount: 10},
		},
		{
			"0x06000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001",
			Payment{PresentAssetID: true, AssetID: crypto.Digest{0x6, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, Amount: 1},
		},
		{
			"0x01000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000009",
			Payment{PresentAssetID: true, AssetID: crypto.Digest{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5}, Amount: 9},
		},
	}
	for _, tc := range tests {
		bts, err := hex.DecodeString(strings.TrimPrefix(tc.hexInput, "0x"))
		require.NoError(t, err)
		actualPayment, err := unpackPayment(bts)
		require.NoError(t, err)

		require.Equal(t, tc.expectedPayment, actualPayment)
	}
}

func TestUnpackPayments(t *testing.T) {
	tests := []struct {
		hexCallData         string
		paymentsSliceOffset int
		expectedPayments    []Payment
	}{
		{
			hexCallData:         "0x3e08c22800000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000573616664730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			paymentsSliceOffset: 32,
			expectedPayments:    make([]Payment, 0),
		},
	}
	for _, tc := range tests {
		bts, err := hex.DecodeString(strings.TrimPrefix(tc.hexCallData, "0x"))
		require.NoError(t, err)

		payments, err := unpackPayments(tc.paymentsSliceOffset, bts[SelectorSize:])
		require.NoError(t, err)
		require.Equal(t, tc.expectedPayments, payments)
	}
}

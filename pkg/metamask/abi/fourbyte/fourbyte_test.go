package fourbyte

import (
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"testing"
)

func TestSignature_Selector(t *testing.T) {
	// from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const testSignatureMint = Signature("mint(string,string,address,uint256,uint256,uint256,uint256)")
	require.Equal(t, "bdc01110", testSignatureMint.Selector().Hex())
}

func TestBuildSignatureFromRideFunctionMeta(t *testing.T) {
	testdata := []struct {
		expectedSig Signature
		metadata    meta.Function
		payments    bool
	}{
		{expectedSig: "meta()", metadata: meta.Function{Name: "meta"}, payments: false},
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
		},
		{
			expectedSig: "metaPayments(bool,bytes,(bytes,int64)[])",
			metadata:    meta.Function{Name: "metaPayments", Arguments: []meta.Type{meta.Boolean, meta.Bytes}},
			payments:    true,
		},
	}

	for _, test := range testdata {
		actualSig, err := NewSignatureFromRideFunctionMeta(test.metadata, test.payments)
		require.NoError(t, err)
		require.Equal(t, test.expectedSig, actualSig)
	}
}

func TestAbiTypeFromRideMetaType(t *testing.T) {
	testdata := []struct {
		expected Type
		metaType meta.Type
	}{
		{expected: Type{T: IntTy, Size: 64, StringKind: "int64"}, metaType: meta.Int},
		{expected: Type{T: BoolTy, StringKind: "bool"}, metaType: meta.Boolean},
		{expected: Type{T: StringTy, StringKind: "string"}, metaType: meta.String},
		{expected: Type{T: BytesTy, StringKind: "bytes"}, metaType: meta.Bytes},
		{
			expected: Type{
				Elem: &Type{
					T:          IntTy,
					Size:       64,
					StringKind: "int64",
				},
				T:          SliceTy,
				StringKind: "int64[]",
			},
			metaType: meta.ListType{Inner: meta.Int}},
		{
			expected: Type{
				Elem: &Type{
					T:          TupleTy,
					StringKind: "(uint8,bool,string,bytes,int64)",
					TupleElems: []Type{
						{T: UintTy, Size: 8, StringKind: "uint8"},
						{T: BoolTy, StringKind: "bool"},
						{T: StringTy, StringKind: "string"},
						{T: BytesTy, StringKind: "bytes"},
						{T: IntTy, Size: 64, StringKind: "int64"},
					},
					TupleRawNames: make([]string, 5),
				},
				T:          SliceTy,
				StringKind: "(uint8,bool,string,bytes,int64)[]",
			},
			metaType: meta.ListType{Inner: meta.UnionType{meta.Boolean, meta.String, meta.Bytes, meta.Int}},
		},
	}
	for _, test := range testdata {
		actual, err := AbiTypeFromRideTypeMeta(test.metaType)
		require.NoError(t, err)
		require.Equal(t, test.expected, actual)
	}
}

func TestNewDBFromRideDAppMeta(t *testing.T) {
	dAppMeta := meta.DApp{
		Version: 1,
		Functions: []meta.Function{
			{Name: "func1", Arguments: []meta.Type{meta.Int, meta.Boolean}},
			{Name: "boba8", Arguments: []meta.Type{meta.String, meta.Bytes, meta.ListType{Inner: meta.String}}},
			{
				Name: "allKind",
				Arguments: []meta.Type{
					meta.String,
					meta.Int,
					meta.Bytes,
					meta.Boolean,
					meta.ListType{Inner: meta.Int},
					meta.UnionType{meta.String, meta.Boolean, meta.Int, meta.Bytes},
				}},
		},
	}
	expectedFuncs := []Method{
		{
			RawName: "func1",
			Sig:     "func1(int64,bool)",
			Inputs: Arguments{
				{Name: "", Type: Type{Size: 64, T: IntTy, StringKind: "int64"}},
				{Name: "", Type: Type{T: BoolTy, StringKind: "bool"}},
			},
			Payments: nil,
		},
		{
			RawName: "boba8",
			Sig:     "boba8(string,bytes,string[])",
			Inputs: Arguments{
				{Name: "", Type: Type{T: StringTy, StringKind: "string"}},
				{Name: "", Type: Type{T: BytesTy, StringKind: "bytes"}},
				{
					Name: "",
					Type: Type{
						T:          SliceTy,
						StringKind: "string[]",
						Elem:       &Type{T: StringTy, StringKind: "string"}},
				},
			},
			Payments: nil,
		},
		{
			RawName: "allKind",
			Sig:     "allKind(string,int64,bytes,bool,int64[],(uint8,string,bool,int64,bytes))",
			Inputs: Arguments{
				{Name: "", Type: Type{T: StringTy, StringKind: "string"}},
				{Name: "", Type: Type{Size: 64, T: IntTy, StringKind: "int64"}},
				{Name: "", Type: Type{T: BytesTy, StringKind: "bytes"}},
				{Name: "", Type: Type{T: BoolTy, StringKind: "bool"}},
				{
					Name: "",
					Type: Type{
						T:          SliceTy,
						StringKind: "int64[]",
						Elem:       &Type{Size: 64, T: IntTy, StringKind: "int64"}},
				},
				{
					Name: "",
					Type: Type{
						T:          TupleTy,
						StringKind: "(uint8,string,bool,int64,bytes)",
						TupleElems: []Type{
							{Size: 8, T: UintTy, StringKind: "uint8"},
							{T: StringTy, StringKind: "string"},
							{T: BoolTy, StringKind: "bool"},
							{Size: 64, T: IntTy, StringKind: "int64"},
							{T: BytesTy, StringKind: "bytes"},
						},
						TupleRawNames: make([]string, 5),
					},
				},
			},
			Payments: nil,
		},
	}

	db, err := NewDBFromRideDAppMeta(dAppMeta, false)
	require.NoError(t, err)

	for _, expectedFunc := range expectedFuncs {
		actualFunc, err := db.MethodBySelector(expectedFunc.Sig.Selector())
		require.NoError(t, err, "failed while looking function %q", expectedFunc.String())
		require.Equal(t, expectedFunc, actualFunc)
	}
}

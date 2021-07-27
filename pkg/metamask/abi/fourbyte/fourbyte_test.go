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
			expectedSig: "metaPayments(bool,bytes,(address,uint64)[])",
			metadata:    meta.Function{Name: "metaPayments", Arguments: []meta.Type{meta.Boolean, meta.Bytes}},
			payments:    true,
		},
	}

	for _, test := range testdata {
		actualSig, err := BuildSignatureFromRideFunctionMeta(test.metadata, test.payments)
		require.NoError(t, err)
		require.Equal(t, test.expectedSig, actualSig)
	}
}

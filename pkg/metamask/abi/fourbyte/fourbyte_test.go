package fourbyte

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSignature_Selector(t *testing.T) {
	// from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const testSignatureMint = Signature("mint(string,string,address,uint256,uint256,uint256,uint256)")
	require.Equal(t, "bdc01110", testSignatureMint.Selector().Hex())
}

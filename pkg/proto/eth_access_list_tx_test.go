package proto

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthereumAccessListTxCanonical(t *testing.T) {
	mustDecodeFomHexString := func(hexString string) []byte {
		data, err := DecodeFromHexString(hexString)
		require.NoError(t, err)
		return data
	}
	const expectedCanonical = "0x01f8630103018261a894b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a825544c001a0c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b2660a032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d37521"
	var (
		expectedCanonicalBytes = mustDecodeFomHexString(expectedCanonical)
		testAddrBytes          = mustDecodeFomHexString("0xb94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	)

	var testAddr EthereumAddress
	copy(testAddr[:], testAddrBytes)

	var v, r, s big.Int
	v.SetInt64(1)
	r.SetString("91059096689536776704183241450754166041908571671439170983238491105279368439392", 10)
	s.SetString("23043059890712937631597966600904561206765736472665286406396949327530054350113", 10)

	inner := &EthereumAccessListTx{
		ChainID:  big.NewInt(1),
		Nonce:    3,
		To:       &testAddr,
		Value:    big.NewInt(10),
		Gas:      25000,
		GasPrice: big.NewInt(1),
		Data:     mustDecodeFomHexString("5544"),
		V:        &v,
		R:        &r,
		S:        &s,
	}

	t.Run("EncodeCanonical", func(t *testing.T) {
		ethTx := EthereumTransaction{inner: inner}

		bts, err := ethTx.EncodeCanonical()
		require.NoError(t, err)

		actualCanonicalBytes := EncodeToHexString(bts)
		require.Equal(t, expectedCanonical, actualCanonicalBytes)
	})

	t.Run("DecodeCanonical", func(t *testing.T) {
		var ethTx EthereumTransaction
		err := ethTx.DecodeCanonical(expectedCanonicalBytes)
		require.NoError(t, err)
		require.Equal(t, inner, ethTx.inner)
	})
}

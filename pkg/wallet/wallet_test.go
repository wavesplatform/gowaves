package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO remove this test.
func TestWallet_EncodeDecode(t *testing.T) {
	password := []byte("123456")

	w := NewWallet()
	err := w.AddAccountSeed([]byte("exile region inmate brass mobile hour best spy gospel gown grace actor armed gift radar"))
	require.NoError(t, err)

	bts, err := w.Encode(password)
	require.NoError(t, err)

	w2, err := Decode(bts, password)
	require.NoError(t, err)
	assert.Equal(t, w.AccountSeeds(), w2.AccountSeeds())

	_, err = Decode(bts, []byte("unknown password"))
	require.Error(t, err)
}

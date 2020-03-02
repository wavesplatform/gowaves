package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWallet_EncodeDecode(t *testing.T) {
	password := []byte("123456")

	w := NewWallet()
	err := w.AddSeed([]byte("exile region inmate brass mobile hour best spy gospel gown grace actor armed gift radar"))
	require.NoError(t, err)

	bts, err := w.Encode(password)
	require.NoError(t, err)

	w2, err := Decode(bts, password)
	require.NoError(t, err)
	assert.Equal(t, w.Seeds(), w2.Seeds())

	_, err = Decode(bts, []byte("unknown password"))
	require.Error(t, err)
}

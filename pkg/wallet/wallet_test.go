package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWallet_EncodeDecode(t *testing.T) {
	password := []byte("123456")

	w := NewWallet()
	err := w.AddAccountSeed([]byte("BjABqgaAhXb13wAcjUNBv1hzaxDmzT9hdQC3uCzZGmVmL79kzru97FKVnz8jKvpEFECwTxZvMGZxEhfGteDuhGL2euXMt9UuopJH1x9ti52Nmz5uuigU4Wm"))
	require.NoError(t, err)

	bts, err := w.Encode(password)
	require.NoError(t, err)

	w2, err := Decode(bts, password)
	require.NoError(t, err)
	assert.Equal(t, w.AccountSeeds(), w2.AccountSeeds())

	_, err = Decode(bts, []byte("unknown password"))
	require.Error(t, err)
}

func TestWallet_EncodeDecodeMultipleAccountSeeds(t *testing.T) {
	password := []byte("123456")

	w := NewWallet()
	err := w.AddAccountSeed([]byte("BjABqgaAhXb13wAcjUNBv1hzaxDmzT9hdQC3uCzZ"))
	require.NoError(t, err)
	err = w.AddAccountSeed([]byte("MGZxEhfGteDuhGL2euXMt9UuopJH1x9ti52Nmz5uuigU4Wm"))
	require.NoError(t, err)
	err = w.AddAccountSeed([]byte("GmVmL79kzru97FKVnz8jKvpEFECwTxZv"))
	require.NoError(t, err)

	bts, err := w.Encode(password)
	require.NoError(t, err)

	w2, err := Decode(bts, password)
	require.NoError(t, err)
	assert.Equal(t, w.AccountSeeds(), w2.AccountSeeds())

	_, err = Decode(bts, []byte("unknown password"))
	require.Error(t, err)
}

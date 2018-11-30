package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestWallet_EncodeDecode(t *testing.T) {
	password := []byte("123456")

	w, err := NewWalletFromSeed([]byte("exile region inmate brass mobile hour best spy gospel gown grace actor armed gift radar"))
	require.NoError(t, err)

	bts, err := w.Encode(password)
	require.NoError(t, err)

	w2, err := Decode(bts, password)
	require.NoError(t, err)
	assert.Equal(t, w.Seed(), w2.Seed())

	_, err = Decode(bts, []byte("unknown password"))
	require.Error(t, err)

	_, public, err := w.GenPair()
	require.NoError(t, err)

	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, public)
	require.NoError(t, err)

	assert.Equal(t, "3MqBoAUmn1XKCebApkszLSUqFpGf5yQZqeL", addr.String())
}

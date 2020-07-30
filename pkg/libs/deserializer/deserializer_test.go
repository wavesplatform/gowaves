package deserializer

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestDeserializer_Byte(t *testing.T) {
	b := []byte{4}
	d := NewDeserializer(b)
	t.Run("valid", func(t *testing.T) {
		rs, err := d.Byte()
		require.NoError(t, err)
		require.EqualValues(t, 4, rs)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := d.Byte()
		require.Error(t, err)
	})
}

func TestDeserializer_Signature(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("3SfsgXd34pi663APrFi4ewhBDmhdJBV14QPDWGYrHA83jYgTqGdNBqCBMGJjg9M76SjvZhYzzsrPtEWrRUXhNgN6")
	b := sig.Bytes()
	d := NewDeserializer(b)
	t.Run("valid", func(t *testing.T) {
		rs, err := d.Signature()
		require.NoError(t, err)
		require.EqualValues(t, sig, rs)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := d.Signature()
		require.Error(t, err)
	})
}

func TestDeserializer_Uint32(t *testing.T) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, 100500)
	d := NewDeserializer(b)
	t.Run("valid", func(t *testing.T) {
		rs, err := d.Uint32()
		require.NoError(t, err)
		require.EqualValues(t, 100500, rs)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := d.Uint32()
		require.Error(t, err)
	})
}

func TestDeserializer_Bytes(t *testing.T) {
	b := []byte{1, 2, 3}
	d := NewDeserializer(b)
	t.Run("valid", func(t *testing.T) {
		rs, err := d.Bytes(3)
		require.NoError(t, err)
		require.EqualValues(t, b, rs)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := d.Bytes(3)
		require.Error(t, err)
	})
}

func TestDeserializer_PublicKey(t *testing.T) {
	public, _ := crypto.NewPublicKeyFromBase58("Ao159h5j1piHBhoEbCAYyaiKNd6uoKvcdwzRZF9za3Vv")
	d := NewDeserializer(public.Bytes())
	t.Run("valid", func(t *testing.T) {
		rs, err := d.PublicKey()
		require.NoError(t, err)
		require.EqualValues(t, public, rs)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := d.PublicKey()
		require.Error(t, err)
	})
}

func TestDeserializer_ByteStringWithUint16Len(t *testing.T) {
	_, err := NewDeserializer(nil).ByteStringWithUint16Len()
	require.Error(t, err)
}

func TestDeserializer_Uint64(t *testing.T) {
	_, err := NewDeserializer(nil).Uint64()
	require.Error(t, err)
}

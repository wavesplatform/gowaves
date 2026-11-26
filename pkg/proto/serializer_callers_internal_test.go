package proto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

func TestFunctionCallSerializeMatchesMarshalBinary(t *testing.T) {
	calls := []FunctionCall{
		{},
		NewFunctionCall("default", Arguments{}),
		NewFunctionCall("call", Arguments{NewIntegerArgument(-7), NewStringArgument("abc")}),
	}
	for _, c := range calls {
		expected, err := c.MarshalBinary()
		require.NoError(t, err)
		buf := &bytes.Buffer{}
		s := serializer.New(buf)
		require.NoError(t, c.Serialize(s))
		require.Equal(t, expected, buf.Bytes())
		require.EqualValues(t, len(expected), s.N())
	}
}

func TestFunctionCallSerializeHeaderDoesNotAllocate(t *testing.T) {
	c := NewFunctionCall("f", Arguments{})
	s := serializer.New(io.Discard)
	allocs := testing.AllocsPerRun(100, func() {
		_ = c.Serialize(s)
	})
	require.Zero(t, allocs)
}

func TestTransferWithProofsBodySerializeMatchesBodyMarshalBinary(t *testing.T) {
	sk, pk, err := crypto.GenerateKeyPair([]byte("transfer-body-serialize"))
	require.NoError(t, err)
	addr, err := NewAddressFromPublicKey(TestNetScheme, pk)
	require.NoError(t, err)
	asset := NewOptionalAssetFromDigest(crypto.MustDigestFromBase58("B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"))
	for _, v := range []byte{2, 3} {
		for _, aa := range []OptionalAsset{NewOptionalAssetWaves(), *asset} {
			tx := NewUnsignedTransferWithProofs(v, pk, aa, NewOptionalAssetWaves(), 1_700_000_000_000, 100, 10,
				NewRecipientFromAddress(addr), Attachment("att"))
			require.NoError(t, tx.Sign(TestNetScheme, sk))
			if v >= 3 {
				// Protobuf versions have no binary body, only check the serializer prefix is written correctly.
				buf := &bytes.Buffer{}
				require.NoError(t, tx.BodySerialize(serializer.New(buf)))
				require.Equal(t, []byte{byte(TransferTransaction), v}, buf.Bytes()[:2])
				continue
			}
			expected, bErr := tx.BodyMarshalBinary(TestNetScheme)
			require.NoError(t, bErr)
			buf := &bytes.Buffer{}
			s := serializer.New(buf)
			require.NoError(t, tx.BodySerialize(s))
			require.Equal(t, expected, buf.Bytes())
			require.EqualValues(t, len(expected), s.N())

			full, mErr := tx.MarshalBinary(TestNetScheme)
			require.NoError(t, mErr)
			buf.Reset()
			require.NoError(t, tx.Serialize(serializer.New(buf)))
			require.Equal(t, full, buf.Bytes())
		}
	}
}

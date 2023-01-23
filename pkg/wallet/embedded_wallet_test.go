package wallet

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

type seederTest []byte

func (a seederTest) AccountSeeds() [][]byte {
	return [][]byte{a}
}

func TestEmbeddedWalletImpl_SignTransactionWith(t *testing.T) {
	_, pub, err := crypto.GenerateKeyPair([]byte("test"))
	require.NoError(t, err)

	t.Run("sign successful", func(t *testing.T) {
		tx := byte_helpers.TransferWithSig.Transaction.Clone()
		tx.SenderPK = pub

		w := NewEmbeddedWallet(nil, seederTest("test"), proto.TestNetScheme)
		err = w.SignTransactionWith(pub, tx)
		require.NoError(t, err)
	})

	t.Run("sign failure", func(t *testing.T) {
		tx := byte_helpers.TransferWithSig.Transaction.Clone()
		tx.SenderPK = pub

		w := NewEmbeddedWallet(nil, seederTest("test"), proto.TestNetScheme)
		err = w.SignTransactionWith(crypto.PublicKey{}, tx)
		require.Equal(t, PublicKeyNotFound, err)
	})
}

type testLoader struct {
	bts []byte
	err error
}

func (a testLoader) Load() ([]byte, error) {
	return a.bts, a.err
}

func TestEmbeddedWalletImpl_Load(t *testing.T) {
	wal := NewWallet()
	_ = wal.AddAccountSeed([]byte("seed"))
	bts, err := wal.Encode([]byte("pass"))
	require.NoError(t, err)

	t.Run("successful", func(t *testing.T) {
		w := NewEmbeddedWallet(testLoader{bts: bts}, nil, proto.TestNetScheme)
		require.NoError(t, w.Load([]byte("pass")))
		require.Equal(t, [][]byte{[]byte("seed")}, w.AccountSeeds())
	})

	t.Run("failure", func(t *testing.T) {
		w := NewEmbeddedWallet(testLoader{bts: bts}, nil, proto.TestNetScheme)
		require.Errorf(t, w.Load([]byte("incorrect")), "invalid password")
	})

	t.Run("load error", func(t *testing.T) {
		w := NewEmbeddedWallet(testLoader{bts: nil, err: errors.New("loaderr")}, nil, proto.TestNetScheme)
		require.Errorf(t, w.Load(nil), "loaderr")
	})
}

package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestTransferConvert(t *testing.T) {
	addr, err := NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	assert.NoError(t, err)
	// Test unsigned.
	waves := OptionalAsset{Present: false}
	tx := NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, NewRecipientFromAddress(addr), "attachment")
	err = tx.GenerateID()
	assert.NoError(t, err)
	txProto, err := tx.ToProtobuf(MainNetScheme)
	assert.NoError(t, err)
	var c ProtobufConverter
	resTx, err := c.Transaction(txProto)
	assert.NoError(t, err)
	assert.Equal(t, tx, resTx)

	// Test signed.
	err = tx.Sign(sk)
	assert.NoError(t, err)
	txProtoSigned, err := tx.ToProtobufSigned(MainNetScheme)
	assert.NoError(t, err)
	resTx, err = c.SignedTransaction(txProtoSigned)
	assert.NoError(t, err)
	assert.Equal(t, tx, resTx)
}

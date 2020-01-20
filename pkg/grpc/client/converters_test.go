package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestTransferConvert(t *testing.T) {
	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	assert.NoError(t, err)
	// Test unsigned.
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferV1(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), "attachment")
	tx.GenerateID()
	txProto, err := tx.ToProtobuf(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err)
	var c SafeConverter
	resTx, err := c.Transaction(txProto)
	assert.NoError(t, err)
	assert.Equal(t, tx, resTx)

	// Test signed.
	err = tx.Sign(sk)
	assert.NoError(t, err)
	txProtoSigned, err := tx.ToProtobufSigned(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err)
	resTx, err = c.SignedTransaction(txProtoSigned)
	assert.NoError(t, err)
	assert.Equal(t, tx, resTx)
}

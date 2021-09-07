package state

import (
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math/big"
	"testing"
)



func TestEthereumTransferWaves(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txHandler, err := newTransactionHandler(genBlockId('1'), nil, nil)
	store := blockchainEntitiesStorage{features: &features{}}
	assert.NoError(t, err)
	txAppender := txAppender{
		txHandler: txHandler,
		stor: &store,
	}

	senderPK, err := proto.NewEthereumPublicKeyFromHexString("0xc4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)

	recipientBytes, err := base58.Decode("241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	recipient := proto.BytesToEthereumAddress(recipientBytes)
	ethereumTxData := &proto.EthereumLegacyTx{
		Value: big.NewInt(100),
		To: &recipient,
		Data: nil,
		GasPrice: big.NewInt(1),
		Nonce: 2,
		Gas: 100,
	}
	tx := proto.EthereumTransaction{
		Inner:    ethereumTxData,
		TxKind:   &proto.EthereumTransferWavesTx{},
		ID:       nil,
		SenderPK: senderPK,
	}

	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	fmt.Println(applRes)
}

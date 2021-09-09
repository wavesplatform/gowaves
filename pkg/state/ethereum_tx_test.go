package state

import (
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"math/big"
	"testing"
)

func defaultTxAppender(t *testing.T) txAppender {
	txHandler, err := newTransactionHandler(genBlockId('1'), nil, &settings.BlockchainSettings{FunctionalitySettings: settings.FunctionalitySettings{CheckTempNegativeAfterTime: 1, AllowLeasedBalanceTransferUntilTime: 1}})
	assert.NoError(t, err)
	var feautures = &MockFeaturesState{
		newestIsActivatedFunc: func(featureID int16) (bool, error) {
			return false, nil
		},
	}
	store := blockchainEntitiesStorage{features: feautures}
	txAppender := txAppender{
		txHandler:   txHandler,
		stor:        &store,
		blockDiffer: &blockDiffer{handler: txHandler, settings: &settings.BlockchainSettings{}},
	}
	return txAppender
}

func defaultEthereumLegacyTxData(value int64, to *proto.EthereumAddress) *proto.EthereumLegacyTx {
	return &proto.EthereumLegacyTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     nil,
		GasPrice: big.NewInt(1),
		Nonce:    2,
		Gas:      100,
	}
}

func defaultEthereumDynamicFeeTx(value int64, to *proto.EthereumAddress) *proto.EthereumDynamicFeeTx {
	return &proto.EthereumDynamicFeeTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     nil,
		Nonce:    2,
		Gas:      100,
	}
}

func defaultEthereumAccessListTx(value int64, to *proto.EthereumAddress) *proto.EthereumAccessListTx {
	return &proto.EthereumAccessListTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     nil,
		GasPrice: big.NewInt(1),
		Nonce:    2,
		Gas:      100,
	}
}

func TestEthereumTransferWaves(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t)

	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)

	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	var TxSeveralData []proto.EthereumTxData
	TxSeveralData = append(TxSeveralData, defaultEthereumLegacyTxData(1000000000000000, &recipientEth), defaultEthereumDynamicFeeTx(1000000000000000, &recipientEth), defaultEthereumAccessListTx(1000000000000000, &recipientEth))


	for _, txData := range TxSeveralData {

		tx := proto.EthereumTransaction{
			Inner:    txData,
			TxKind:   &proto.EthereumTransferWavesTx{},
			ID:       nil,
			SenderPK: senderPK,
		}

		applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
		assert.NoError(t, err)
		assert.True(t, applRes.status)

		sender, err := tx.SenderPK.EthereumAddress().ToWavesAddress(0)
		assert.NoError(t, err)

		wavesAsset := proto.NewOptionalAssetWaves()

		senderKey := byteKey(sender, wavesAsset.ToID())

		recipient, err := recipientEth.ToWavesAddress(0)
		assert.NoError(t, err)
		recipientKey := byteKey(recipient, wavesAsset.ToID())

		senderBalance := applRes.changes.diff[string(senderKey)].balance
		recipientBalance := applRes.changes.diff[string(recipientKey)].balance

		assert.Equal(t, senderBalance, int64(-100100))
		assert.Equal(t, recipientBalance, int64(100000))
	}
}



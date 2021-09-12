package state

import (
	"encoding/hex"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state/ethabi"
	"github.com/wavesplatform/gowaves/pkg/types"
	"math/big"
	"strings"
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

func defaultEthereumLegacyTxData(value int64, to *proto.EthereumAddress, data []byte) *proto.EthereumLegacyTx {
	return &proto.EthereumLegacyTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
		GasPrice: big.NewInt(1),
		Nonce:    2,
		Gas:      100,
	}
}

func defaultEthereumDynamicFeeTx(value int64, to *proto.EthereumAddress, data []byte) *proto.EthereumDynamicFeeTx {
	return &proto.EthereumDynamicFeeTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
		Nonce:    2,
		Gas:      100,
	}
}

func defaultEthereumAccessListTx(value int64, to *proto.EthereumAddress, data []byte) *proto.EthereumAccessListTx {
	return &proto.EthereumAccessListTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
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

	var TxInnerSeveralData []proto.EthereumTxData
	TxInnerSeveralData = append(TxInnerSeveralData, defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil), defaultEthereumDynamicFeeTx(1000000000000000, &recipientEth, nil), defaultEthereumAccessListTx(1000000000000000, &recipientEth, nil))


	for _, txInnerData := range TxInnerSeveralData {

		tx := proto.EthereumTransaction{
			Inner:    txInnerData,
			ID:       nil,
			SenderPK: senderPK,
		}

		tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, nil)
		assert.NoError(t, err)

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

func smartStateDappFromDapp() types.SmartState {
	return &ride.MockSmartState{
		NewestLeasingInfoFunc: func(id crypto.Digest) (*proto.LeaseInfo, error) {
			return nil, nil
		},
	}
}

func TestEthereumTransferAssets(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t)
	txAppender.state =
	//
	//state := &ride.MockSmartState{
	//	NewestLeasingInfoFunc: func(id crypto.Digest) (*proto.LeaseInfo, error) {
	//		return nil, nil
	//	},
	//}
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)

	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	//var TxSeveralData []proto.EthereumTxData
	//TxSeveralData = append(TxSeveralData, defaultEthereumLegacyTxData(1000000000000000, &recipientEth), defaultEthereumDynamicFeeTx(1000000000000000, &recipientEth), defaultEthereumAccessListTx(1000000000000000, &recipientEth))

	/*
		from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5
		transfer(address,uint256)
		1 = 0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c
		2 = 209470300000000000000000
	*/
	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)


	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, data)
		tx := proto.EthereumTransaction{
			Inner:    txData,
			ID:       nil,
			SenderPK: senderPK,
		}
		db := ethabi.NewDatabase(nil)
		assert.NotNil(t, tx.Data())
		decodedData, err := db.ParseCallDataRide(tx.Data(), true)
		tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, decodedData)
		appendTxParams.decodedAbiData = decodedData

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



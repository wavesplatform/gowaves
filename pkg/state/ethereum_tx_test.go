package state

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
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

func defaultTxAppender(t *testing.T, storage ScriptStorageState, state types.SmartState) txAppender {
	var feautures = &MockFeaturesState{
		newestIsActivatedFunc: func(featureID int16) (bool, error) {
			return false, nil
		},
	}
	store := blockchainEntitiesStorage{features: feautures, scriptsStorage: storage}
	blockchainSettings := &settings.BlockchainSettings{FunctionalitySettings: settings.FunctionalitySettings{CheckTempNegativeAfterTime: 1, AllowLeasedBalanceTransferUntilTime: 1}}
	txHandler, err := newTransactionHandler(genBlockId('1'), &store, blockchainSettings, state)
	assert.NoError(t, err)

	blockchainEntitiesStor := blockchainEntitiesStorage{scriptsStorage: storage}
	txAppender := txAppender{
		txHandler:   txHandler,
		stor:        &store,
		state:       state,
		blockDiffer: &blockDiffer{handler: txHandler, settings: &settings.BlockchainSettings{}},
		ia:          &invokeApplier{sc: &scriptCaller{stor: &store, state: state, settings: blockchainSettings}, blockDiffer: &blockDiffer{stor: &store, handler: txHandler, settings: blockchainSettings}, state: state, txHandler: txHandler, settings: blockchainSettings, stor: &blockchainEntitiesStor, invokeDiffStor: &diffStorageWrapped{invokeDiffsStor: &diffStorage{changes: []balanceChanges{}}}},
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
		Value: big.NewInt(value),
		To:    to,
		Data:  data,
		Nonce: 2,
		Gas:   100,
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
	txAppender := defaultTxAppender(t, nil, nil)

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

func lessenDecodedDataAmount(t *testing.T, decodedData *ethabi.DecodedCallData) {
	v, ok := decodedData.Inputs[1].Value.(ride.RideBigInt)
	assert.True(t, ok)
	res := new(big.Int).Div(v.V, big.NewInt(int64(diffEthWaves)))
	decodedData.Inputs[1].Value = ride.RideBigInt{V: res}
}

func TestEthereumTransferAssets(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
	}
	txAppender := defaultTxAppender(t, nil, state)

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
	assert.NoError(t, err)
	lessenDecodedDataAmount(t, decodedData)
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, decodedData)
	assert.NoError(t, err)
	appendTxParams.decodedAbiData = decodedData

	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	assert.True(t, applRes.status)

	sender, err := tx.SenderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)

	wavesAsset := proto.NewOptionalAssetWaves()

	senderWavesKey := byteKey(sender, wavesAsset.ToID())

	txKindTransferAssets := tx.TxKind.(*proto.EthereumTransferAssetsErc20Tx)

	senderKey := byteKey(sender, txKindTransferAssets.Asset.ToID())

	recipient, err := recipientEth.ToWavesAddress(0)
	assert.NoError(t, err)
	rideEthRecipientAddress, ok := decodedData.Inputs[0].Value.(ride.RideBytes)
	assert.True(t, ok)
	ethRecipientAddress := proto.BytesToEthereumAddress(rideEthRecipientAddress)
	recipient, err = ethRecipientAddress.ToWavesAddress(0)
	assert.NoError(t, err)
	recipientKey := byteKey(recipient, txKindTransferAssets.Asset.ToID())

	senderWavesBalance := applRes.changes.diff[string(senderWavesKey)].balance
	senderBalance := applRes.changes.diff[string(senderKey)].balance
	recipientBalance := applRes.changes.diff[string(recipientKey)].balance

	assert.Equal(t, senderWavesBalance, int64(-100))
	assert.Equal(t, senderBalance, int64(-20947030000000))
	assert.Equal(t, recipientBalance, int64(20947030000000))

}

func defaultDecodedData(name string, arguments []ethabi.DecodedArg, payments []ethabi.Payment) ethabi.DecodedCallData {
	var decodedData ethabi.DecodedCallData
	decodedData.Name = name
	decodedData.Inputs = arguments
	decodedData.Payments = payments
	return decodedData
}

func applyScript(t *testing.T, tx *proto.EthereumTransaction, stor ScriptStorageState, info *fallibleValidationParams) (proto.WavesAddress, *ride.Tree) {
	scriptAddr, err := tx.WavesAddressTo(0)
	require.NoError(t, err)
	tree, err := stor.newestScriptByAddr(scriptAddr, !info.initialisation)
	require.NoError(t, err)
	return scriptAddr, tree
}

func TestEthereumInvoke(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress, filter bool) (*ride.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}

			@Callable(i)
			func call(number: Int) = {
			  [
			    IntegerEntry("int", number)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAEAAAAGbnVtYmVyCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQFAAAABm51bWJlcgUAAAADbmlsAAAAAE5VO+E=")
		require.NoError(t, err)

		tree, err := ride.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)

		return tree, nil
	}
	storage := &MockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress, filter bool) (crypto.PublicKey, error) {
			return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
		},
	}
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil)
	tx := proto.EthereumTransaction{
		Inner:    txData,
		ID:       nil,
		SenderPK: senderPK,
	}

	decodedData := defaultDecodedData("call", []ethabi.DecodedArg{{Value: ride.RideInt(10)}}, []ethabi.Payment{{Amount: 5}})
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, &decodedData)
	assert.NoError(t, err)
	appendTxParams.decodedAbiData = &decodedData

	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}

	scriptAddress, tree := applyScript(t, &tx, storage, fallibleInfo)
	fallibleInfo.rideV5Activated = true
	ok, actions, err := txAppender.ia.sc.ethereumInvokeFunction(tree, &tx, fallibleInfo, scriptAddress)
	assert.NoError(t, err)
	assert.True(t, ok)
	fmt.Println(actions)

	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 10}},
	}

	assert.Equal(t, expectedDataEntryWrites[0], actions[0])

}

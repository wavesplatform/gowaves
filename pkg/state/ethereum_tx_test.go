package state

import (
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func defaultTxAppender(t *testing.T, storage scriptStorageState, state types.SmartState, assetsUncertain map[proto.AssetID]assetInfo, scheme proto.Scheme) txAppender {
	activatedFeatures := map[settings.Feature]struct{}{
		settings.SmartAccounts:  {},
		settings.FeeSponsorship: {},
		settings.Ride4DApps:     {},
		settings.RideV6:         {},
	}
	feat := &mockFeaturesState{
		newestIsActivatedFunc: func(featureID int16) (bool, error) {
			_, ok := activatedFeatures[settings.Feature(featureID)]
			return ok, nil
		},
		newestIsActivatedForNBlocksFunc: func(featureID int16, n int) (bool, error) {
			const (
				expectedFeature = int16(settings.NG)
				expectedN       = 1
			)
			if featureID == expectedFeature && n == expectedN {
				return true, nil
			}
			return false, errors.Errorf("unexpected values: got (featureID=%d,n=%d), want (featureID=%d,n=%d)",
				featureID, n, expectedFeature, expectedN,
			)
		},
	}
	sett := *settings.MainNetSettings
	sett.SponsorshipSingleActivationPeriod = true
	stor := createStorageObjectsWithOptions(t, testStorageObjectsOptions{
		Settings: &sett,
	})
	newAssets := newAssets(stor.db, stor.dbBatch, stor.hs)
	if assetsUncertain == nil {
		assetsUncertain = make(map[proto.AssetID]assetInfo)
	}
	newAssets.uncertainAssetInfo = assetsUncertain

	store := blockchainEntitiesStorage{features: feat, scriptsStorage: storage, sponsoredAssets: &sponsoredAssets{features: feat, settings: &sett}, assets: newAssets}
	blockchainSettings := &settings.BlockchainSettings{FunctionalitySettings: settings.FunctionalitySettings{CheckTempNegativeAfterTime: 1, AllowLeasedBalanceTransferUntilTime: 1, AddressSchemeCharacter: scheme}}
	txHandler, err := newTransactionHandler(genBlockId('1'), &store, blockchainSettings)
	assert.NoError(t, err)
	blockchainEntitiesStor := blockchainEntitiesStorage{scriptsStorage: storage}
	txAppender := txAppender{
		txHandler:   txHandler,
		stor:        &store,
		ethInfo:     newEthInfo(&store, blockchainSettings),
		blockDiffer: &blockDiffer{handler: txHandler, settings: &settings.BlockchainSettings{}},
		ia:          &invokeApplier{sc: &scriptCaller{stor: &store, state: state, settings: blockchainSettings}, blockDiffer: &blockDiffer{stor: &store, handler: txHandler, settings: blockchainSettings}, state: state, txHandler: txHandler, settings: blockchainSettings, stor: &blockchainEntitiesStor, invokeDiffStor: &diffStorageWrapped{invokeDiffsStor: &diffStorage{changes: []balanceChanges{}}}},
	}
	return txAppender
}
func defaultEthereumLegacyTxData(value int64, to *proto.EthereumAddress, data []byte, gas uint64, scheme proto.Scheme) *proto.EthereumLegacyTx {
	v := big.NewInt(int64(scheme)) // TestNet byte
	v.Mul(v, big.NewInt(2))
	v.Add(v, big.NewInt(35))

	return &proto.EthereumLegacyTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
		GasPrice: big.NewInt(int64(proto.EthereumGasPrice)),
		Nonce:    1479168000000,
		Gas:      gas,
		V:        v,
	}
}
func TestEthereumTransferWaves(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	storage := &mockScriptStorageState{
		newestAccountHasVerifierFunc: func(addr proto.WavesAddress) (bool, error) {
			return false, nil
		},
	}
	//assetsUncertain := newAssets
	txAppender := defaultTxAppender(t, storage, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 100000, proto.TestNetScheme)
	tx := proto.NewEthereumTransaction(txData, nil, nil, &senderPK, 0)
	tx.TxKind, err = txAppender.ethInfo.ethereumTransactionKind(&tx, nil)

	assert.NoError(t, err)
	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	assert.True(t, applRes.status)

	sender, err := senderPK.EthereumAddress().ToWavesAddress(proto.TestNetScheme)
	assert.NoError(t, err)

	wavesAsset := proto.NewOptionalAssetWaves()
	senderKey := byteKey(sender.ID(), wavesAsset)
	recipient, err := recipientEth.ToWavesAddress(proto.TestNetScheme)
	assert.NoError(t, err)
	recipientKey := byteKey(recipient.ID(), wavesAsset)
	senderBalance := applRes.changes.diff[string(senderKey)].balance
	recipientBalance := applRes.changes.diff[string(recipientKey)].balance
	assert.Equal(t, senderBalance, int64(-200000))
	assert.Equal(t, recipientBalance, int64(100000))

}
func lessenDecodedDataAmount(t *testing.T, decodedData *ethabi.DecodedCallData) {
	v, ok := decodedData.Inputs[1].Value.(ethabi.BigInt)
	assert.True(t, ok)
	res := new(big.Int).Div(v.V, big.NewInt(int64(proto.DiffEthWaves)))
	decodedData.Inputs[1].Value = ethabi.BigInt{V: res}
}

func TestEthereumTransferAssets(t *testing.T) {
	storage := &mockScriptStorageState{
		newestScriptBasicInfoByAddressIDFunc: func(id proto.AddressID) (scriptBasicInfoRecord, error) {
			return scriptBasicInfoRecord{PK: crypto.MustPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")}, nil
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID) (bool, error) {
			return false, nil
		},
		newestAccountHasVerifierFunc: func(addr proto.WavesAddress) (bool, error) {
			return false, nil
		},
	}

	appendTxParams := defaultAppendTxParams()
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	assetsUncertain := map[proto.AssetID]assetInfo{
		proto.AssetID(recipientEth): {},
	}
	txAppender := defaultTxAppender(t, storage, &AnotherMockSmartState{}, assetsUncertain, proto.TestNetScheme)
	/*
		from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5
		transfer(address,uint256)
		1 = 0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c
		2 = 209470300000000000000000
	*/
	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := proto.DecodeFromHexString(hexdata)
	require.NoError(t, err)

	//0x989393922c92e07c209f67636731bf1f04871d8b
	txData := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, data, 100000, proto.TestNetScheme)
	tx := proto.NewEthereumTransaction(txData, nil, nil, &senderPK, 0)

	db := ethabi.NewErc20MethodsMap()
	assert.NotNil(t, tx.Data())
	decodedData, err := db.ParseCallDataRide(tx.Data())
	assert.NoError(t, err)
	lessenDecodedDataAmount(t, decodedData)

	erc20arguments, err := ethabi.GetERC20TransferArguments(decodedData)
	assert.NoError(t, err)

	assetID := (*proto.AssetID)(tx.To())

	assetInfo, err := txAppender.ethInfo.stor.assets.newestAssetInfo(*assetID)
	require.NoError(t, err)
	fullAssetID := proto.ReconstructDigest(*assetID, assetInfo.tail)
	tx.TxKind = proto.NewEthereumTransferAssetsErc20TxKind(*decodedData, *proto.NewOptionalAssetFromDigest(fullAssetID), erc20arguments)
	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	assert.True(t, applRes.status)

	sender, err := senderPK.EthereumAddress().ToWavesAddress(proto.TestNetScheme)
	assert.NoError(t, err)

	wavesAsset := proto.NewOptionalAssetWaves()
	senderWavesKey := byteKey(sender.ID(), wavesAsset)
	senderKey := byteKey(sender.ID(), *proto.NewOptionalAssetFromDigest(fullAssetID))

	rideEthRecipientAddress, ok := decodedData.Inputs[0].Value.(ethabi.Bytes)
	assert.True(t, ok)
	ethRecipientAddress := proto.BytesToEthereumAddress(rideEthRecipientAddress)
	recipient, err := ethRecipientAddress.ToWavesAddress(proto.TestNetScheme)
	assert.NoError(t, err)
	recipientKey := byteKey(recipient.ID(), *proto.NewOptionalAssetFromDigest(fullAssetID))
	senderWavesBalance := applRes.changes.diff[string(senderWavesKey)].balance
	senderBalance := applRes.changes.diff[string(senderKey)].balance
	recipientBalance := applRes.changes.diff[string(recipientKey)].balance
	assert.Equal(t, senderWavesBalance, int64(-100000))
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

func applyScript(t *testing.T, tx *proto.EthereumTransaction, stor scriptStorageState) (proto.WavesAddress, *ast.Tree) {
	scriptAddr, err := tx.WavesAddressTo(0)
	require.NoError(t, err)
	tree, err := stor.newestScriptByAddr(*scriptAddr)
	require.NoError(t, err)
	return *scriptAddr, tree
}

func TestEthereumInvoke(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress) (*ast.Tree, error) {
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
		tree, err := serialization.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &mockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		newestScriptBasicInfoByAddressIDFunc: func(id proto.AddressID) (scriptBasicInfoRecord, error) {
			return scriptBasicInfoRecord{PK: crypto.MustPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")}, nil
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID) (bool, error) {
			return false, nil
		},
		newestAccountHasVerifierFunc: func(addr proto.WavesAddress) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	assetsUncertain := map[proto.AssetID]assetInfo{
		proto.AssetID(recipientEth): {},
	}
	txAppender := defaultTxAppender(t, storage, state, assetsUncertain, proto.TestNetScheme)

	txData := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000, proto.TestNetScheme)
	decodedData := defaultDecodedData("call", []ethabi.DecodedArg{{Value: ethabi.Int(10)}}, []ethabi.Payment{{Amount: 5, AssetID: proto.NewOptionalAssetWaves().ID}})
	txKind := proto.NewEthereumInvokeScriptTxKind(decodedData)
	tx := proto.NewEthereumTransaction(txData, txKind, &crypto.Digest{}, &senderPK, 0)

	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage)
	fallibleInfo.rideV5Activated = true
	res, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress)
	assert.NoError(t, err)
	assert.True(t, res.Result())

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 10}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], res.ScriptActions()[0])

	// fee test
	txDataForFeeCheck := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 499999, proto.MainNetScheme)
	tx = proto.NewEthereumTransaction(txDataForFeeCheck, txKind, &crypto.Digest{}, &senderPK, 0)

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	require.Error(t, err)
}

func TestTransferZeroAmount(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(0, &recipientEth, nil, 100000, proto.TestNetScheme)
	tx := proto.NewEthereumTransaction(txData, nil, nil, &senderPK, 0)
	tx.TxKind, err = txAppender.ethInfo.ethereumTransactionKind(&tx, nil)
	assert.NoError(t, err)

	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.EqualError(t, err, "the amount of ethereum transfer waves is 0, which is forbidden")
}

func TestTransferTestNetTestnet(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(100, &recipientEth, nil, 100000, proto.TestNetScheme)
	tx := proto.NewEthereumTransaction(txData, nil, nil, &senderPK, 0)
	tx.TxKind, err = txAppender.ethInfo.ethereumTransactionKind(&tx, nil)
	assert.NoError(t, err)

	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.EqualError(t, err, "the amount of ethereum transfer waves is 0, which is forbidden")
}

func TestTransferCheckFee(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(100, &recipientEth, nil, 100, proto.TestNetScheme)
	tx := proto.NewEthereumTransaction(txData, nil, nil, &senderPK, 0)
	tx.TxKind, err = txAppender.ethInfo.ethereumTransactionKind(&tx, nil)
	assert.NoError(t, err)

	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.EqualError(t, err, "the amount of ethereum transfer waves is 0, which is forbidden")
}

func TestEthereumInvokeWithoutPaymentsAndArguments(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress) (*ast.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			@Callable(i)
			func call() = {
			  [
			    IntegerEntry("int", 1)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2ludAAAAAAAAAAAAQUAAAADbmlsAAAAAOhqG0I=")
		require.NoError(t, err)
		tree, err := serialization.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &mockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		newestScriptBasicInfoByAddressIDFunc: func(id proto.AddressID) (scriptBasicInfoRecord, error) {
			return scriptBasicInfoRecord{PK: crypto.MustPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")}, nil
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID) (bool, error) {
			return false, nil
		},
		newestAccountHasVerifierFunc: func(addr proto.WavesAddress) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000, proto.TestNetScheme)
	decodedData := defaultDecodedData("call", nil, nil)
	tx := proto.NewEthereumTransaction(txData, proto.NewEthereumInvokeScriptTxKind(decodedData), &crypto.Digest{}, &senderPK, 0)

	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage)
	fallibleInfo.rideV5Activated = true
	res, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress)
	assert.NoError(t, err)
	assert.True(t, res.Result())

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], res.ScriptActions()[0])

}

func TestEthereumInvokeAllArguments(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress) (*ast.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			@Callable(i)
			func call(num: Int, flag: Boolean, vec:ByteVector, lis: Int|String, list: List[Int]) = {
			  [
				IntegerEntry("int", num)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAsIAhIHCgUBBAIJEQAAAAAAAAABAAAAAWkBAAAABGNhbGwAAAAFAAAAA251bQAAAARmbGFnAAAAA3ZlYwAAAANsaXMAAAAEbGlzdAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50BQAAAANudW0FAAAAA25pbAAAAAC7za7+")
		require.NoError(t, err)
		tree, err := serialization.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &mockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		newestScriptBasicInfoByAddressIDFunc: func(id proto.AddressID) (scriptBasicInfoRecord, error) {
			return scriptBasicInfoRecord{PK: crypto.MustPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")}, nil
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID) (bool, error) {
			return false, nil
		},
		newestAccountHasVerifierFunc: func(addr proto.WavesAddress) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	txData := defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000, proto.TestNetScheme)
	decodedData := defaultDecodedData("call", []ethabi.DecodedArg{
		{Value: ethabi.Int(1)},
		{Value: ethabi.Bool(true)},
		{Value: ethabi.Bytes([]byte{2})},
		{Value: ethabi.Bool(true)}, // will leave it here
		{Value: ethabi.List{ethabi.Int(4)}},
	}, nil)
	tx := proto.NewEthereumTransaction(txData, proto.NewEthereumInvokeScriptTxKind(decodedData), &crypto.Digest{}, &senderPK, 0)

	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage)
	fallibleInfo.rideV5Activated = true
	res, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress)
	assert.NoError(t, err)
	assert.True(t, res.Result())

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], res.ScriptActions()[0])
}

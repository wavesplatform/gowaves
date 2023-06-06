package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	UpdateAssetInfoMinVersion = 1
	UpdateAssetInfoMaxVersion = 1
)

type UpdateAssetInfoTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	AssetName string
	AssetDesc string
	Fee       uint64
	FeeAsset  proto.OptionalAsset
	Timestamp uint64
	ChainID   proto.Scheme
	Expected  T
}

type UpdateAssetInfoExpectedPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

type UpdateAssetInfoExpectedNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	_                 struct{}
}

func NewUpdateAssetInfoTestData[T any](account config.AccountInfo, assetID crypto.Digest, assetName, assetDesc string,
	fee, timestamp uint64, feeAssetID *crypto.Digest, chainID proto.Scheme, expected T) UpdateAssetInfoTestData[T] {
	var feeAsset proto.OptionalAsset
	if feeAssetID == nil {
		feeAsset = proto.NewOptionalAssetWaves()
	} else {
		feeAsset = *proto.NewOptionalAssetFromDigest(*feeAssetID)
	}
	return UpdateAssetInfoTestData[T]{
		Account:   account,
		AssetID:   assetID,
		AssetName: assetName,
		AssetDesc: assetDesc,
		Fee:       fee,
		FeeAsset:  feeAsset,
		Timestamp: timestamp,
		ChainID:   chainID,
		Expected:  expected,
	}
}

func GetUpdateAssetInfoPositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedPositive] {
	return map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedPositive]{
		"Min values for fee, name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(4, utl.CommonSymbolSet),
			"",
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Middle values for name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(500, utl.CommonSymbolSet),
			2*utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: 2 * utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Max values for name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(16, utl.CommonSymbolSet),
			utl.RandStringBytes(1000, utl.CommonSymbolSet),
			1000*utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: 1000 * utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
	}
}

func GetUpdateSmartAssetInfoPositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedPositive] {
	return map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedPositive]{
		"Min values for fee, name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(4, utl.CommonSymbolSet),
			"",
			utl.MinTxFeeWavesSmartAsset,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: utl.MinTxFeeWavesSmartAsset,
				AssetDiffBalance: 0,
			}),
		"Middle values for name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(500, utl.CommonSymbolSet),
			2*utl.MinTxFeeWavesSmartAsset,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: 2 * utl.MinTxFeeWavesSmartAsset,
				AssetDiffBalance: 0,
			}),
		"Max values for name and desc len": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(16, utl.CommonSymbolSet),
			utl.RandStringBytes(1000, utl.CommonSymbolSet),
			1000*utl.MinTxFeeWavesSmartAsset,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedPositive{
				WavesDiffBalance: 1000 * utl.MinTxFeeWavesSmartAsset,
				AssetDiffBalance: 0,
			}),
	}
}

func GetUpdateAssetInfoWithoutWaitingNegativeData(suite *f.BaseSuite, assetID crypto.Digest) []UpdateAssetInfoTestData[UpdateAssetInfoExpectedNegative] {
	return []UpdateAssetInfoTestData[UpdateAssetInfoExpectedNegative]{
		NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(4, utl.CommonSymbolSet),
			"",
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
	}
}

func GetUpdateAssetInfoNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedNegative] {
	return map[string]UpdateAssetInfoTestData[UpdateAssetInfoExpectedNegative]{
		"Invalid asset name (len < min)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(3, utl.CommonSymbolSet),
			utl.RandStringBytes(3, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid asset name (len > max)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(17, utl.CommonSymbolSet),
			utl.RandStringBytes(3, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid asset desc (len > max)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(8, utl.CommonSymbolSet),
			utl.RandStringBytes(1001, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid encoding in asset name": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			"\\u0069\\u006e\\u0076\\u0061\\u006c\\u0069\\u0064",
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Update info when there are not enough funds on the account balance": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (fee = 0)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			0,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (0 < fee < min)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			10,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (fee > max)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid asset ID (asset ID not exist)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			utl.TestChainID,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid chainID (value=0)": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			0,
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Custom chainID": NewUpdateAssetInfoTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.RandStringBytes(10, utl.CommonSymbolSet),
			utl.RandStringBytes(100, utl.CommonSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			nil,
			'T',
			UpdateAssetInfoExpectedNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
	}
}

package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BurnTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	Quantity  uint64
	ChainID   proto.Scheme
	Timestamp uint64
	Fee       uint64
	Expected  T
}

type BurnExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{} // this field is necessary to force using explicit struct initialization
}

type BurnExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	_                 struct{} // this field is necessary to force using explicit struct initialization
}

func NewBurnTestData[T any](account config.AccountInfo, assetID crypto.Digest, quantity uint64, chainID proto.Scheme,
	timestamp uint64, fee uint64, expected T) *BurnTestData[T] {
	return &BurnTestData[T]{
		Account:   account,
		AssetID:   assetID,
		Quantity:  quantity,
		ChainID:   chainID,
		Timestamp: timestamp,
		Fee:       fee,
		Expected:  expected,
	}
}

func GetBurnPositiveDataMatrix(suite *f.BaseSuite, assetId crypto.Digest) map[string]BurnTestData[BurnExpectedValuesPositive] {
	middleAssetValue := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId) / 2
	var t = map[string]BurnTestData[BurnExpectedValuesPositive]{
		"Burn zero amount(quantity) of asset": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			0,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Burn valid min values for amount(quantity) of asset": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			1,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 1,
			}),
		"Valid middle values for amount(quantity) of asset": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			uint64(middleAssetValue),
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: middleAssetValue,
			}),
	}
	return t
}

func GetBurnAllAssetWithMaxAvailableFee(suite *f.BaseSuite, assetId crypto.Digest, accNumber int) map[string]BurnTestData[BurnExpectedValuesPositive] {
	assetValue := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	fee := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address)
	var t = map[string]BurnTestData[BurnExpectedValuesPositive]{
		"Burn all available asset, max available fee": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			uint64(assetValue),
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			uint64(fee),
			BurnExpectedValuesPositive{
				WavesDiffBalance: fee,
				AssetDiffBalance: assetValue,
			}),
	}
	return t
}

func GetBurnNFTFromOwnerAccount(suite *f.BaseSuite, assetId crypto.Digest) map[string]BurnTestData[BurnExpectedValuesPositive] {
	var t = map[string]BurnTestData[BurnExpectedValuesPositive]{
		"Burn NFT from owner account": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			assetId,
			1,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 1,
			}),
	}
	return t
}

func GetBurnNegativeDataMatrix(suite *f.BaseSuite, assetId crypto.Digest) map[string]BurnTestData[BurnExpectedValuesNegative] {
	var t = map[string]BurnTestData[BurnExpectedValuesNegative]{
		"Burn amount > max of asset": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			9223372036854775808,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid asset ID (asset ID not exist)": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			100,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (fee = 0)": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			0,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (0 < fee < min)": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			10,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "(10 in WAVES) does not exceed minimal value of 100000 WAVES",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid fee (fee > max)": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MaxAmount+1,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Burn token when there are not enougth funds on the account balance": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Accounts balance error",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Burn WAVES": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewOptionalAssetWaves().ID,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Burn from account that not own asset": *NewBurnTestData(
			utl.GetAccount(suite, 4),
			assetId,
			10000,
			utl.TestChainID,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Accounts balance errors",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
	}
	return t
}

func GetBurnChainIDNegativeDataMatrix(suite *f.BaseSuite, assetId crypto.Digest) map[string]BurnTestData[BurnExpectedValuesNegative] {
	var t = map[string]BurnTestData[BurnExpectedValuesNegative]{
		"Custom chainID": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			10000,
			'T',
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature for",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
		"Invalid chainID (value=0)": *NewBurnTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetId,
			100000,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.MinTxFeeWaves,
			BurnExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature for",
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
			}),
	}
	return t
}

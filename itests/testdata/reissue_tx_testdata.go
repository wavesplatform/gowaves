package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ReissueTestData[T any] struct {
	Account    config.AccountInfo
	AssetID    crypto.Digest
	Fee        uint64
	Timestamp  uint64
	ChainID    proto.Scheme
	Quantity   uint64
	Reissuable bool
	Expected   T
}

type ReissueExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	Reissuable       bool
	_                struct{}
}

type ReissueExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	Reissuable        bool
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

type ReissueNotReissuableExpectedValuesNegative struct {
	Positive ReissueExpectedValuesPositive
	Negative ReissueExpectedValuesNegative
	_        struct{}
}

func NewReissueTestData[T any](account config.AccountInfo, assetID crypto.Digest, fee uint64, timestamp uint64,
	chainID proto.Scheme, quantity uint64, reissuable bool, expected T) *ReissueTestData[T] {
	return &ReissueTestData[T]{
		Account:    account,
		AssetID:    assetID,
		Fee:        fee,
		Timestamp:  timestamp,
		ChainID:    chainID,
		Quantity:   quantity,
		Reissuable: reissuable,
		Expected:   expected,
	}
}

func ReissueDataChangedTimestamp[T any](td *ReissueTestData[T]) ReissueTestData[T] {
	return *NewReissueTestData(td.Account, td.AssetID, td.Fee, utl.GetCurrentTimestampInMs(),
		td.ChainID, td.Quantity, td.Reissuable, td.Expected)
}

func GetReissuePositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesPositive] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesPositive]{
		"Min values for fee and quantity": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			1,
			true,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 1,
				Reissuable:       true,
			}),
		"Middle values for fee and quantity": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000000,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000000,
			true,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: 100000000,
				AssetDiffBalance: 10000000000,
				Reissuable:       true,
			}),
	}
	return t
}

func GetReissueMaxQuantityValue(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesPositive] {
	var assetBalance = utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetID)
	var t = map[string]ReissueTestData[ReissueExpectedValuesPositive]{
		"Max values for quantity": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			uint64(utl.MaxAmount-assetBalance),
			false,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: int64(utl.MaxAmount - assetBalance),
				Reissuable:       false,
			}),
	}
	return t
}

func GetReissueNFTData(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Reissue NFT": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			1,
			false,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        false,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Asset is not reissuable",
			}),
	}
	return t
}

func GetNotReissuableTestData(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueNotReissuableExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueNotReissuableExpectedValuesNegative]{
		"Reissue not reissuable token": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			100000,
			false,
			ReissueNotReissuableExpectedValuesNegative{
				Positive: ReissueExpectedValuesPositive{
					WavesDiffBalance: utl.MinTxFeeWaves,
					AssetDiffBalance: 100000,
				},
				Negative: ReissueExpectedValuesNegative{
					WavesDiffBalance:  0,
					AssetDiffBalance:  0,
					Reissuable:        false,
					ErrGoMsg:          errMsg,
					ErrScalaMsg:       errMsg,
					ErrBrdCstGoMsg:    errBrdCstMsg,
					ErrBrdCstScalaMsg: "Asset is not reissuable",
				},
			}),
	}
	return t
}

func GetReissueNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Invalid token quantity (quantity > max)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			utl.MaxAmount+1,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			}),
		"Invalid token quantity (quantity < min)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			0,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "non-positive amount: 0 of assets",
			}),
		"Invalid fee (fee > max)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			}),
		"Invalid fee (0 < fee < min)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "(10 in WAVES) does not exceed minimal value of 100000 WAVES",
			}),
		"Invalid fee (fee = 0)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
			}),
		"Reissue token when there are not enough funds on the account balance": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Accounts balance error",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
			}),
		"Reissue by another account": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Asset was issued by other address",
			}),
		"Invalid asset ID": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
			}),
	}
	return t
}

func GetReissueChainIDNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Custom chainID": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			'T',
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature for",
			}),
		"Invalid chainID (value=0)": *NewReissueTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			0,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature for",
			}),
	}
	return t
}

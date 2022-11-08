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

func GetReissuePositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesPositive] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesPositive]{
		"Min values for fee and quantity": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			1,
			true,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: 1,
				Reissuable:       true,
			}),
		"Middle values for fee and quantity": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000000,
			false,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: 100000000,
				AssetDiffBalance: 10000000000,
				Reissuable:       false,
			}),
	}
	return t
}

func GetReissueMaxQuantityValue(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesPositive] {
	var assetBalance = utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetID.Bytes())
	var t = map[string]ReissueTestData[ReissueExpectedValuesPositive]{
		"Max values for quantity": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			uint64(9223372036854775807-assetBalance),
			true,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: int64(9223372036854775807 - assetBalance),
				Reissuable:       true,
			}),
	}
	return t
}

func GetReissueNFTData(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Reissue NFT": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			1,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        false,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
	}
	return t
}

func GetReissueNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Invalid token quantity (quantity > max)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			9223372036854775808,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Invalid token quantity (quantity < min>)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			0,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Invalid fee (fee > max)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			9223372036854775808,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Invalid fee (0 < fee < min)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			10,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Invalid fee (fee = 0)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			0,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Reissue token when there are not enough funds on the account balance": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)),
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs()-7215000,
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs()+54160000,
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Custom chainID": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
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
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Invalid chainID (value=0)": *NewReissueTestData(
			utl.GetAccount(suite, 2),
			assetID,
			100000,
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
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Reissue by another account": *NewReissueTestData(
			utl.GetAccount(suite, 3),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				Reissuable:        true,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
	}
	return t
}

package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ReissueTestData[T any] struct {
	Version    byte
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
	WavesDiffBalance int64
	AssetDiffBalance int64
	Reissuable       bool
	ErrGoMsg         string
	ErrScalaMsg      string
	_                struct{}
}

func NewReissueTestData[T any](version byte, account config.AccountInfo, assetID crypto.Digest, fee uint64, timestamp uint64,
	chainID proto.Scheme, quantity uint64, reissuable bool, expected T) *ReissueTestData[T] {
	return &ReissueTestData[T]{
		Version:    version,
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
			1,
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
				Reissuable:       false,
			}),
		/*"Max values for quantity": *NewReissueTestData(
		1,
		utl.GetAccount(suite, 2),
		assetID,
		9223372036854775808,
		utl.GetCurrentTimestampInMs(),
		testChainID,
		9223372036854775807,
		true,
		ReissueExpectedValuesPositive{
			WavesDiffBalance: 100000,
			AssetDiffBalance: 1,
			Reissuable:       false,
		}),*/
	}
	return t
}

func GetReissueNFTData(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Reissue NFT": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			1,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       false,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
	}
	return t
}

func GetReissueNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesNegative] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesNegative]{
		"Invalid token quantity (quantity > max)": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			9223372036854775808,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Invalid token quantity (quantity < min>)": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			0,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Invalid fee (fee > max)": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			9223372036854775808,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Invalid fee (0 < fee < min)": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			10,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Invalid fee (fee = 0)": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			0,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Reissue token when there are not enough funds on the account balance": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)),
			utl.GetCurrentTimestampInMs(),
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs()-7215000,
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs()+54160000,
			testChainID,
			10000000,
			true,
			ReissueExpectedValuesNegative{
				WavesDiffBalance: 0,
				AssetDiffBalance: 0,
				Reissuable:       true,
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
			}),
	}
	return t
}

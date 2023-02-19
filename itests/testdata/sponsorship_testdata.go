package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SponsorshipTestData[T any] struct {
	Account              config.AccountInfo
	AssetID              crypto.Digest
	MinSponsoredAssetFee uint64
	Fee                  uint64
	Timestamp            uint64
	ChainID              proto.Scheme
	Expected             T
}

type SponsorshipExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

type SponsorshipExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

func NewSponsorshipTestData[T any](account config.AccountInfo, assetID crypto.Digest,
	minSponsoredAssetFee, fee, timestamp uint64, chainID proto.Scheme, expected T) *SponsorshipTestData[T] {
	return &SponsorshipTestData[T]{
		Account:              account,
		AssetID:              assetID,
		MinSponsoredAssetFee: minSponsoredAssetFee,
		Fee:                  fee,
		Timestamp:            timestamp,
		ChainID:              chainID,
		Expected:             expected,
	}
}

func GetSponsorshipPositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive] {
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive]{
		"Min values for fee and MinSponsoredAssetFee": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			1,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
		"Valid values for fee and MinSponsoredAssetFee": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
	}
	return t
}

func GetSponsorshipNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative] {
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative]{
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		"Try to do sponsorship when fee more than funds on the sender balance": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
	}
	return t
}

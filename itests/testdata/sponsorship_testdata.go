package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	SponsorshipMaxVersion       = 3
	DefaultMinSponsoredAssetFee = 100000
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
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)
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
			uint64(wavesAmount/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesPositive{
				WavesDiffBalance: wavesAmount / 4,
				AssetDiffBalance: 0,
			}),
	}
	return t
}

func GetSponsorshipMaxValuesPositive(suite *f.BaseSuite, assetID crypto.Digest, accNumber int) map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive] {
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address)
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive]{
		"Max values for fee and MinSponsoredAssetFee": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MaxAmount,
			uint64(wavesAmount),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesPositive{
				WavesDiffBalance: wavesAmount,
				AssetDiffBalance: 0,
			}),
	}
	return t
}

type SponsorshipEnabledDisabledData struct {
	Enabled  SponsorshipTestData[SponsorshipExpectedValuesPositive]
	Disabled SponsorshipTestData[SponsorshipExpectedValuesPositive]
}

func GetSponsorshipEnabledDisabledData(suite *f.BaseSuite, assetID crypto.Digest) SponsorshipEnabledDisabledData {
	return SponsorshipEnabledDisabledData{
		Enabled: *NewSponsorshipTestData(
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
		Disabled: *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			0,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesPositive{
				WavesDiffBalance: utl.MinTxFeeWaves,
				AssetDiffBalance: 0,
			}),
	}
}

func GetSponsorshipNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative] {
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative]{
		"Sponsorship off, not issuer": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultRecipientNotMiner),
			assetID,
			0,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Asset was issued by other address",
			}),
		"Invalid fee (fee > max)": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			}),
		"Invalid fee (0 < fee < min)": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Fee for IssueTransaction (10 in WAVES) does not exceed minimal value",
			}),
		"Invalid fee (fee = 0)": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			100000,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
			}),
		"MinSponsoredAssetFee > max value": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			assetID,
			utl.MaxAmount+1,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			}),
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
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
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
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
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
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Transaction application leads to negative waves balance",
			}),
		"Invalid asset ID (asset ID not exist)": *NewSponsorshipTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandDigest(suite.T(), 32, utl.LettersAndDigits),
			100000,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SponsorshipExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Referenced assetId not found",
			}),
	}
	return t
}

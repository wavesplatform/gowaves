package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	Sponsor         = utl.DefaultSenderNotMiner
	RecipientSender = utl.DefaultRecipientNotMiner
	Recipient       = utl.FirstRecipientNotMiner
)

type TransferSponsoredExpectedValuesPositive struct {
	WavesDiffBalanceSender    int64
	AssetDiffBalanceSender    int64
	FeeAssetDiffBalanceSender int64
	WavesDiffBalanceRecipient int64
	AssetDiffBalanceRecipient int64
	WavesDiffBalanceSponsor   int64
	AssetDiffBalanceSponsor   int64
	_                         struct{}
}

type TransferSponsoredExpectedValuesNegative struct {
	WavesDiffBalanceSender    int64
	AssetDiffBalanceSender    int64
	FeeAssetDiffBalanceSender int64
	WavesDiffBalanceRecipient int64
	AssetDiffBalanceRecipient int64
	WavesDiffBalanceSponsor   int64
	AssetDiffBalanceSponsor   int64
	ErrGoMsg                  string
	ErrScalaMsg               string
	ErrBrdCstGoMsg            string
	ErrBrdCstScalaMsg         string
	_                         struct{}
}

func GetTransferSponsoredPositiveData(suite *f.BaseSuite, assetId, sponsoredAssetId crypto.Digest) map[string]TransferTestData[TransferSponsoredExpectedValuesPositive] {
	sponsoredAssetDetails := utl.GetAssetInfo(suite, sponsoredAssetId)
	assetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, assetId)
	sponsoredAssetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, sponsoredAssetId)
	wavesAmountRecipientSender := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, RecipientSender).Address)
	var t = map[string]TransferTestData[TransferSponsoredExpectedValuesPositive]{
		"Transfer Assets, fee in the same Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&sponsoredAssetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(sponsoredAssetAmountRecipientSender/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    sponsoredAssetAmountRecipientSender/4 + int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				FeeAssetDiffBalanceSender: sponsoredAssetAmountRecipientSender/4 + int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: sponsoredAssetAmountRecipientSender / 4,
				WavesDiffBalanceSponsor:   100000, //=minSponsoredAssetFee
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
		"Transfer Waves, fee in the Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			nil,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(wavesAmountRecipientSender/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    wavesAmountRecipientSender / 8,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: wavesAmountRecipientSender / 8,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   100000, //=minSponsoredAssetFee
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
		"Transfer Assets, fee in the different Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(assetAmountRecipientSender/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    assetAmountRecipientSender / 4,
				FeeAssetDiffBalanceSender: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: assetAmountRecipientSender / 4,
				WavesDiffBalanceSponsor:   100000, //=minSponsoredAssetFee
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
	}
	return t
}

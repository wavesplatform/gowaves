package testdata

import (
	"fmt"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	Sponsor                     = utl.DefaultSenderNotMiner
	RecipientSender             = utl.DefaultRecipientNotMiner
	Recipient                   = utl.FirstRecipientNotMiner
	MinSponsoredAssetFeeInWaves = 100000
)

type TransferSponsoredTestData[T any] struct {
	MinSponsoredAssetFee uint64
	TransferTestData     TransferTestData[T]
	_                    struct{}
}

func NewTransferSponsoredTestData[T any](minSponsoredAssetFee uint64, sender config.AccountInfo,
	recipient proto.Recipient, assetID *crypto.Digest, feeAssetID *crypto.Digest, fee, amount, timestamp uint64,
	chainID proto.Scheme, attachment proto.Attachment, expected T) *TransferSponsoredTestData[T] {
	return &TransferSponsoredTestData[T]{
		MinSponsoredAssetFee: minSponsoredAssetFee,
		TransferTestData: *NewTransferTestData(sender, recipient, assetID, feeAssetID, fee, amount,
			timestamp, chainID, attachment, expected),
	}
}

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

func GetSponsoredTransferPositiveData(suite *f.BaseSuite, assetId, sponsoredAssetId crypto.Digest) map[string]TransferTestData[TransferSponsoredExpectedValuesPositive] {
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
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
		"Transfer Assets to oneself, fee in the same Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, RecipientSender).Address),
			&sponsoredAssetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(sponsoredAssetAmountRecipientSender/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				FeeAssetDiffBalanceSender: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
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
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
		"Transfer Waves to oneself, fee in the Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, RecipientSender).Address),
			nil,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(wavesAmountRecipientSender/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
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
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
		"Transfer Assets to oneself, fee in the different Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, RecipientSender).Address),
			&assetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(assetAmountRecipientSender/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: int64(sponsoredAssetDetails.MinSponsoredAssetFee),
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   int64(sponsoredAssetDetails.MinSponsoredAssetFee),
			},
		),
	}
	return t
}

func GetSposoredTransferBySponsorAsSender(suite *f.BaseSuite, sponsoredAssetId, assetId crypto.Digest) map[string]TransferTestData[TransferSponsoredExpectedValuesPositive] {
	sponsoredAssetDetails := utl.GetAssetInfo(suite, sponsoredAssetId)
	sponsoredAssetAmountSponsor := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, Sponsor).Address, sponsoredAssetId)
	assetAmountSponsor := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, Sponsor).Address, assetId)
	wavesAmountSponsor := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, Sponsor).Address)
	return map[string]TransferTestData[TransferSponsoredExpectedValuesPositive]{
		"Sponsor transfer Assets to himself, fee in the same Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, Sponsor),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Sponsor).Address),
			&sponsoredAssetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(sponsoredAssetAmountSponsor/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   0,
			},
		),
		"Sponsor transfer Waves to himself, fee in the Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, Sponsor),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Sponsor).Address),
			nil,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(wavesAmountSponsor/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   0,
			},
		),
		"Sponsor transfer Assets to himself, fee in the different Sponsored Asset": *NewTransferTestData(
			utl.GetAccount(suite, Sponsor),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Sponsor).Address),
			&assetId,
			&sponsoredAssetId,
			sponsoredAssetDetails.MinSponsoredAssetFee,
			uint64(assetAmountSponsor/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   0,
			},
		),
	}
}

func GetTransferSponsoredAssetsWithDifferentMinSponsoredFeeData(suite *f.BaseSuite,
	sponsoredAssetId, assetId crypto.Digest) map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesPositive] {
	assetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, assetId)
	return map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesPositive]{
		"Min values for minSponsoredAssetFee and fee": *NewTransferSponsoredTestData(
			1,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			1,
			uint64(assetAmountRecipientSender/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    assetAmountRecipientSender / 4,
				FeeAssetDiffBalanceSender: 1,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: assetAmountRecipientSender / 4,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   1,
			}),
		"Valid value for minSponsoredAssetFee": *NewTransferSponsoredTestData(
			1,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			1111222,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    assetAmountRecipientSender / 16,
				FeeAssetDiffBalanceSender: 1111222,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: assetAmountRecipientSender / 16,
				WavesDiffBalanceSponsor:   1111222 * MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   1111222,
			}),
		"Check fee in Waves, integer part ": *NewTransferSponsoredTestData(
			3,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			5,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    assetAmountRecipientSender / 16,
				FeeAssetDiffBalanceSender: 5,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: assetAmountRecipientSender / 16,
				WavesDiffBalanceSponsor:   166666,
				AssetDiffBalanceSponsor:   5,
			}),
	}
}

func GetTransferWithSponsorshipMaxAmountPositive(suite *f.BaseSuite, sponsoredAssetId,
	assetId crypto.Digest) map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesPositive] {
	assetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, assetId)
	return map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesPositive]{
		"Max value for minSponsoredFee": *NewTransferSponsoredTestData(
			utl.MaxAmount,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			utl.MaxAmount,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesPositive{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    assetAmountRecipientSender / 16,
				FeeAssetDiffBalanceSender: utl.MaxAmount,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: assetAmountRecipientSender / 16,
				WavesDiffBalanceSponsor:   MinSponsoredAssetFeeInWaves,
				AssetDiffBalanceSponsor:   utl.MaxAmount,
			}),
	}
}

func GetTransferWithSponsorshipMaxValuesDataNegative(suite *f.BaseSuite, sponsoredAssetId,
	assetId crypto.Digest) map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesNegative] {
	assetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, assetId)
	sponsoredAssetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, sponsoredAssetId)
	return map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesNegative]{
		"Fee more than funds on the sponsor balance": *NewTransferSponsoredTestData(
			1,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			utl.MaxAmount/100000,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         "negative waves balance",
			}),
		"Fee in Waves from sponsor balance that out of long range": *NewTransferSponsoredTestData(
			1,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			1+utl.MaxAmount/100000,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         "BigInteger out of long range",
			}),
		"Overflow by fee and transferred amount in Sponsored Asset": *NewTransferSponsoredTestData(
			100000,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&sponsoredAssetId,
			&sponsoredAssetId,
			100000,
			uint64(sponsoredAssetAmountRecipientSender),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         fmt.Sprintf("asset %s overflow", sponsoredAssetId),
			}),
	}
}

func GetTransferWithSponsorshipDataNegative(suite *f.BaseSuite, sponsoredAssetId,
	assetId crypto.Digest) map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesNegative] {
	assetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, assetId)
	sponsoredAssetAmountRecipientSender := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, RecipientSender).Address, sponsoredAssetId)
	return map[string]TransferSponsoredTestData[TransferSponsoredExpectedValuesNegative]{
		"Fee in Sponsored Asset more that amount of sponsored asset on sender balance": *NewTransferSponsoredTestData(
			100000,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			uint64(sponsoredAssetAmountRecipientSender+1),
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         "Transaction application leads to negative asset",
			}),
		"Invalid fee in sponsored asset (0 < fee < min)": *NewTransferSponsoredTestData(
			100000,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			10,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         fmt.Sprintf("Fee for TransferTransaction (10 in %s) does not exceed minimal value", sponsoredAssetId),
			}),
		"Invalid fee in sponsored asset (fee = 0)": *NewTransferSponsoredTestData(
			100000,
			utl.GetAccount(suite, RecipientSender),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, Recipient).Address),
			&assetId,
			&sponsoredAssetId,
			0,
			uint64(assetAmountRecipientSender/16),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferSponsoredExpectedValuesNegative{
				WavesDiffBalanceSender:    0,
				AssetDiffBalanceSender:    0,
				FeeAssetDiffBalanceSender: 0,
				WavesDiffBalanceRecipient: 0,
				AssetDiffBalanceRecipient: 0,
				WavesDiffBalanceSponsor:   0,
				AssetDiffBalanceSponsor:   0,
				ErrGoMsg:                  errMsg,
				ErrScalaMsg:               errMsg,
				ErrBrdCstGoMsg:            errBrdCstMsg,
				ErrBrdCstScalaMsg:         "insufficient fee",
			}),
	}
}

package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransferTestData[T any] struct {
	Sender     config.AccountInfo
	Recipient  proto.Recipient
	Asset      proto.OptionalAsset
	FeeAsset   proto.OptionalAsset
	Fee        uint64
	Amount     uint64
	Attachment proto.Attachment
	Timestamp  uint64
	ChainID    proto.Scheme
	Expected   T
}

type TransferExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

func NewTransferTestData[T any](sender config.AccountInfo, recipient proto.Recipient, asset proto.OptionalAsset,
	feeAsset proto.OptionalAsset, fee, amount, timestamp uint64, chainID proto.Scheme, attachment proto.Attachment,
	expected T) *TransferTestData[T] {
	return &TransferTestData[T]{
		Sender:     sender,
		Recipient:  recipient,
		Asset:      asset,
		FeeAsset:   feeAsset,
		Fee:        fee,
		Amount:     amount,
		Timestamp:  timestamp,
		ChainID:    chainID,
		Attachment: attachment,
		Expected:   expected,
	}
}

type CommonTransferData struct {
	Asset TransferTestData[TransferExpectedValuesPositive]
	NFT   TransferTestData[TransferExpectedValuesPositive]
}

func GetCommonTransferData(suite *f.BaseSuite, assetId crypto.Digest) CommonTransferData {
	commonAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId) / 4
	return CommonTransferData{
		Asset: *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			*proto.NewOptionalAssetFromDigest(assetId),
			proto.NewOptionalAssetWaves(),
			100000,
			uint64(commonAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: commonAmount,
			},
		),
		NFT: *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			*proto.NewOptionalAssetFromDigest(assetId),
			proto.NewOptionalAssetWaves(),
			100000,
			1,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: 1,
			},
		),
	}
}

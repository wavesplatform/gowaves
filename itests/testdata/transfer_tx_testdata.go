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
	WavesDiffBalanceSender    int64
	AssetDiffBalance          int64
	WavesDiffBalanceRecipient int64

	_ struct{}
}

type TransferExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

func NewTransferTestData[T any](sender config.AccountInfo, recipient proto.Recipient, assetID *crypto.Digest,
	feeAssetID *crypto.Digest, fee, amount, timestamp uint64, chainID proto.Scheme, attachment proto.Attachment,
	expected T) *TransferTestData[T] {
	var asset, feeAsset proto.OptionalAsset
	if assetID == nil {
		asset = proto.NewOptionalAssetWaves()
	} else {
		asset = *proto.NewOptionalAssetFromDigest(*assetID)
	}
	if feeAssetID == nil {
		feeAsset = proto.NewOptionalAssetWaves()
	} else {
		feeAsset = *proto.NewOptionalAssetFromDigest(*assetID)
	}
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

func GetCommonTransferData(suite *f.BaseSuite, assetId *crypto.Digest) CommonTransferData {
	commonAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, *assetId) / 4
	return CommonTransferData{
		Asset: *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			assetId,
			nil,
			100000,
			uint64(commonAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          commonAmount,
				WavesDiffBalanceRecipient: 0,
			},
		),
		NFT: *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			assetId,
			nil,
			100000,
			1,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          1,
				WavesDiffBalanceRecipient: 0,
			},
		),
	}
}

// fee in waves => feeAssetID = nil
func GetTransferPositiveData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesPositive] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId)
	//wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)
	var t = map[string]TransferTestData[TransferExpectedValuesPositive]{
		//минимальные зн-я fee,amount,attach, указан адрес получателя
		"Min values for fee, attachment and amount": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			1,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          1,
				WavesDiffBalanceRecipient: 0,
			},
		),
		//максимальные зн-я fee,amount,attach(спец.символы), указан адрес получателя (в отдельный случай)
		/*"Max values for fee, amount, attachment": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			uint64(wavesAmount),
			uint64(assetAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("2GjX8YcCcSdmYm3pVP41e1TL1t5nQrHUkCx6V9L7SC5LxQSEmcE3irKh2NtV2x57fNU5MoRML6CVyKVbatfbcNWstN3cuernPDaF4kgpn5g1DdPkfH6gge94TesYMkhdSHwVoVvXwacr"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender: wavesAmount,
				AssetDiffBalance: assetAmount,
				WavesDiffBalanceRecipient: 0,
			},
		),*/
		//валидные зн-я fee,amount,attach, указан алиас (4 байта) получателя
		"Valid values for fee, amount, attachment, alias 4 bytes": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAlias(*proto.NewAlias(TestChainID, "test")),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("6oCrsKJu7Ev52rjB72t1d3y98G5DQmvt7TYVvW7HT4vGbqgKJxJmBzA77LpC9vcW4WNQqZ2imMghaK2gkCX5J"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          assetAmount / 4,
				WavesDiffBalanceRecipient: 0,
			}),
		//валидные зн-я fee,amount,attach, указан алиас (30 байта) получателя
		"Valid values for fee, amount, attachment, alias 30 bytes": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAlias(*proto.NewAlias(TestChainID, "MaxValidAliasValueForTestingTx")),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("6oCrsKJu7Ev52rjB72t1d3y98G5DQmvt7TYVvW7HT4vGbqgKJxJmBzA77LpC9vcW4WNQqZ2imMghaK2gkCX5J"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          assetAmount / 4,
				WavesDiffBalanceRecipient: 0,
			}),
		//перевод waves(выделить в отдельный случай?)
		/*"Waves transfer": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			nil,
			nil,
			100000,
			100000,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("6oCrsKJu7Ev52rjB72t1d3y98G5DQmvt7TYVvW7HT4vGbqgKJxJmBzA77LpC9vcW4WNQqZ2imMghaK2gkCX5J"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender: 200000,
				AssetDiffBalance: 0,
				WavesDiffBalanceRecipient: 0,
			}),
		//перевод токена с аккаунта, который не является его эмитентом (выделить в отдельный случай)
		"Transfer from account that isn't issuer": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender: wavesAmount,
				AssetDiffBalance: assetAmount,
				WavesDiffBalanceRecipient: 0,
			},
		),*/
	}
	return t
}

/*func GetTransferNegativeData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		//Перевод токена, значение attachment >max, в качестве получателя указан адрес аккаунта
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, значение attachment содержит в невалидной кодировке, в качестве получателя указан адрес аккаунта
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, значение amount=0, указан адрес аккаунта (каков результат?????)
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, значение amount>max, указан адрес аккаунта
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, значение amount<0, указан адрес аккаунта
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, указан невалидный alias 3 bytes
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, указан невалидный alias 31 bytes
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, указан невалидный alias, содержащий спецсимволы
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, в качестве адреса/алиаса указана пустая строка
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//invalid time 7200000ms in the past
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Invalid time  5400000ms in the future
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, invalid chainId=256
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, invalid chainId=0
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, assetId special symbols
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
		//Перевод токена, assetId invalid encoding
		"": *NewTransferTestData(TransferExpectedValuesNegative{}),
	}
	return t
}*/

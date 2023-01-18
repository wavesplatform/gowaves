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

func GetCommonTransferData(suite *f.BaseSuite, assetId *crypto.Digest, accountNumbers ...int) CommonTransferData {
	from, to := utl.SetFromToAccounts(accountNumbers)
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, from).Address, *assetId)
	return CommonTransferData{
		Asset: *NewTransferTestData(
			utl.GetAccount(suite, from),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address),
			assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000,
				AssetDiffBalance:          assetAmount / 4,
				WavesDiffBalanceRecipient: 0,
			},
		),
		NFT: *NewTransferTestData(
			utl.GetAccount(suite, from),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address),
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

func GetTransferPositiveData(suite *f.BaseSuite, assetId crypto.Digest, alias string) map[string]TransferTestData[TransferExpectedValuesPositive] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId)
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)
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
		//валидные зн-я fee,amount,attach
		"Valid values for fee, amount, attachment, alias": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAlias(*proto.NewAlias(TestChainID, alias)),
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
		//перевод waves, attachment contains special symbols in base58
		"Waves transfer": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			nil,
			nil,
			100000,
			uint64(wavesAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("2qcsACR1T95dchPf3anZ6W2CEMyNHnwUYuFeHDQt"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    100000 + wavesAmount/4,
				AssetDiffBalance:          0,
				WavesDiffBalanceRecipient: wavesAmount / 4,
			}),
	}
	return t
}

func GetTransferMaxAmountPositive(suite *f.BaseSuite, assetId crypto.Digest, accNumber int) map[string]TransferTestData[TransferExpectedValuesPositive] {
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address)
	var t = map[string]TransferTestData[TransferExpectedValuesPositive]{
		//перевод токена с аккаунта, который не является его эмитентом
		//максимальные зн-я amount,attach, указан адрес получателя, комиссия равна балансу вэйвов на счету аккаунта
		"Max values for amount, attachment": *NewTransferTestData(
			utl.GetAccount(suite, accNumber),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			uint64(wavesAmount),
			9223372036854775807,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("2GjX8YcCcSdmYm3pVP41e1TL1t5nQrHUkCx6V9L7SC5LxQSEmcE3irKh2NtV2x57fNU5MoRML6CVyKVbatfbcNWst"+
				"N3cuernPDaF4kgpn5g1DdPkfH6gge94TesYMkhdSHwVoVvXwacr"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    wavesAmount,
				AssetDiffBalance:          9223372036854775807,
				WavesDiffBalanceRecipient: 0,
			},
		),
	}
	return t
}

func GetTransferNegativeData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId)
	invalidAssetId := utl.RandDigest(suite.T(), 32, utl.LettersAndDigits)
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		//Перевод токена, значение attachment >max, в качестве получателя указан адрес аккаунта
		"Attachment > max": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("2GjX8YcCcSdmYm3pVP41e1TL1t5nQrHUkCx6V9L7SC5LxQSEmcE3irKh2NtV2x57fNU5MoRML6CVyKVbatfbcNWst"+
				"N3cuernPDaF4kgpn5g1DdPkfH6gge94TesYMkhdSHwVoVvXwacr2Gj"),
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			},
		),
		//Перевод токена, значение attachment содержит в невалидной кодировке, в качестве получателя указан адрес аккаунта
		//TODO seems it bug in this case (utf-8 and utf-16)
		/*"Attachment in invalid encoding": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			proto.Attachment("\\xff\\xfea\\x00b\\x00c\\x00d\\x00e\\x00f\\x00g\\x00h\\x00i\\x00j\\x00k\\x00l\\x00m"+
				"\\x00n\\x00o\\x00p\\x00q\\x00r\\x00s\\x00t\\x00u\\x00v\\x00w\\x00x\\x00y\\x00z\\"),
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			},
		),*/
		//Перевод токена, значение amount=0, указан адрес аккаунта
		"Asset amount = 0": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			0,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			},
		),
		//Перевод waves, значение amount=0, указан адрес аккаунта
		"Waves amount = 0": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			nil,
			nil,
			100000,
			0,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			},
		),
		//Перевод токена, значение amount>max, указан адрес аккаунта
		"Asset amount > max": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			9223372036854775808,
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			},
		),
		//invalid time 7200000ms in the past
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs()-7260000,
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		//Invalid time  5400000ms in the future
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs()+54160000,
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		//Перевод токена, assetId special symbols
		"Invalid asset ID": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&invalidAssetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		//Перевод токена, assetId invalid encoding
		"Transfer token when there are not enough funds on the account balance": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, 2).Address)),
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			TestChainID,
			nil,
			TransferExpectedValuesNegative{
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

func GetTransferChainIDNegativeData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId)
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		//Перевод токена, invalid chainId=0
		"Invalid chainID (value=0)": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			0,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
			}),
		//Перевод токена, invalid chainId=T
		"Custom chainID": *NewTransferTestData(
			utl.GetAccount(suite, 2),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, 3).Address),
			&assetId,
			nil,
			100000,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			'T',
			nil,
			TransferExpectedValuesNegative{
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

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
	_                         struct{}
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

func TransferDataChangedTimestamp[T any](td *TransferTestData[T]) TransferTestData[T] {
	return *NewTransferTestData(td.Sender, td.Recipient, td.Asset.ToDigest(), td.FeeAsset.ToDigest(), td.Fee, td.Amount,
		utl.GetCurrentTimestampInMs(), td.ChainID, td.Attachment, td.Expected)
}

type CommonTransferData struct {
	Asset TransferTestData[TransferExpectedValuesPositive]
	NFT   TransferTestData[TransferExpectedValuesPositive]
}

func GetCommonTransferData(suite *f.BaseSuite, assetId *crypto.Digest, accountNumbers ...int) CommonTransferData {
	from, to, _ := utl.SetFromToAccounts(accountNumbers...)
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, from).Address, *assetId)
	return CommonTransferData{
		Asset: *NewTransferTestData(
			utl.GetAccount(suite, from),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address),
			assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          assetAmount / 4,
				WavesDiffBalanceRecipient: 0,
			},
		),
		NFT: *NewTransferTestData(
			utl.GetAccount(suite, from),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, to).Address),
			assetId,
			nil,
			utl.MinTxFeeWaves,
			1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          1,
				WavesDiffBalanceRecipient: 0,
			},
		),
	}
}

func GetTransferPositiveData(suite *f.BaseSuite, assetId crypto.Digest, alias string) map[string]TransferTestData[TransferExpectedValuesPositive] {
	rcpntAddress, errAddr := proto.NewRecipientFromString(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address.String())
	suite.NoError(errAddr, "Error when creating recipient from string address: ")
	rcpntAlias, errAls := proto.NewRecipientFromString("alias:" + string(utl.TestChainID) + ":" + alias)
	suite.NoError(errAls, "Error when creating recipient from string alias: ")

	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)

	var t = map[string]TransferTestData[TransferExpectedValuesPositive]{
		"Min values for fee, attachment and amount": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          1,
				WavesDiffBalanceRecipient: 0,
			},
		),
		"Valid values for fee, amount, attachment, alias": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAlias(*proto.NewAlias(utl.TestChainID, alias)),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is valid attach string (69 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          assetAmount / 8,
				WavesDiffBalanceRecipient: 0,
			}),
		"Waves transfer": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			nil,
			nil,
			utl.MinTxFeeWaves,
			uint64(wavesAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is valid attach string (69 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves + wavesAmount/8,
				AssetDiffBalance:          0,
				WavesDiffBalanceRecipient: wavesAmount / 8,
			}),
		"Transfer assets to oneself": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is valid attach string (69 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          0,
				WavesDiffBalanceRecipient: utl.MinTxFeeWaves,
			}),
		"Transfer waves to oneself": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address),
			nil,
			nil,
			utl.MinTxFeeWaves,
			uint64(wavesAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is valid attach string (69 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          0,
				WavesDiffBalanceRecipient: utl.MinTxFeeWaves,
			}),
		"Address as string": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			rcpntAddress,
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          assetAmount / 8,
				WavesDiffBalanceRecipient: 0,
			},
		),
		"Alias as string": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			rcpntAlias,
			nil,
			nil,
			utl.MinTxFeeWaves,
			uint64(wavesAmount/8),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves + wavesAmount/8,
				AssetDiffBalance:          0,
				WavesDiffBalanceRecipient: wavesAmount / 8,
			},
		),
	}
	return t
}

func GetTransferMaxAmountPositive(suite *f.BaseSuite, assetId crypto.Digest, accNumber int) map[string]TransferTestData[TransferExpectedValuesPositive] {
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address)
	var t = map[string]TransferTestData[TransferExpectedValuesPositive]{
		"Max values for amount, attachment": *NewTransferTestData(
			utl.GetAccount(suite, accNumber),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			uint64(wavesAmount),
			utl.MaxAmount,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is valid attach string (140 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{}). "+
				"This is valid attach string (140 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})"),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    wavesAmount,
				AssetDiffBalance:          utl.MaxAmount,
				WavesDiffBalanceRecipient: 0,
			},
		),
	}
	return t
}

func GetTransferNegativeData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	wavesAmount := utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		"Attachment > max": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			proto.Attachment("This is invalid attached str (141 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."+
				" This is invalid attach str (141 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "base58-encoded string length (193) exceeds maximum length of 192",
			},
		),
		"Asset amount = 0": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "non-positive amount: 0",
			},
		),
		"Waves amount = 0": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			nil,
			nil,
			utl.MinTxFeeWaves,
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "non-positive amount: 0",
			},
		),
		"Asset amount > max": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
			},
		),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
			}),
		"Transfer token when fee more than funds on the sender balance": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			uint64(100000000+utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address)),
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Transaction application leads to negative waves balance",
			}),
		"Transfer of a bigger number of tokens than there are on the sender balance": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount+1),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Transaction application leads to negative asset",
			}),
		"Transfer all waves, not enough funds for fee": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			nil,
			nil,
			utl.MinTxFeeWaves,
			uint64(wavesAmount),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Transaction application leads to negative waves balance",
			}),
	}
	return t
}

func GetTransferChainIDChangedNegativeData(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		"Invalid chainID (value=0)": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAddressWithNewSchema(
				suite, 0, utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address)),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			0,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "invalid address",
			}),
		"Custom chainID": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAddressWithNewSchema(
				suite, 'T', utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address)),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			'T',
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "invalid address",
			}),
	}
	return t
}

func GetTransferChainIDDataNegative(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesNegative] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	var t = map[string]TransferTestData[TransferExpectedValuesNegative]{
		"Invalid chainID (value=0),which ignored for v1 and v2": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			0,
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature",
			}),
		"Custom chainID, which ignored for v1 and v2": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/4),
			utl.GetCurrentTimestampInMs(),
			'T',
			nil,
			TransferExpectedValuesNegative{
				WavesDiffBalance:  0,
				AssetDiffBalance:  0,
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Proof doesn't validate as signature",
			}),
	}
	return t
}

func GetTransferChainIDDataBinaryVersions(suite *f.BaseSuite, assetId crypto.Digest) map[string]TransferTestData[TransferExpectedValuesPositive] {
	assetAmount := utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address, assetId)
	return map[string]TransferTestData[TransferExpectedValuesPositive]{
		"Invalid chainID (value=0),which ignored for v1 and v2": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/8),
			utl.GetCurrentTimestampInMs(),
			0,
			nil,
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          assetAmount / 8,
				WavesDiffBalanceRecipient: 0,
			},
		),
		"Custom chainID, which ignored for v1 and v2": *NewTransferTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			proto.NewRecipientFromAddress(utl.GetAccount(suite, utl.DefaultRecipientNotMiner).Address),
			&assetId,
			nil,
			utl.MinTxFeeWaves,
			uint64(assetAmount/8),
			utl.GetCurrentTimestampInMs(),
			'T',
			proto.Attachment("This is valid attach string (69 bytes) (~!|#$%^&*()_+=\";:/?><|\\][{})."),
			TransferExpectedValuesPositive{
				WavesDiffBalanceSender:    utl.MinTxFeeWaves,
				AssetDiffBalance:          assetAmount / 8,
				WavesDiffBalanceRecipient: 0,
			},
		),
	}
}

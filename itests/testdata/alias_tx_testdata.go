package testdata

import (
	"math/rand"

	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	AliasSymbolSet        = "abcdefghijklmnopqrstuvwxyz0123456789.-_@"
	AliasInvalidSymbolSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ~!|#$%^&*()+=;:/?><|][{}\\\""
)

type AliasTestData[T any] struct {
	Account   config.AccountInfo
	Alias     string
	Fee       uint64
	Timestamp uint64
	ChainID   proto.Scheme
	Expected  T
}

type AliasExpectedValuesPositive struct {
	ExpectedAddress  proto.WavesAddress
	WavesDiffBalance int64
	_                struct{}
}

func (a AliasExpectedValuesPositive) Positive() bool {
	return true
}

type AliasExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	_                 struct{}
}

type SameAliasExpectedValuesNegative struct {
	ErrGoMsg                     string
	ErrScalaMsg                  string
	ErrBrdCstGoMsg               string
	ErrBrdCstScalaMsg            string
	WavesDiffBalanceAfterFirstTx int64
	ExpectedAddressAfterFirstTx  proto.WavesAddress
	WavesDiffBalance             int64
	_                            struct{}
}

func (a AliasExpectedValuesNegative) Positive() bool {
	return false
}

func NewAliasTestData[T any](account config.AccountInfo, alias string, fee uint64, timestamp uint64,
	chainID proto.Scheme, expected T) *AliasTestData[T] {
	return &AliasTestData[T]{
		Account:   account,
		Alias:     alias,
		Fee:       fee,
		Timestamp: timestamp,
		ChainID:   chainID,
		Expected:  expected,
	}
}

func GetAliasPositiveDataMatrix(suite *f.BaseSuite) map[string]AliasTestData[AliasExpectedValuesPositive] {
	var t = map[string]AliasTestData[AliasExpectedValuesPositive]{
		"Valid alias 4 bytes": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(4, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesPositive{
				ExpectedAddress:  utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address,
				WavesDiffBalance: utl.MinTxFeeWaves,
			}),
		"Valid alias 15 bytes, middle values for fee": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000000000,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesPositive{
				ExpectedAddress:  utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address,
				WavesDiffBalance: 100000000000,
			}),
	}
	return t
}

func GetAliasMaxPositiveDataMatrix(suite *f.BaseSuite, accNumber int) map[string]AliasTestData[AliasExpectedValuesPositive] {
	var t = map[string]AliasTestData[AliasExpectedValuesPositive]{
		"Valid alias 30 bytes, max available fee": *NewAliasTestData(
			utl.GetAccount(suite, accNumber),
			utl.RandStringBytes(30, AliasSymbolSet),
			uint64(utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address)),
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesPositive{
				ExpectedAddress:  utl.GetAccount(suite, accNumber).Address,
				WavesDiffBalance: utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address),
			}),
	}
	return t
}

func GetAliasNegativeDataMatrix(suite *f.BaseSuite) map[string]AliasTestData[AliasExpectedValuesNegative] {
	var t = map[string]AliasTestData[AliasExpectedValuesNegative]{
		"Invalid alias, 3 bytes": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(3, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "length should be between 4 and 30",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, invalid symbols": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(4, AliasInvalidSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, 31 bytes": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(31, AliasInvalidSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "length should be between 4 and 30",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, empty string": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			"",
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "length should be between 4 and 30",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, invalid encoding": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			"\\u0061\\u0073\\u0073\\u0065",
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (fee=0)": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			0,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "insufficient fee",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (0 < fee < min)": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			10,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "(10 in WAVES) does not exceed minimal value of 100000 WAVES",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (fee > max)": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MaxAmount+1,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "failed to parse json message",
				WavesDiffBalance:  0,
			}),
		"Custom chainID": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			'T',
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "State check failed",
				WavesDiffBalance:  0,
			}),
		"Invalid chainID (value=0)": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			0,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "State check failed",
				WavesDiffBalance:  0,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()-7260000,
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 7200000ms in the past relative to previous block timestamp",
				WavesDiffBalance:  0,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()+54160000,
			utl.TestChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    errBrdCstMsg,
				ErrBrdCstScalaMsg: "is more than 5400000ms in the future relative to block timestamp",
				WavesDiffBalance:  0,
			}),
	}
	return t
}

func GetSameAliasNegativeDataMatrix(suite *f.BaseSuite) []AliasTestData[SameAliasExpectedValuesNegative] {
	var t = []AliasTestData[SameAliasExpectedValuesNegative]{
		*NewAliasTestData(
			utl.GetAccount(suite, utl.DefaultSenderNotMiner),
			utl.RandStringBytes(15, AliasSymbolSet),
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs(),
			utl.TestChainID,
			SameAliasExpectedValuesNegative{
				ErrGoMsg:                     errMsg,
				ErrScalaMsg:                  errMsg,
				ErrBrdCstGoMsg:               errBrdCstMsg,
				ErrBrdCstScalaMsg:            "is already in the state",
				WavesDiffBalanceAfterFirstTx: utl.MinTxFeeWaves,
				ExpectedAddressAfterFirstTx:  utl.GetAccount(suite, utl.DefaultSenderNotMiner).Address,
				WavesDiffBalance:             0,
			}),
	}
	return t
}

func GetSameAliasDiffAddressNegativeDataMatrix(suite *f.BaseSuite) []AliasTestData[SameAliasExpectedValuesNegative] {
	alias := utl.RandStringBytes(15, AliasSymbolSet)
	accCount := 2
	var t []AliasTestData[SameAliasExpectedValuesNegative]
	for i := 0; i < accCount; i++ {
		t = append(t, *NewAliasTestData(
			utl.GetAccount(suite, i+2),
			alias,
			utl.MinTxFeeWaves,
			utl.GetCurrentTimestampInMs()+uint64(rand.Intn(10)),
			utl.TestChainID,
			SameAliasExpectedValuesNegative{
				ErrGoMsg:                     errMsg,
				ErrScalaMsg:                  errMsg,
				ErrBrdCstGoMsg:               errBrdCstMsg,
				ErrBrdCstScalaMsg:            "Alias already claimed",
				WavesDiffBalanceAfterFirstTx: utl.MinTxFeeWaves,
				WavesDiffBalance:             0,
			}))
	}
	return t
}

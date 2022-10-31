package testdata

import (
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
	WavesDiffBalance int64
	_                struct{}
}

type AliasExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	_                 struct{}
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

func GetVersions() []byte {
	return []byte{1, 2, 3}
}

func GetAliasPositiveDataMatrix(suite *f.BaseSuite) map[string]AliasTestData[AliasExpectedValuesPositive] {
	var t = map[string]AliasTestData[AliasExpectedValuesPositive]{
		"Valid alias 4 bytes": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(4, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesPositive{
				WavesDiffBalance: 100000,
			}),
		"Valid alias 15 bytes, middle values for fee": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000000000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesPositive{
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
			testChainID,
			AliasExpectedValuesPositive{
				WavesDiffBalance: utl.GetAvailableBalanceInWavesGo(suite, utl.GetAccount(suite, accNumber).Address),
			}),
	}
	return t
}

func GetAliasNegativeDataMatrix(suite *f.BaseSuite) map[string]AliasTestData[AliasExpectedValuesNegative] {
	var t = map[string]AliasTestData[AliasExpectedValuesNegative]{
		"Invalid alias, 3 bytes": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(3, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, invalid symbols": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(4, AliasInvalidSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, 31 bytes": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(31, AliasInvalidSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, empty string": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			"",
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid alias, invalid encoding": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			"\\u0061\\u0073\\u0073\\u0065",
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (fee=0)": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			0,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (0 < fee < min)": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			10,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid fee (fee > max)": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			9223372036854775808,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Custom chainID": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			'T',
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Invalid chainID (value=0)": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			0,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Timestamp more than 7200000ms in the past relative to previous block timestamp": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs()-7215000,
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
		"Timestamp more than 5400000ms in the future relative to previous block timestamp": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(15, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs()+54160000,
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:          errMsg,
				ErrScalaMsg:       errMsg,
				ErrBrdCstGoMsg:    "",
				ErrBrdCstScalaMsg: "",
				WavesDiffBalance:  0,
			}),
	}
	return t
}

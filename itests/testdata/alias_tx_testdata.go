package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	AliasSymbolSet        = "abcdefghijklmnopqrstuvwxyz0123456789.-_@"
	AliasInvalidSymbolSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ~!|#$%^&*()+=\\\";:/?><|][{}"
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
	ErrGoMsg         string
	ErrScalaMsg      string
	WavesDiffBalance int64
	_                struct{}
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

func GetVersions(suite *f.BaseSuite) []byte {
	return []byte{1, 2, 3}
}

func GetAliasPositiveDataMatrix(suite *f.BaseSuite) map[string]AliasTestData[AliasExpectedValuesPositive] {
	var t = map[string]AliasTestData[AliasExpectedValuesPositive]{
		"Valid alias, 4 bytes": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(4, AliasSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesPositive{
				WavesDiffBalance: 100000,
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
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
				WavesDiffBalance: 0,
			}),
		"Invalid alias, invalid symbols": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(4, AliasInvalidSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
				WavesDiffBalance: 0,
			}),
		"Invalid alias, invalid symbols, 3 bytes": *NewAliasTestData(
			utl.GetAccount(suite, 2),
			utl.RandStringBytes(3, AliasInvalidSymbolSet),
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			AliasExpectedValuesNegative{
				ErrGoMsg:         errMsg,
				ErrScalaMsg:      errMsg,
				WavesDiffBalance: 0,
			}),
	}
	return t
}

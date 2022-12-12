package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BurnTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	Quantity  uint64
	ChainID   proto.Scheme
	Timestamp uint64
	Fee       uint64
	Expected  T
}

type BurnExpectedValuesPositive struct {
	WavesDiffBalance uint64
	AssetDiffBalance uint64
	_                struct{} // this field is necessary to force using explicit struct initialization
}

type BurnExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  uint64
	AssetBalance      uint64
	_                 struct{} // this field is necessary to force using explicit struct initialization
}

func NewBurnTestData[T any](account config.AccountInfo, assetID crypto.Digest, quantity uint64, chainID proto.Scheme,
	timestamp uint64, fee uint64, expected T) *BurnTestData[T] {
	return &BurnTestData[T]{
		Account:   account,
		AssetID:   assetID,
		Quantity:  quantity,
		ChainID:   chainID,
		Timestamp: timestamp,
		Fee:       fee,
		Expected:  expected,
	}
}

func GetBurnPositiveDataMatrix(suite *f.BaseSuite, assetId crypto.Digest) map[string]BurnTestData[BurnExpectedValuesPositive] {
	middleAssetValue := uint64(utl.GetAssetBalanceGo(suite, utl.GetAccount(suite, 2).Address, assetId) / 2)
	var t = map[string]BurnTestData[BurnExpectedValuesPositive]{
		"Valid min values for amount(quantity) of asset": *NewBurnTestData(
			utl.GetAccount(suite, 2),
			assetId,
			1,
			TestChainID,
			utl.GetCurrentTimestampInMs(),
			100000,
			BurnExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: 1,
			}),
		"Valid middle values for amount(quantity) of asset": *NewBurnTestData(
			utl.GetAccount(suite, 2),
			assetId,
			middleAssetValue,
			TestChainID,
			utl.GetCurrentTimestampInMs(),
			100000,
			BurnExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: middleAssetValue,
			}),
	}
	return t
}

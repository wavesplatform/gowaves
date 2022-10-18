package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ReissueTestData[T any] struct {
	Version    byte
	Account    config.AccountInfo
	AssetID    crypto.Digest
	Fee        uint64
	Timestamp  uint64
	ChainID    proto.Scheme
	Quantity   uint64
	Reissuable bool
	Expected   T
}

type ReissueExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	Reissuable       bool
	_                struct{}
}

type ReissueExpectedValuesNegative struct {
	ErrGoMsg    string
	ErrScalaMsg string
	_           struct{}
}

func NewReissueTestData[T any](version byte, account config.AccountInfo, assetID crypto.Digest, fee uint64, timestamp uint64,
	chainID proto.Scheme, quantity uint64, reissuable bool, expected T) *ReissueTestData[T] {
	return &ReissueTestData[T]{
		Version:    version,
		Account:    account,
		AssetID:    assetID,
		Fee:        fee,
		Timestamp:  timestamp,
		ChainID:    chainID,
		Quantity:   quantity,
		Reissuable: reissuable,
		Expected:   expected,
	}
}

func GetReissuePositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]ReissueTestData[ReissueExpectedValuesPositive] {
	var t = map[string]ReissueTestData[ReissueExpectedValuesPositive]{
		"Min values for fee and quantity": *NewReissueTestData(
			1,
			utl.GetAccount(suite, 2),
			assetID,
			100000,
			utl.GetCurrentTimestampInMs(),
			testChainID,
			1,
			true,
			ReissueExpectedValuesPositive{
				WavesDiffBalance: 100000,
				AssetDiffBalance: 1,
				Reissuable:       true,
			}),
	}
	return t
}

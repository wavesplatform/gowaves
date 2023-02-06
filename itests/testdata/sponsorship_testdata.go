package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type SponsorshipTestData[T any] struct {
	Account              config.AccountInfo
	AssetID              crypto.Digest
	MinSponsoredAssetFee uint64
	Fee                  uint64
	Timestamp            uint64
	Expected             T
}

type SponsorshipExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetDiffBalance int64
	_                struct{}
}

type SponsorshipExpectedValuesNegative struct {
	WavesDiffBalance  int64
	AssetDiffBalance  int64
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	_                 struct{}
}

func NewSponsorshipTestData[T any](account config.AccountInfo, assetID crypto.Digest,
	minSponsoredAssetFee, fee, timestamp uint64, expected T) *SponsorshipTestData[T] {
	return &SponsorshipTestData[T]{
		Account:              account,
		AssetID:              assetID,
		MinSponsoredAssetFee: minSponsoredAssetFee,
		Fee:                  fee,
		Timestamp:            timestamp,
		Expected:             expected,
	}
}

func GetSponsorshipPositiveDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive] {
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesPositive]{}
	return t
}

func GetSponsorshipNegativeDataMatrix(suite *f.BaseSuite, assetID crypto.Digest) map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative] {
	var t = map[string]SponsorshipTestData[SponsorshipExpectedValuesNegative]{}
	return t
}

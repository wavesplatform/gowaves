package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type RewardDistributionApiTestData[T any] struct {
	Expected T
}

type RewardInfoApiExpectedValues struct {
	Term                uint64
	NextCheck           uint64
	VotingIntervalStart uint64
	_                   struct{}
}

func NewRewardDistributionApiTestData[T any](expected T) RewardDistributionApiTestData[T] {
	return RewardDistributionApiTestData[T]{
		Expected: expected,
	}
}

func ExpectedRewardInfoAPITestData(suite *f.BaseSuite, f func(*f.BaseSuite) uint64, height proto.Height) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	period := f(suite)
	m := (height + period - 1) / period
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			Term:                period,
			NextCheck:           period * m,
			VotingIntervalStart: period * m,
		})
}

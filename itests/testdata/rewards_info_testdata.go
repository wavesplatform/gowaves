package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
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

func GetRewardInfoApiAfterPreactivated20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	period := utl.GetRewardTermAfter20Cfg(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			Term:                period,
			NextCheck:           period,
			VotingIntervalStart: period,
		})
}

func GetRewardInfoApiAfterSupported20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	period := utl.GetRewardTermAfter20Cfg(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			Term:                period,
			NextCheck:           2 * period,
			VotingIntervalStart: 2 * period,
		})
}

func GetRewardInfoApiBefore20TestData(suite *f.BaseSuite) RewardDistributionApiTestData[RewardInfoApiExpectedValues] {
	period := utl.GetRewardTermCfg(suite)
	return NewRewardDistributionApiTestData(
		RewardInfoApiExpectedValues{
			Term:                period,
			NextCheck:           period,
			VotingIntervalStart: period,
		})
}

package testdata

import (
	"math"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

const (
	BoostMultiplier = 10
)

type BoostRewardDistributionExpectedValues struct {
	MinersSumDiffBalance int64
	DaoDiffBalance       int64
	XtnDiffBalance       int64
	_                    struct{}
}

func currentRewardToInt64(suite *f.BaseSuite, height uint64) int64 {
	currentReward := utl.GetCurrentReward(suite, height)
	if currentReward > math.MaxInt64 {
		suite.T().Fatalf("reward overflow: %d cannot fit into int64", currentReward)
	}
	return int64(currentReward)
}

func GetRewardForMinersXtnDaoWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution, height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       BoostMultiplier * MaxAddressReward,
			XtnDiffBalance:       BoostMultiplier * MaxAddressReward,
		},
	)
}

func GetRewardToMinersDaoWithoutBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution, height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
		},
	)
}

func GetRewardToMinersDaoWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       BoostMultiplier * MaxAddressReward,
			XtnDiffBalance:       0,
		},
	)
}

func GetRewardToMinersXtnDaoWithoutBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
		},
	)
}

func GetRewardToMinersXtnWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       BoostMultiplier * MaxAddressReward,
		},
	)
}

func GetRewardToMinersXtnWithoutBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
		},
	)
}

func GetRewardToMinersTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := currentRewardToInt64(suite, height)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		},
	)
}

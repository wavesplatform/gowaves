package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

// Need to use AddressesForDistribution
// Need to use RewardDistributionTestData
// Need to use GetAddressesMinersDaoXtn....

const (
	BoostMultiplier = 10
	DefaultReward   = 600000000
)

type BoostRewardDistributionExpectedValues struct {
	MinersSumDiffBalance int64
	DaoDiffBalance       int64
	XtnDiffBalance       int64
	_                    struct{}
}

// miners, dao, xtn
// periods, when feature 21 and feature 23 are active

func GetBoostRewardForMinersXtnDaoDuringPeriodsTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution, height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       BoostMultiplier * MaxAddressReward,
			XtnDiffBalance:       BoostMultiplier * MaxAddressReward,
		},
	)
}

// miners, dao, xtn (or without xtn)
// periods, when feature 21 and feature 23 are not active
func GetBoostRewardToMinersDaoWithoutXtnWithoutBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution, height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
		},
	)
}

// miners, dao, without xtn
// periods, when feature 21 is not active and feature 23 is active
func GetBoostRewardToMinersDaoWithoutXTNWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       BoostMultiplier * MaxAddressReward,
			XtnDiffBalance:       0,
		},
	)
}

// miners, dao, xtn
// periods, when feature 23 is not active
func GetBoostRewardToMinersXtnDaoWithoutBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
		},
	)
}

// miners, xtn, without dao
// periods, when feature 21 and feature 23 are active
func GetBoostRewardToMinersXtnWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - BoostMultiplier*MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       BoostMultiplier * MaxAddressReward,
		},
	)
}

// miners, xtn, without dao
// periods, when feature 23 is not active
func GetBoostRewardToMinersXtnAfterBoostPeriodTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
		},
	)
}

// miners, xtn (when when feature 21 is not activa) or without xtn, without dao
// with boost (period, when feature 23 is active)
func GetBoostRewardToMinersWithBoostTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: BoostMultiplier * DefaultReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		},
	)
}

// miners, xtn (or without xtn), without dao
// periods, when feature 21 and feature 23 is not active
func GetBoostRewardToMinersAfterPeriodsTestData(suite *f.BaseSuite,
	addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: DefaultReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		},
	)
}

package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func GetRewardDistributionAfterF14Before19TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 858.
func GetRollbackBeforeF19TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	p := currentReward / 3
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*p,
			DaoDiffBalance:       p,
			XtnDiffBalance:       p,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 859.
func GetRollbackAfterF19TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	p := currentReward / 3
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - 2*p,
			DaoDiffBalance:       p,
			XtnDiffBalance:       p,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 860
func GetRollbackBeforeF20TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 861
func GetRollbackAfterF20TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase.json")
// NODE - 862.
func GetRollbackBeforeF21TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRollbackAfterF21TestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetDesiredReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

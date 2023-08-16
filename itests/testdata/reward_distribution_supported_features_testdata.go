package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

// preactivated features 14, 19 and supported 20

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_supported_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
func GetRewardIncreaseDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - 200000000 - 200000000,
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, dao, xtn, initR=600000000, increment = 1, desiredR = 600000000
// ("preactivated_14_19_supported_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815
func GetRewardUnchangedDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - 200000000 - 200000000,
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_supported_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816
func GetRewardDecreaseDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: 200000000,
			DaoDiffBalance:       int64((utl.GetInitReward(suite) + utl.GetRewardIncrement(suite) - 200000000) / 2),
			XtnDiffBalance:       int64((utl.GetInitReward(suite) + utl.GetRewardIncrement(suite) - 200000000) / 2),
		})
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_supported_20/7W_2miners_dao_increase.json")
// NODE - 817
func GetRewardIncreaseDaoSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - 200000000,
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       0,
		})
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_supported_20/6W_2miners_xtn_not_changed.json")
// NODE - 817
func GetRewardUnchangedXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - 200000000,
			DaoDiffBalance:       0,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_supported_20/5W_2miners_xtn_decrease.json")
// NODE - 818
func GetRewardDecreaseXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - int64((utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)-200000000)/2),
			DaoDiffBalance:       0,
			XtnDiffBalance:       int64((utl.GetInitReward(suite) + utl.GetRewardIncrement(suite) - 200000000) / 2),
		})
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_supported_20/5W_2miners_dao_decrease.json")
// NODE - 818
func GetRewardDecreaseDaoSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)) - int64((utl.GetInitReward(suite)+utl.GetRewardIncrement(suite)-200000000)/2),
			DaoDiffBalance:       int64((utl.GetInitReward(suite) + utl.GetRewardIncrement(suite) - 200000000) / 2),
			XtnDiffBalance:       0,
		})
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_supported_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818
func GetReward2WUnchangedDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite) + utl.GetRewardIncrement(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_supported_20/2miners_increase.json")
// NODE - 820
func GetRewardSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite) + utl.GetRewardIncrement(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_supported_20/2miners_dao_xtn_without_f9.json")
// NODE - 821
func GetRewardDaoXtnSupportedWithout19TestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite) + utl.GetRewardIncrement(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

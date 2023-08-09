package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

// 2 miners, dao and xtn, 14, 19 preactivated,20 supported, R>6
func GetRewardIncreaseDaoXtnTestDataPositive2(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
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

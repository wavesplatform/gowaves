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

type RewardDistributionRollbackData struct {
	BeforeFeature RewardDistributionTestData[RewardDistributionExpectedValues]
	AfterFeature  RewardDistributionTestData[RewardDistributionExpectedValues]
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 858.
func GetRollbackBeforeF19TestData(suite *f.BaseSuite) RewardDistributionRollbackData {
	return RewardDistributionRollbackData{
		BeforeFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
		AfterFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - 2*int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite))/3,
				DaoDiffBalance:       int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
				XtnDiffBalance:       int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
	}
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 859.
func GetRollbackAfterF19TestData(suite *f.BaseSuite) RewardDistributionRollbackData {
	return RewardDistributionRollbackData{
		AfterFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) / 3,
				DaoDiffBalance:       int64(utl.GetInitRewardCfg(suite)) / 3,
				XtnDiffBalance:       int64(utl.GetInitRewardCfg(suite)) / 3,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
	}
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 860
func GetRollbackBeforeF20TestData(suite *f.BaseSuite) RewardDistributionRollbackData {
	return RewardDistributionRollbackData{
		BeforeFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
		AfterFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - 2*MaxAddressReward,
				DaoDiffBalance:       MaxAddressReward,
				XtnDiffBalance:       MaxAddressReward,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
func GetRollbackAfterF20TestData(suite *f.BaseSuite) RewardDistributionRollbackData {
	return RewardDistributionRollbackData{
		AfterFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - 2*MaxAddressReward,
				DaoDiffBalance:       MaxAddressReward,
				XtnDiffBalance:       MaxAddressReward,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

type RewardDistributionRollbackCeaseXtnBuyBackData struct {
	BeforeFeature RewardDistributionTestData[RewardDistributionExpectedValues]
	AfterFeature  RewardDistributionCeaseXtnBuybackData
}

func GetRollbackBeforeF21TestData(suite *f.BaseSuite) RewardDistributionRollbackCeaseXtnBuyBackData {
	return RewardDistributionRollbackCeaseXtnBuyBackData{
		BeforeFeature: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - 2*MaxAddressReward,
				DaoDiffBalance:       MaxAddressReward,
				XtnDiffBalance:       MaxAddressReward,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterFeature: RewardDistributionCeaseXtnBuybackData{
			BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
				getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
				getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
				getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
				getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
				RewardDistributionExpectedValues{
					MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - 2*MaxAddressReward,
					DaoDiffBalance:       MaxAddressReward,
					XtnDiffBalance:       MaxAddressReward,
					Term:                 utl.GetRewardTermAfter20Cfg(suite),
				}),
			AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
				getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
				getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
				getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
				getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
				RewardDistributionExpectedValues{
					MinersSumDiffBalance: int64(utl.GetDesiredReward(suite, utl.GetHeight(suite))) - MaxAddressReward,
					DaoDiffBalance:       MaxAddressReward,
					XtnDiffBalance:       0,
					Term:                 utl.GetRewardTermAfter20Cfg(suite),
				}),
		},
	}
}

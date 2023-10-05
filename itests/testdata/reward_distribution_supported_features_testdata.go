package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

//----- preactivated features 14 and supported 19, 20, Features Voting Period = 1 -----

func GetRewardDistributionAfter14Before19(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
// NODE - 860
func GetRewardIncreaseDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_supported_19_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815
func GetRewardUnchangedDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816
func GetRewardDecreaseDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: MaxAddressReward,
			DaoDiffBalance:       int64((utl.GetInitRewardCfg(suite) - utl.GetRewardIncrementCfg(suite) - MaxAddressReward) / 2),
			XtnDiffBalance:       int64((utl.GetInitRewardCfg(suite) - utl.GetRewardIncrementCfg(suite) - MaxAddressReward) / 2),
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_increase.json")
// NODE - 817
func GetRewardIncreaseDaoSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_supported_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817
func GetRewardUnchangedXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818
func GetRewardDecreaseXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)-utl.GetRewardIncrementCfg(suite)) -
				int64((utl.GetInitRewardCfg(suite)-utl.GetRewardIncrementCfg(suite)-MaxAddressReward)/2),
			DaoDiffBalance: 0,
			XtnDiffBalance: int64((utl.GetInitRewardCfg(suite) - utl.GetRewardIncrementCfg(suite) - MaxAddressReward) / 2),
			Term:           utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_dao_decrease.json")
// NODE - 818
func GetRewardDecreaseDaoSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)-utl.GetRewardIncrementCfg(suite)) -
				int64((utl.GetInitRewardCfg(suite)-utl.GetRewardIncrementCfg(suite)-MaxAddressReward)/2),
			DaoDiffBalance: int64((utl.GetInitRewardCfg(suite) - utl.GetRewardIncrementCfg(suite) - MaxAddressReward) / 2),
			XtnDiffBalance: 0,
			Term:           utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_supported_19_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818
func GetReward2WUnchangedDaoXtnSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_supported_19_20/2miners_increase.json")
// NODE - 820
func GetRewardSupportedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite) + utl.GetRewardIncrementCfg(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f19.json")
// NODE - 821
func GetRewardDaoXtnSupportedWithout19TestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite) + utl.GetRewardIncrementCfg(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 858.
func GetRewardDaoXtnSupported19TestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) - 2*int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite))/3,
			DaoDiffBalance:       int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
			XtnDiffBalance:       int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 859.
func GetRewardDaoXtnPreactivated19TestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) / 3,
			DaoDiffBalance:       int64(utl.GetInitRewardCfg(suite)) / 3,
			XtnDiffBalance:       int64(utl.GetInitRewardCfg(suite)) / 3,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

//----- preactivated features 14, 19, 20 and supported 21 Features Voting Period = 1 -----

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_supported_21/7W_2miners_dao_xtn_increase.json")
// NODE - 825
func GetRewardIncreaseDaoXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
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
	}
}

// 2 miners, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_supported_21/7W_2miners_xtn_increase.json")
// NODE - 825
func GetRewardIncreaseXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			nil,
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - MaxAddressReward,
				DaoDiffBalance:       0,
				XtnDiffBalance:       MaxAddressReward,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			nil,
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: utl.GetDesiredReward(suite, utl.GetHeight(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_not_changed.json")
// NODE - 825
func GetRewardUnchangedDaoXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
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
	}
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_supported_21/5W_2miners_dao_xtn_decrease.json")
// NODE - 826
func GetRewardDecreaseDaoXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: MaxAddressReward,
				DaoDiffBalance:       int64((utl.GetInitRewardCfg(suite) - MaxAddressReward) / 2),
				XtnDiffBalance:       int64((utl.GetInitRewardCfg(suite) - MaxAddressReward) / 2),
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: utl.GetDesiredReward(suite, utl.GetHeight(suite)) - (utl.GetDesiredReward(suite, utl.GetHeight(suite))-int64(MaxAddressReward))/2,
				DaoDiffBalance:       (utl.GetDesiredReward(suite, utl.GetHeight(suite)) - int64(MaxAddressReward)) / 2,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_supported_21/5W_2miners_xtn_decrease.json")
// NODE - 826
func GetRewardDecreaseXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			nil,
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)) - int64((utl.GetInitRewardCfg(suite)-MaxAddressReward)/2),
				DaoDiffBalance:       0,
				XtnDiffBalance:       int64((utl.GetInitRewardCfg(suite) - MaxAddressReward) / 2),
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: utl.GetDesiredReward(suite, utl.GetHeight(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20_supported_21/2W_2miners_dao_xtn_not_change.json")
// NODE - 826
func GetReward2WUnchangedDaoXtnCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20_supported_21/5W_2miners_increase.json")
// NODE - 829
func GetReward5W2MinersIncreaseCeaseXTNBuybackSupportedTestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			nil,
			nil,
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			nil,
			nil,
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: utl.GetDesiredReward(suite, utl.GetHeight(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermAfter20Cfg(suite),
			}),
	}
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830
func GetRewardDaoXtnSupportedWithout20TestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
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
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: 2 * int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
				DaoDiffBalance:       int64(utl.GetInitRewardCfg(suite)+utl.GetRewardIncrementCfg(suite)) / 3,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
	}
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_19_20.json")
// NODE - 830
func GetRewardDaoXtnSupportedWithout19And20TestData(suite *f.BaseSuite) RewardDistributionCeaseXtnBuybackData {
	return RewardDistributionCeaseXtnBuybackData{
		BeforeXtnBuyBackPeriod: NewRewardDistributionTestData(
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
		AfterXtnBuyBackPeriod: NewRewardDistributionTestData(
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
			getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
			getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
			getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
			RewardDistributionExpectedValues{
				MinersSumDiffBalance: int64(utl.GetInitRewardCfg(suite) + utl.GetRewardIncrementCfg(suite)),
				DaoDiffBalance:       0,
				XtnDiffBalance:       0,
				Term:                 utl.GetRewardTermCfg(suite),
			}),
	}
}

package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

const (
	MaxAddressReward = 200000000
)

type RewardDistributionTestData[T any] struct {
	Miner1Account     *config.AccountInfo
	Miner2Account     *config.AccountInfo
	DaoAccount        *config.AccountInfo
	XtnBuyBackAccount *config.AccountInfo
	Expected          T
}

type RewardDistributionExpectedValues struct {
	MinersSumDiffBalance int64
	DaoDiffBalance       int64
	XtnDiffBalance       int64
	Term                 uint64
	_                    struct{}
}

func getAccountPtr(account config.AccountInfo) *config.AccountInfo {
	return &account
}

func NewRewardDistributionTestData[T any](miner1Account, miner2Account, daoAccount, xtnAccount *config.AccountInfo, expected T) RewardDistributionTestData[T] {
	return RewardDistributionTestData[T]{
		Miner1Account:     miner1Account,
		Miner2Account:     miner2Account,
		DaoAccount:        daoAccount,
		XtnBuyBackAccount: xtnAccount,
		Expected:          expected,
	}
}

// preactivated features 14, 19, 20, FeaturesVotingPeriod = 1

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
func GetRewardIncreaseDaoXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, dao, xtn, initR=600000000, increment = 1, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815
func GetRewardUnchangedDaoXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816
func GetRewardDecreaseDaoXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: MaxAddressReward,
			DaoDiffBalance:       int64((utl.GetInitReward(suite) - MaxAddressReward) / 2),
			XtnDiffBalance:       int64((utl.GetInitReward(suite) - MaxAddressReward) / 2),
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_increase.json")
// NODE - 817
func GetRewardIncreaseDaoPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817
func GetRewardUnchangedXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818
func GetRewardDecreaseXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - int64((utl.GetInitReward(suite)-MaxAddressReward)/2),
			DaoDiffBalance:       0,
			XtnDiffBalance:       int64((utl.GetInitReward(suite) - MaxAddressReward) / 2),
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_decrease.json")
// NODE - 818
func GetRewardDecreaseDaoPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - int64((utl.GetInitReward(suite)-MaxAddressReward)/2),
			DaoDiffBalance:       int64((utl.GetInitReward(suite) - MaxAddressReward) / 2),
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818
func GetReward2WUnchangedDaoXtnPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20/2miners_increase.json")
// NODE - 820
func GetRewardPreactivatedTestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f9.json")
// NODE - 821
func GetRewardDaoXtnPreactivatedWithout19TestData(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20(suite),
		})
}

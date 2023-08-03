package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

type RewardDistributionTestData[T any] struct {
	Miner1Account     *config.AccountInfo
	Miner2Account     *config.AccountInfo
	DaoAccount        *config.AccountInfo
	XtnBuyBackAccount *config.AccountInfo
	Expected          T
}

type RewardDistributionExpectedValuesPositive struct {
	MinersSumDiffBalance int64
	DaoDiffBalance       int64
	XtnDiffBalance       int64
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

// 2 miners, dao and xtn, R>6
func GetRewardIncreaseDaoXtnTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
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

// 2 miners, dao and xtn, R=6, reward isn't change
func GetRewardUnchangedTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: 200000000, //200000000
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, dao and xtn, R<6
func GetRewardDecreaseDaoXtnTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: 200000000,
			DaoDiffBalance:       int64((utl.GetInitReward(suite) - 200000000) / 2),
			XtnDiffBalance:       int64((utl.GetInitReward(suite) - 200000000) / 2),
		})
}

// 2 miners, dao or xtn, R>6
func GetRewardIncreaseDaoTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - 200000000,
			DaoDiffBalance:       200000000,
			XtnDiffBalance:       0,
		})
}

// 2 miners, dao or xtn, R=6
func GetRewardUnchangedXtnTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)) - 200000000,
			DaoDiffBalance:       0,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, dao or xtn, R<6
func GetRewardDecreaseXtnTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: 200000000,
			DaoDiffBalance:       0,
			XtnDiffBalance:       200000000,
		})
}

// 2 miners, R
func GetRewardTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

// 2 miners, xtn, dao, R = 2
func GetRewardR2TestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: 200000000,
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

// 2 miners, dao, xtn, 19 not activated
func GetRewardF19NotActivateTestDataPositive(suite *f.BaseSuite) RewardDistributionTestData[RewardDistributionExpectedValuesPositive] {
	return NewRewardDistributionTestData(
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
		RewardDistributionExpectedValuesPositive{
			MinersSumDiffBalance: int64(utl.GetInitReward(suite)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
		})
}

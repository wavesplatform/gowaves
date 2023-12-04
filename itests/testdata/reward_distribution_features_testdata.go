package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

const (
	MaxAddressReward = 200000000
)

type AddressesForDistribution struct {
	MinerGoAccount    *config.AccountInfo
	MinerScalaAccount *config.AccountInfo
	DaoAccount        *config.AccountInfo
	XtnBuyBackAccount *config.AccountInfo
}

type RewardDistributionTestData[T any] struct {
	Addresses AddressesForDistribution
	Expected  T
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

func NewRewardDistributionTestData[T any](addresses AddressesForDistribution, expected T) RewardDistributionTestData[T] {
	return RewardDistributionTestData[T]{
		Addresses: addresses,
		Expected:  expected,
	}
}

func GetAddressesMinersDaoXtn(suite *f.BaseSuite) AddressesForDistribution {
	return AddressesForDistribution{
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
	}
}

func GetAddressesMinersDao(suite *f.BaseSuite) AddressesForDistribution {
	return AddressesForDistribution{
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		getAccountPtr(utl.GetAccount(suite, utl.DAOAccount)),
		nil,
	}
}

func GetAddressesMinersXtn(suite *f.BaseSuite) AddressesForDistribution {
	return AddressesForDistribution{
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		getAccountPtr(utl.GetAccount(suite, utl.XTNBuyBackAccount)),
	}
}

func GetAddressesMiners(suite *f.BaseSuite) AddressesForDistribution {
	return AddressesForDistribution{
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerGo)),
		getAccountPtr(utl.GetAccount(suite, utl.DefaultMinerScala)),
		nil,
		nil,
	}
}

//----- preactivated features 14, 19, 20, FeaturesVotingPeriod = 1 -----
//----- preactivated features 14 and supported 19, 20, Features Voting Period = 1 -----

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
func GetRewardIncreaseDaoXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_dao_xtn_not_changed.json")
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
func GetRewardUnchangedDaoXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_xtn_decrease.json")
// ("preactivated_14_supported_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816
func GetRewardDecreaseDaoXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: MaxAddressReward,
			DaoDiffBalance:       (currentReward - MaxAddressReward) / 2,
			XtnDiffBalance:       (currentReward - MaxAddressReward) / 2,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_increase.json")
// ("preactivated_14_supported_19_20/7W_2miners_dao_increase.json")
// NODE - 817
func GetRewardIncreaseDaoTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_xtn_not_changed.json")
// ("preactivated_14_supported_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817
func GetRewardUnchangedXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_xtn_decrease.json")
// ("preactivated_14_supported_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818
func GetRewardDecreaseXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - (currentReward-MaxAddressReward)/2,
			DaoDiffBalance:       0,
			XtnDiffBalance:       (currentReward - MaxAddressReward) / 2,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_decrease.json")
// ("preactivated_14_supported_19_20/5W_2miners_dao_decrease.json")
// NODE - 818
func GetRewardDecreaseDaoTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - (currentReward-MaxAddressReward)/2,
			DaoDiffBalance:       (currentReward - MaxAddressReward) / 2,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20/2W_2miners_dao_xtn_not_changed.json")
// ("preactivated_14_supported_19_20/2miners_increase.json")
// NODE - 818
func GetReward2WUnchangedDaoXtnTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20/2miners_increase.json")
// ("preactivated_14_supported_19_20/2miners_increase.json")
// NODE - 820
func GetRewardMinersTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f19.json")
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f19.json")
// NODE - 821
func GetRewardDaoXtnWithout19TestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

//----- preactivated features 14, 19, 20, 21 FeaturesVotingPeriod = 1 -----
//----- preactivated features 14, 19, 20 and supported 21 Features Voting Period = 1 -----

type RewardDistributionCeaseXtnBuybackData struct {
	BeforeXtnBuyBackPeriod RewardDistributionTestData[RewardDistributionExpectedValues]
	AfterXtnBuyBackPeriod  RewardDistributionTestData[RewardDistributionExpectedValues]
}

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_21/7W_2miners_dao_xtn_increase.json")
// ("preactivated_14_19_20_supported_21/7W_2miners_dao_xtn_increase.json")
// NODE - 825
func GetRewardIncreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRewardIncreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_21/7W_2miners_xtn_increase.json")
// ("preactivated_14_19_20_supported_21/7W_2miners_xtn_increase.json")
// NODE - 825
func GetRewardIncreaseXtnCeaseXTNBuybackBeforePeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       0,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRewardIncreaseXtnCeaseXTNBuybackAfterPeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_not_changed.json")
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_not_changed.json")
// NODE - 825
func GetRewardUnchangedDaoXtnCeaseXTNBuybackBeforePeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - 2*MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       MaxAddressReward,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRewardUnchangedDaoXtnCeaseXTNBuybackAfterPeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)) - MaxAddressReward,
			DaoDiffBalance:       MaxAddressReward,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_21/5W_2miners_dao_xtn_decrease.json")
// ("preactivated_14_19_20_supported_21/5W_2miners_dao_xtn_decrease.json")
// NODE - 826
func GetRewardDecreaseDaoXtnCeaseXTNBuybackBeforePeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: MaxAddressReward,
			DaoDiffBalance:       (currentReward - MaxAddressReward) / 2,
			XtnDiffBalance:       (currentReward - MaxAddressReward) / 2,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRewardDecreaseDaoXtnCeaseXTNBuybackAfterPeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - (currentReward-MaxAddressReward)/2,
			DaoDiffBalance:       (currentReward - MaxAddressReward) / 2,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_21/5W_2miners_xtn_decrease.json")
// ("preactivated_14_19_20_supported_21/5W_2miners_xtn_decrease.json")
// NODE - 826
func GetRewardDecreaseXtnCeaseXTNBuybackBeforePeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - (currentReward-MaxAddressReward)/2,
			DaoDiffBalance:       0,
			XtnDiffBalance:       (currentReward - MaxAddressReward) / 2,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

func GetRewardDecreaseXtnCeaseXTNBuybackAfterPeriodTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20_21/2W_2miners_dao_xtn_not_change.json")
// ("preactivated_14_19_20_supported_21/2W_2miners_dao_xtn_not_change.json")
// NODE - 826
func GetReward2WUnchangedDaoXtnCeaseXTNBuybackTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20_21/5W_2miners_increase.json")
// ("preactivated_14_19_20_supported_21/5W_2miners_increase.json")
// NODE - 829
func GetReward5W2MinersIncreaseCeaseXTNBuybackTestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermAfter20Cfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_increase_without_20.json")
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830
func GetRewardDaoXtnBeforePeriodWithout20TestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward / 3,
			DaoDiffBalance:       currentReward / 3,
			XtnDiffBalance:       currentReward / 3,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

func GetRewardDaoXtnAfterPeriodWithout20TestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	currentReward := int64(utl.GetCurrentReward(suite, height))
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: currentReward - currentReward/3,
			DaoDiffBalance:       currentReward / 3,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_increase_without_19_20.json")
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_19_20.json")
// NODE - 830
func GetRewardDaoXtnWithout19And20TestData(suite *f.BaseSuite, addresses AddressesForDistribution,
	height uint64) RewardDistributionTestData[RewardDistributionExpectedValues] {
	return NewRewardDistributionTestData(
		addresses,
		RewardDistributionExpectedValues{
			MinersSumDiffBalance: int64(utl.GetCurrentReward(suite, height)),
			DaoDiffBalance:       0,
			XtnDiffBalance:       0,
			Term:                 utl.GetRewardTermCfg(suite),
		})
}

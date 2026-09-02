package testdata

import (
	f "github.com/wavesplatform/gowaves/itests/fixtures"
)

// Block reward distribution parameters of feature 26 "Adjusted Block Reward Distribution".
const (
	AdjustedFullReward            = 2000000000
	AdjustedDaoReward             = 1000000000
	AdjustedXtnBuybackReward      = 200000000
	AdjustedGuaranteedMinerReward = AdjustedFullReward - AdjustedDaoReward - AdjustedXtnBuybackReward

	// The DAO address receives 5/6 and the XTN buy-back address receives 1/6 of the block reward left
	// after the guaranteed miner reward.
	adjustedDaoRemainderDividend        = 5
	adjustedXtnBuybackRemainderDividend = 1
	adjustedRemainderDivider            = 6
)

// adjustedRewardShares splits the given block reward the way feature 26 prescribes and returns the
// shares of the miner, the DAO address and the XTN buy-back address.
//
// The block reward stays votable after the activation of feature 26, so the shares have to be derived
// from the reward the blockchain currently has and not from the full reward of 20 WAVES.
// Below the guaranteed miner reward of 8 WAVES the whole reward goes to the miner. Below the full
// reward of 20 WAVES the miner keeps its guaranteed reward and the addresses share the remainder as
// 5/6 and 1/6 of it, the remainder is divided first and multiplied after. At or above the full reward
// the addresses receive their maximum shares of 10 and 2 WAVES. Everything left, including the
// truncated part of the remainder and the shares of the addresses that are not taken into account,
// goes to the miner.
func adjustedRewardShares(reward int64, daoTaken, xtnBuybackTaken bool) (int64, int64, int64) {
	var dao, xtn int64
	switch {
	case reward < AdjustedGuaranteedMinerReward: // the whole reward goes to the miner
	case reward < AdjustedFullReward:
		remainder := reward - AdjustedGuaranteedMinerReward
		dao = remainder / adjustedRemainderDivider * adjustedDaoRemainderDividend
		xtn = remainder / adjustedRemainderDivider * adjustedXtnBuybackRemainderDividend
	default:
		dao, xtn = AdjustedDaoReward, AdjustedXtnBuybackReward
	}
	if !daoTaken {
		dao = 0
	}
	if !xtnBuybackTaken {
		xtn = 0
	}
	return reward - dao - xtn, dao, xtn
}

func adjustedRewardDistributionTestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64, daoTaken, xtnBuybackTaken bool,
) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	miner, dao, xtn := adjustedRewardShares(currentRewardToInt64(suite, height), daoTaken, xtnBuybackTaken)
	return NewRewardDistributionTestData(
		addresses,
		BoostRewardDistributionExpectedValues{
			MinersSumDiffBalance: miner,
			DaoDiffBalance:       dao,
			XtnDiffBalance:       xtn,
		},
	)
}

// GetRewardAdjustedDistributionDaoXtnTestData returns the expected reward distribution with both reward
// addresses configured. At the full reward of 20 WAVES it is 8 WAVES to the miner, 10 WAVES to the DAO
// address and 2 WAVES to the XTN buy-back address.
func GetRewardAdjustedDistributionDaoXtnTestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	return adjustedRewardDistributionTestData(suite, addresses, height, true, true)
}

// GetRewardAdjustedDistributionDaoTestData returns the expected reward distribution when only the DAO
// address is configured or the XTN buy-back is ceased: the share of the XTN buy-back address goes to
// the miner.
func GetRewardAdjustedDistributionDaoTestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	return adjustedRewardDistributionTestData(suite, addresses, height, true, false)
}

// GetRewardAdjustedDistributionXtnTestData returns the expected reward distribution when only the XTN
// buy-back address is configured: the share of the DAO address goes to the miner.
func GetRewardAdjustedDistributionXtnTestData(
	suite *f.BaseSuite, addresses AddressesForDistribution, height uint64,
) RewardDistributionTestData[BoostRewardDistributionExpectedValues] {
	return adjustedRewardDistributionTestData(suite, addresses, height, false, true)
}

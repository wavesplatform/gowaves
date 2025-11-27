package itests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// Preactivated features 14, 19, 20, 21, 22, 23,
// 2 miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 800000000,
// min_xtn_buy_back_period = block_reward_boost_period.
type RewardBoostDaoXtnEqualPeriodsPreactivatedAllTestSuite struct {
	f.RewardBoostDaoXtnEqualPeriodsPreactivatedAllSuite
}

func (s *RewardBoostDaoXtnEqualPeriodsPreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostDaoXtnEqualPeriodsPreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostDaoXtnEqualPeriodsPreactivatedAllTestSuite) Test_BoostBlockRewardPreactivatedFeaturesEqPeriods() {
	const name = "Activation periods for feature 21 and feature 23 are equal"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, xtn, dao balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardForMinersXtnDaoWithBoostTestData)
		// Get current height and calculate heights where features periods end
		// This heights should be equal each other.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		assert.Equal(s.T(), ceaseXtnBuybackHeight, boostHeight)
		// Wait for these heights.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn, dao balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithoutBoostTestData)
	})
}

func TestBoostBlockRewardTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostDaoXtnEqualPeriodsPreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// 2 miners, dao, xtn, initR = 900000000, increment = 100000000, desiredR = 700000000,
// min_xtn_buy_back_period < block_reward_boost_period.
type RewardBoostDaoXtnP21LessP23PreactivatedAllTestSuite struct {
	f.RewardBoostDaoXtnP21LessP23PreactivatedAllSuite
}

func (s *RewardBoostDaoXtnP21LessP23PreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostDaoXtnP21LessP23PreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostDaoXtnP21LessP23PreactivatedAllTestSuite) Test_BoostBlockRewardPreactivatedFeaturesP21LessP23() {
	const name = "Activation period for feature 21 is less than activation period for feature 23"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, xtn, dao balances
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardForMinersXtnDaoWithBoostTestData)
		// Get current height and calculate heights where features periods end
		// ceaseXtnBuybackHeight should be less than boostHeight.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		assert.Less(s.T(), ceaseXtnBuybackHeight, boostHeight)
		// Wait for ceaseXtnBuybackHeight height.
		utl.WaitForHeight(&s.BaseSuite, ceaseXtnBuybackHeight,
			config.WaitWithTimeoutInBlocks(utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn, dao balances after feature 21 period ends.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithBoostTestData)
		// Wait for boost height.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn, dao balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithoutBoostTestData)
	})
}

func TestRewardBoostDaoXtnP21LessP23PreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostDaoXtnP21LessP23PreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// 2 miners, dao, xtn, initR = 900000000, increment = 100000000, desiredR = 700000000,
// min_xtn_buy_back_period > block_reward_boost_period.
type RewardBoostDaoXtnP21GreaterP23PreactivatedAllTestSuite struct {
	f.RewardBoostDaoXtnP21GreaterP23PreactivatedAllSuite
}

func (s *RewardBoostDaoXtnP21GreaterP23PreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostDaoXtnP21GreaterP23PreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostDaoXtnP21GreaterP23PreactivatedAllTestSuite) Test_BoostBlockRewardPreactivatedAllP21GreaterP23() {
	const name = "Activation period for feature 21 is greater than activation period for feature 23"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, xtn, dao balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardForMinersXtnDaoWithBoostTestData)
		// Get current height and calculate heights where features periods end:
		// ceaseXtnBuybackHeight should be greater than boostHeight.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		assert.Greater(s.T(), ceaseXtnBuybackHeight, boostHeight)
		// Wait for boostHeight height.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn, dao balances after feature 23 period ends.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersXtnDaoWithoutBoostTestData)
		// Wait for ceaseXtnBuybackHeight.
		utl.WaitForHeight(&s.BaseSuite, ceaseXtnBuybackHeight,
			config.WaitWithTimeoutInBlocks(utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn, dao balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithoutBoostTestData)
	})
}

func TestRewardBoostDaoXtnP21GreaterP23PreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostDaoXtnP21GreaterP23PreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// 1 miner, dao, initR = 600000000, increment = 100000000, desiredR = 600000000.
type RewardBoostMinerDaoPreactivatedAllTestSuite struct {
	f.RewardBoostMinerDaoPreactivatedAllSuite
}

func (s *RewardBoostMinerDaoPreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostMinerDaoPreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostMinerDaoPreactivatedAllTestSuite) Test_BoostBlockRewardMinerDaoPreactivatedAll() {
	const name = "All features are preactivated, one miner, dao accounts are used"
	addresses := testdata.GetAddressesMinersDao(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, dao balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithBoostTestData)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		// Wait for boostHeight height.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, dao balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithoutBoostTestData)
	})
}

func TestRewardBoostMinerDaoPreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostMinerDaoPreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// 1 miner, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000,
// min_xtn_buy_back_period < block_reward_boost_period.
type RewardBoostMinerXtnP21LessP23PreactivatedAllTestSuite struct {
	f.RewardBoostMinerXtnP21LessP23PreactivatedAllSuite
}

func (s *RewardBoostMinerXtnP21LessP23PreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostMinerXtnP21LessP23PreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostMinerXtnP21LessP23PreactivatedAllTestSuite) Test_BoostBlockRewardMinerXtnPreactivatedAll() {
	const name = "All features are preactivated, one miner, xtn accounts are used, " +
		"min_xtn_buy_back_period < block_reward_boost_period"
	addresses := testdata.GetAddressesMinersXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, xtn balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersXtnWithBoostTestData)
		// Get current height and calculate heights where features periods end
		// ceaseXtnBuybackHeight should be less than boostHeight.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		assert.Less(s.T(), ceaseXtnBuybackHeight, boostHeight)
		// Wait for ceaseXtnBuybackHeight.
		utl.WaitForHeight(&s.BaseSuite, ceaseXtnBuybackHeight,
			config.WaitWithTimeoutInBlocks(utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)))
		// Check miner's xtn balances after feature 21 period end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersWithBoostTestData)
		// Wait for boostHeight.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersWithoutBoostTestData)
	})
}

func TestRewardBoostMinerXtnP21LessP23PreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostMinerXtnP21LessP23PreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// 1 miner, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000,
// min_xtn_buy_back_period < block_reward_boost_period.
type RewardBoostMinerXtnP21GreaterP23PreactivatedAllTestSuite struct {
	f.RewardBoostMinerXtnP21GreaterP23PreactivatedAllSuite
}

func (s *RewardBoostMinerXtnP21GreaterP23PreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostMinerXtnP21GreaterP23PreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostMinerXtnP21GreaterP23PreactivatedAllTestSuite) Test_BoostBlockRewardMinerXtnPreactivatedAll() {
	const name = "All features are preactivated, one miner, xtn accounts are used, " +
		"min_xtn_buy_back_period > block_reward_boost_period"
	addresses := testdata.GetAddressesMinersXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners, xtn balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersXtnWithBoostTestData)
		// Get current height and calculate heights where features periods end
		// ceaseXtnBuybackHeight should be less than boostHeight.
		ceaseXtnBuybackHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BlockRewardDistribution, utl.GetHeight(&s.BaseSuite)) +
			utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		assert.Greater(s.T(), ceaseXtnBuybackHeight, boostHeight)
		// Wait for boostHeight.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's xtn balances after feature 23 period end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersXtnWithoutBoostTestData)
		// Wait for ceaseXtnBuybackHeight.
		utl.WaitForHeight(&s.BaseSuite, ceaseXtnBuybackHeight,
			config.WaitWithTimeoutInBlocks(utl.GetXtnBuybackPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn balances after features periods end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersWithoutBoostTestData)
	})
}

func TestRewardBoostMinerXtnP21GreaterP23PreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostMinerXtnP21GreaterP23PreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22, 23,
// miners, initR = 600000000, increment = 100000000, desiredR = 600000000.
type RewardBoostMinersPreactivatedAllTestSuite struct {
	f.RewardBoostMinersPreactivatedAllSuite
}

func (s *RewardBoostMinersPreactivatedAllTestSuite) SetupSuite() {
	s.RewardBoostMinersPreactivatedAllSuite.SetupSuite()
}

func (s *RewardBoostMinersPreactivatedAllTestSuite) Test_BoostMinersPreactivatedAll() {
	const name = "All features are preactivated, miners accounts are used"
	addresses := testdata.GetAddressesMiners(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode,
			settings.BoostBlockReward)
		// Check miners balances.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersWithBoostTestData)
		// Get current height and calculate heights where features periods end
		// ceaseXtnBuybackHeight should be less than boostHeight.
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		// Wait for boostHeight.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miners balances after feature 23 period end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersWithoutBoostTestData)
	})
}

func TestRewardBoostMinersPreactivatedAllTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostMinersPreactivatedAllTestSuite))
}

// Preactivated features 14, 19, 20, 21, 22 supported feature 23,
// miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000,
// xtn buyback period ends when start boost period.
type RewardBoostMinerXtnDaoSupportedF23TestSuite struct {
	f.RewardBoostMinerXtnDaoSupportedF23Suite
}

func (s *RewardBoostMinerXtnDaoSupportedF23TestSuite) SetupSuite() {
	s.RewardBoostMinerXtnDaoSupportedF23Suite.SetupSuite()
}

func (s *RewardBoostMinerXtnDaoSupportedF23TestSuite) Test_BoostBlockRewardMinerXtnSupported23() {
	const name = "Feature 23 is supported, miners, dao and xtn account were used, ceaseXtnBuyback period ends at " +
		"the same time when feature 23 is activated and boost period starts"
	addresses := testdata.GetAddressesMinersDaoXtn(&s.BaseSuite)
	s.Run(name, func() {
		// Check activation of features 14-22.
		utl.GetActivationOfFeatures(&s.BaseSuite,
			settings.BlockReward,
			settings.BlockRewardDistribution,
			settings.CappedRewards,
			settings.XTNBuyBackCessation,
			settings.LightNode)
		// Check miners, xtn, dao balances without boost when feature 21 is active.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersXtnDaoWithoutBoostTestData)
		// Check activation of feature 23.
		utl.GetActivationOfFeatures(&s.BaseSuite, settings.BoostBlockReward)
		// Check miners, dao and xtn balances with boost.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithBoostTestData)
		// Get current height and calculate heights where features periods end.
		boostHeight := utl.GetFeatureActivationHeight(
			&s.BaseSuite, settings.BoostBlockReward, utl.GetHeight(&s.BaseSuite)) +
			utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)
		// Wait for boostHeight.
		utl.WaitForHeight(&s.BaseSuite, boostHeight,
			config.WaitWithTimeoutInBlocks(utl.GetBoostBlockRewordPeriodCfg(&s.BaseSuite)))
		// Check miner's, xtn and dao balances after feature 23 period end.
		reward.GetRewardDistributionAndChecksWithoutTerm(
			&s.BaseSuite, addresses, testdata.GetRewardToMinersDaoWithoutBoostTestData)
	})
}

func TestRewardBoostMinerXtnDaoSupportedF23TestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardBoostMinerXtnDaoSupportedF23TestSuite))
}

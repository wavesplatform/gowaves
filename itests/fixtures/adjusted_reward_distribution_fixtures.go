package fixtures

import (
	"github.com/wavesplatform/gowaves/itests/config"
)

const (
	adjustedRewardSettingsFolder = "adjusted_reward_settings"
)

// AdjustedRewardDaoXtnPreactivatedSuite uses preactivated features 14, 19, 20, 26,
// 2 miners, dao, xtn, initR = 2000000000, increment = 100000000, desiredR = 2000000000.
type AdjustedRewardDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardDaoXtnPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_26_dao_xtn.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardDaoXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardDaoPreactivatedSuite uses preactivated features 14, 19, 20, 26,
// 2 miners, dao only, initR = 2000000000, increment = 100000000, desiredR = 2000000000.
type AdjustedRewardDaoPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardDaoPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_26_dao.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardDaoPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardXtnPreactivatedSuite uses preactivated features 14, 19, 20, 26,
// 2 miners, xtn only, initR = 2000000000, increment = 100000000, desiredR = 2000000000.
type AdjustedRewardXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardXtnPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_26_xtn.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardBelowFullRewardPreactivatedSuite uses preactivated features 14, 19, 20, 26,
// 2 miners, dao, xtn, initR = 1400000000, increment = 100000000, desiredR = 1400000000.
// The block reward is below the full reward of feature 26.
type AdjustedRewardBelowFullRewardPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardBelowFullRewardPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_26_below_full_reward.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardBelowFullRewardPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardCeaseXtnBuybackPreactivatedSuite uses preactivated features 14, 19, 20, 21, 26,
// 2 miners, dao, xtn, initR = 2000000000, increment = 100000000, desiredR = 2000000000,
// min_xtn_buy_back_period = 5.
type AdjustedRewardCeaseXtnBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardCeaseXtnBuybackPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_21_26_dao_xtn.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardCeaseXtnBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardWithBoostPreactivatedSuite uses preactivated features 14, 19, 20, 22, 23, 26,
// 2 miners, dao, xtn, initR = 2000000000, increment = 100000000, desiredR = 2000000000.
// Feature 26 supersedes feature 23, so the block reward must not be boosted.
type AdjustedRewardWithBoostPreactivatedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardWithBoostPreactivatedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_23_26_dao_xtn.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardWithBoostPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardSupportedSuite uses preactivated features 14, 19, 20 and supported feature 26,
// 2 miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000.
// The block reward must be reset to 20 WAVES at the activation height of feature 26.
type AdjustedRewardSupportedSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardSupportedSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_supported_26.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardSupportedSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

// AdjustedRewardSupportedIncreaseSuite uses preactivated features 14, 19, 20 and supported feature 26,
// 2 miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 2100000000.
// The block reward is reset to 20 WAVES at the activation height of feature 26 and then the votes of
// the miners increase it to 21 WAVES.
type AdjustedRewardSupportedIncreaseSuite struct {
	BaseSuite
}

func (suite *AdjustedRewardSupportedIncreaseSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		featureAndRewardSettingsFromFile(
			rewardSettingsFolder,
			adjustedRewardSettingsFolder,
			"adjusted_reward_preactivated_14_19_20_supported_26_increase.json",
		),
		config.WithQuorum(2),
	}
}

func (suite *AdjustedRewardSupportedIncreaseSuite) SetupSuite() {
	suite.BaseSetup(suite.BlockchainOpts()...)
}

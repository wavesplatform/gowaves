package fixtures

import "github.com/wavesplatform/gowaves/itests/config"

const (
	boostRewardSettingsFolder = "boost_reward_settings"
)

// ----- preactivated features 14, 19, 20, 21, 22, 23 -----
// 2 miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 800000000
// min_xtn_buy_back_period = block_reward_boost_period
type RewardBoostDaoXtnEqualPeriodsPreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostDaoXtnEqualPeriodsPreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21eqp23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21eqp23.json"),
	}
}

func (suite *RewardBoostDaoXtnEqualPeriodsPreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// ----- preactivated features 14, 19, 20, 21, 22, 23 -----
// 2 miners, dao, xtn, initR = 900000000, increment = 100000000, desiredR = 700000000
// min_xtn_buy_back_period < block_reward_boost_period
type RewardBoostDaoXtnP21LessP23PreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostDaoXtnP21LessP23PreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21lsp23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21lsp23.json"),
	}
}

func (suite *RewardBoostDaoXtnP21LessP23PreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// ----- preactivated features 14, 19, 20, 21, 22, 23 -----
// 2 miners, dao, xtn, initR = 900000000, increment = 100000000, desiredR = 700000000
// min_xtn_buy_back_period > block_reward_boost_period
type RewardBoostDaoXtnP21GreaterP23PreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostDaoXtnP21GreaterP23PreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21grp23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_p21grp23.json"),
	}
}

func (suite *RewardBoostDaoXtnP21GreaterP23PreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// 2 miners, dao, initR = 600000000, increment = 100000000, desiredR = 600000000
// 2 miners, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000
// 2 miners, initR = 600000000, increment = 100000000, desiredR = 600000000
// 1 miner, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000

// 1 miner, dao, initR = 600000000, increment = 100000000, desiredR = 600000000
type RewardBoostMinerDaoPreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostMinerDaoPreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_dao.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_dao.json"),
		config.WithNoScalaMining(),
	}
}

func (suite *RewardBoostMinerDaoPreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// 1 miner, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000
// min_xtn_buy_back_period < block_reward_boost_period
type RewardBoostMinerXtnP21LessP23PreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostMinerXtnP21LessP23PreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_xtn_p21lsp23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_xtn_p21lsp23.json"),
		config.WithNoScalaMining(),
	}
}

func (suite *RewardBoostMinerXtnP21LessP23PreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// 1 miner, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000
// min_xtn_buy_back_period > block_reward_boost_period
type RewardBoostMinerXtnP21GreaterP23PreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostMinerXtnP21GreaterP23PreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_xtn_p21grp23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_1miner_xtn_p21grp23.json"),
		config.WithNoScalaMining(),
	}
}

func (suite *RewardBoostMinerXtnP21GreaterP23PreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// ----- preactivated features 14, 19, 20, 21, 22, supported 23 -----
// 2 miners, dao, xtn, initR = 600000000, increment = 100000000, desiredR = 600000000
// xtn buyback period ends when start boost period
type RewardBoostMinerXtnDaoSupportedF23Suite struct {
	BaseSuite
}

func (suite *RewardBoostMinerXtnDaoSupportedF23Suite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_supported_23.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_supported_23.json"),
	}
}

func (suite *RewardBoostMinerXtnDaoSupportedF23Suite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

// ----- preactivated features 14, 19, 20, 21, 22, 23 -----
// miners, initR = 600000000, increment = 100000000, desiredR = 600000000
type RewardBoostMinersPreactivatedAllSuite struct {
	BaseSuite
}

func (suite *RewardBoostMinersPreactivatedAllSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_miners.json",
		),
		config.WithRewardSettingFromFile(
			rewardSettingsFolder,
			boostRewardSettingsFolder,
			"boost_reward_preactivated_14_19_20_21_22_23_miners.json"),
	}
}

func (suite *RewardBoostMinersPreactivatedAllSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

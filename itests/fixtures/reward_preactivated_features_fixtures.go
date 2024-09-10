package fixtures

import (
	"github.com/wavesplatform/gowaves/itests/config"
)

const (
	preactivatedFeaturesConfigFolder       = "preactivated_14_19_20"
	preactivatedFeaturesWith21ConfigFolder = "preactivated_14_19_20_21"
)

// preactivated features 14, 19, 20.

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815.
type RewardIncreaseDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "7W_2miners_dao_xtn_increase.json"),
	)
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815.
type RewardUnchangedDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "6W_2miners_dao_xtn_not_changed.json"),
	)
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816.
type RewardDecreaseDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "5W_2miners_dao_xtn_decrease.json"),
	)
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_increase.json")
// NODE - 817.
type RewardIncreaseDaoPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "7W_2miners_dao_increase.json"),
	)
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817.
type RewardUnchangedXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "6W_2miners_xtn_not_changed.json"),
	)
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818.
type Reward2WUnchangedDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "2W_2miners_dao_xtn_not_changed.json"),
	)
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_decrease.json")
// NODE -818.
type RewardDecreaseDaoPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "5W_2miners_dao_decrease.json"),
	)
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818.
type RewardDecreaseXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseXtnPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "5W_2miners_xtn_decrease.json"),
	)
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20/2miners_increase.json")
// NODE - 820.
type RewardIncreasePreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreasePreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "2miners_increase.json"),
	)
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f19.json")
// NODE - 821.
type RewardDaoXtnPreactivatedWithout19Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnPreactivatedWithout19Suite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "2miners_dao_xtn_without_f19.json"),
	)
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 859.
type RewardDaoXtnPreactivatedWithout20Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnPreactivatedWithout20Suite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesConfigFolder, "2miners_dao_xtn_without_f20.json"),
	)
}

//----- preactivated features 14, 19, 20, 21 FeaturesVotingPeriod = 1 -----

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_21/7W_2miners_dao_xtn_increase.json")
// NODE - 825.
type RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "7W_2miners_dao_xtn_increase.json"),
	)
}

// 2 miners, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_21/7W_2miners_xtn_increase.json")
// NODE - 825.
type RewardIncreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "7W_2miners_xtn_increase.json"),
	)
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_not_changed.json")
// NODE - 825.
type RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_not_changed.json"),
	)
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_21/5W_2miners_dao_xtn_decrease.json")
// NODE - 826.
type RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "5W_2miners_xtn_dao_decrease.json"),
	)
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_21/5W_2miners_xtn_decrease.json")
// NODE - 826.
type RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "5W_2miners_xtn_decrease.json"),
	)
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20_21/2W_2miners_dao_xtn_not_change.json")
// NODE - 826.
type Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "2W_2miners_dao_xtn_not_changed.json"),
	)
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20_21/5W_2miners_increase.json")
// NODE - 829.
type Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(preactivatedFeaturesWith21ConfigFolder, "5W_2miners_increase.json"),
	)
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830.
type RewardDaoXtnPreactivatedWith21Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnPreactivatedWith21Suite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(
			preactivatedFeaturesWith21ConfigFolder,
			"6W_2miners_dao_xtn_increase_without_20.json",
		),
	)
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830.
type RewardDaoXtnPreactivatedWithout19And20Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnPreactivatedWithout19And20Suite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithRewardSettingFromFile(
			preactivatedFeaturesWith21ConfigFolder,
			"6W_2miners_dao_xtn_increase_without_19_20.json",
		),
	)
}

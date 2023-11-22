package fixtures

import (
	"path/filepath"
)

const (
	supportedFeaturesConfigFolder       = "preactivated_14_supported_19_20"
	supportedFeature20ConfigFolder      = "preactivated_14_19_supported_20"
	supportedFeaturesWith21ConfigFolder = "preactivated_14_19_20_supported_21"
)

// preactivated features 14, features 19, 20 is supported.

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815.
type RewardIncreaseDaoXtnSupportedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "7W_2miners_dao_xtn_increase.json"))
}

// 2 miners, dao, xtn, initR=600000000, increment = 1, desiredR = 600000000
// ("preactivated_14_supported_19_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815.
type RewardUnchangedDaoXtnSupportedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "6W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816.
type RewardDecreaseDaoXtnSupportedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "5W_2miners_dao_xtn_decrease.json"))
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_supported_19_20/7W_2miners_dao_increase.json")
// NODE - 817.
type RewardIncreaseDaoSupportedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "7W_2miners_dao_increase.json"))
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_supported_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817.
type RewardUnchangedXtnSupportedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "6W_2miners_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_supported_19_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818.
type Reward2WUnchangedDaoXtnSupportedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "2W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_dao_decrease.json")
// NODE -818.
type RewardDecreaseDaoSupportedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "5W_2miners_dao_decrease.json"))
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_supported_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818.
type RewardDecreaseXtnSupportedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseXtnSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "5W_2miners_xtn_decrease.json"))
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_supported_19_20/2miners_increase.json")
// NODE - 820.
type RewardIncreaseSupportedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "2miners_increase.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f9.json")
// NODE - 821.
type RewardDaoXtnSupportedWithout19Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnSupportedWithout19Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "2miners_dao_xtn_without_f19.json"))
}

// preactivated features 14, 19, 20, supported 21

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_supported_21/7W_2miners_dao_xtn_increase.json")
// NODE - 825.
type RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "7W_2miners_dao_xtn_increase.json"))
}

// 2 miners, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_supported_21/7W_2miners_xtn_increase.json")
// NODE - 825.
type RewardIncreaseXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "7W_2miners_xtn_increase.json"))
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_not_changed.json")
// NODE - 825.
type RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_supported_21/5W_2miners_dao_xtn_decrease.json")
// NODE - 826.
type RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "5W_2miners_xtn_dao_decrease.json"))
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_supported_21/5W_2miners_xtn_decrease.json")
// NODE - 826.
type RewardDecreaseXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "5W_2miners_xtn_decrease.json"))
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20_supported_21/2W_2miners_dao_xtn_not_change.json")
// NODE - 826.
type Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "2W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20_supported_21/5W_2miners_increase.json")
// NODE - 829.
type Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite struct {
	BaseSuite
}

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackSupportedSuite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "5W_2miners_increase.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830.
type RewardDaoXtnSupportedWithout20Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnSupportedWithout20Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_increase_without_20.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase_without_20.json")
// NODE - 830.
type RewardDaoXtnSupportedWithout19And20Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnSupportedWithout19And20Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_increase_without_19_20.json"))
}

// preactivated features 14, 19, feature 20 is supported

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_supported_20/7W_2miners_dao_xtn_increase.json")
// NODE - 855, 856.
type RewardIncreaseDaoXtnSupported20Suite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnSupported20Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeature20ConfigFolder, "7W_2miners_dao_xtn_increase.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_supported_19_20/2miners_dao_xtn_without_f20.json")
// NODE - 858.
type RewardDaoXtnSupported19Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnSupported19Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "2miners_dao_xtn_without_f20.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20_supported_21/6W_2miners_dao_xtn_increase.json")
// NODE - 862.
type RewardDistributionRollbackBefore21Suite struct {
	BaseSuite
}

func (suite *RewardDistributionRollbackBefore21Suite) SetupSuite() {
	suite.BaseSetup(enableScalaMining,
		filepath.Join(supportedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_increase.json"))
}

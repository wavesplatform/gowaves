package fixtures

import (
	"path/filepath"

	"github.com/stoewer/go-strcase"
)

const (
	preactivatedFeaturesConfigFolder       = "preactivated_14_19_20"
	preactivatedFeaturesWith21ConfigFolder = "preactivated_14_19_20_21"
)

// preactivated features 14, 19, 20

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_xtn_increase.json")
// NODE - 815
type RewardIncreaseDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "7W_2miners_dao_xtn_increase.json"))
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_dao_xtn_not_changed.json")
// NODE - 815
type RewardUnchangedDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "6W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_xtn_decrease.json")
// NODE - 816
type RewardDecreaseDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "5W_2miners_dao_xtn_decrease.json"))
}

// 2 miners, dao, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20/7W_2miners_dao_increase.json")
// NODE - 817
type RewardIncreaseDaoPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "7W_2miners_dao_increase.json"))
}

// 2 miners, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20/6W_2miners_xtn_not_changed.json")
// NODE - 817
type RewardUnchangedXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "6W_2miners_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20/2W_2miners_dao_xtn_not_changed.json")
// NODE - 818
type Reward2WUnchangedDaoXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "2W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_dao_decrease.json")
// NODE -818
type RewardDecreaseDaoPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "5W_2miners_dao_decrease.json"))
}

// 2 miners, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20/5W_2miners_xtn_decrease.json")
// NODE - 818
type RewardDecreaseXtnPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseXtnPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "5W_2miners_xtn_decrease.json"))
}

// 2 miners, initR=500000000, increment = 100000000, desiredR = 700000000
// ("preactivated_14_19_20/2miners_increase.json")
// NODE - 820
type RewardIncreasePreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreasePreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "2miners_increase.json"))
}

// 2 miners,dao, xtn, initR=600000000, increment = 100000000, desiredR = 800000000
// ("preactivated_14_19_20/2miners_dao_xtn_without_f9.json")
// NODE - 821
type RewardDaoXtnPreactivatedWithout19Suite struct {
	BaseSuite
}

func (suite *RewardDaoXtnPreactivatedWithout19Suite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "2miners_dao_xtn_without_f19.json"))
}

//----- preactivated features 14, 19, 20, 21 FeaturesVotingPeriod = 1 -----

// 2 miners, dao, xtn, initR=700000000, increment = 100000000, desiredR = 900000000
// ("preactivated_14_19_20_21/7W_2miners_dao_xtn_increase.json")
// NODE - 825
type RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesWith21ConfigFolder, "7W_2miners_dao_xtn_increase.json"))
}

// 2 miners, dao, xtn, initR=600000000, increment = 100000000, desiredR = 600000000
// ("preactivated_14_19_20_21/6W_2miners_dao_xtn_not_changed.json")
// NODE - 825
type RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesWith21ConfigFolder, "6W_2miners_dao_xtn_not_changed.json"))
}

// 2 miners, dao, xtn, initR=500000000, increment = 100000000, desiredR = 300000000
// ("preactivated_14_19_20_21/5W_2miners_dao_xtn_decrease.json")
// NODE - 826
type RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesWith21ConfigFolder, "5W_2miners_xtn_dao_decrease.json"))
}

// 2 miners, dao, xtn, initR=200000000, increment = 100000000, desiredR = 200000000
// ("preactivated_14_19_20_21/2W_2miners_dao_xtn_not_change.json")
// NODE - 826
type Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	BaseSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesWith21ConfigFolder, "2W_2miners_dao_xtn_not_change.json"))
}

package fixtures

import (
	"path/filepath"

	"github.com/stoewer/go-strcase"
)

const (
	preactivatedFeaturesConfigFolder = "preactivated_14_19_20"
	supportedFeaturesConfigFolder    = "preactivated_14_19_supported_20"
)

// preactivated features 14, 19, 20
type RewardPreactivatedFeaturesSuite struct {
	BaseSuite
}

func (suite *RewardPreactivatedFeaturesSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(preactivatedFeaturesConfigFolder, "7W_2miners_dao_xtn.json"))
}

// preactivated features 14, 19, feature 20 is supported
type RewardSupportedFeaturesSuite struct {
	BaseSuite
}

func (suite *RewardSupportedFeaturesSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "rwrd_pf19_s20_7W_miners_dao_xtn.json"))
}

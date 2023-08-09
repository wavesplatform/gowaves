package fixtures

import (
	"path/filepath"

	"github.com/stoewer/go-strcase"
)

const (
	supportedFeaturesConfigFolder = "preactivated_14_19_supported_20"
)

// preactivated features 14, 19, feature 20 is supported
type RewardSupportedFeaturesSuite struct {
	BaseSuite
}

func (suite *RewardSupportedFeaturesSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining,
		filepath.Join(supportedFeaturesConfigFolder, "7W_miners_dao_xtn.json"))
}

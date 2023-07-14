package fixtures

import "github.com/stoewer/go-strcase"

type RewardSuite struct {
	BaseSuite
}

func (suite *RewardSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())

	suite.BaseSetup(suiteName, enableScalaMining, "reward_settings_increase.json")
}

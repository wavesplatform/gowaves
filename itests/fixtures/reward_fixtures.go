package fixtures

import "github.com/stoewer/go-strcase"

type RewardSuite struct {
	BaseSuite
}

func (suite *RewardSuite) SetupSuite() {
	const enableScalaMining = true
	suiteName := strcase.KebabCase(suite.T().Name())
	suite.BaseSetup(suiteName, enableScalaMining, "rwrd_pf19_20_7W_miners_dao_xtn.json")
}

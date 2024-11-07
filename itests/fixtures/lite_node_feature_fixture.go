package fixtures

import "github.com/wavesplatform/gowaves/itests/config"

const (
	baseSettingsConfigFolder = "base_feature_settings"
)

type BaseSettingSuite struct {
	BaseSuite
}

func (suite *BaseSettingSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithScalaMining(),
		config.WithFeatureSettingFromFile(baseSettingsConfigFolder, "lite_node_feature_fixture.json"),
		config.WithPaymentsSettingFromFile(baseSettingsConfigFolder, "lite_node_feature_fixture.json"),
	)
}

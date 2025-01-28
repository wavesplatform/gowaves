package fixtures

import "github.com/wavesplatform/gowaves/itests/config"

const (
	featureSettingsFolder    = "feature_settings"
	baseSettingsConfigFolder = "base_feature_settings"
)

type BaseSettingSuite struct {
	BaseSuite
}

func (suite *BaseSettingSuite) SetupSuite() {
	suite.BaseSetup(
		config.WithFeatureSettingFromFile(
			featureSettingsFolder,
			baseSettingsConfigFolder,
			"lite_node_feature_fixture.json",
		),
		config.WithPaymentsSettingFromFile(
			featureSettingsFolder,
			baseSettingsConfigFolder,
			"lite_node_feature_fixture.json",
		),
	)
}

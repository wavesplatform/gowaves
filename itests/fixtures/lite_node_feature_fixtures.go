package fixtures

import (
	"github.com/wavesplatform/gowaves/itests/clients"
	"github.com/wavesplatform/gowaves/itests/config"
)

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

type BaseSettingNegativeSuite struct {
	BaseSettingSuite
}

func (suite *BaseSettingNegativeSuite) SetupSuite() {
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
	suite.SendToNodes = append(suite.SendToNodes, clients.NodeScala)
}

type BaseSettingScriptExecutionFailedSuite struct {
	BaseSettingSuite
}

func (suite *BaseSettingScriptExecutionFailedSuite) SetupSuite() {
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
		config.WithNoGoMining(),
	)
	suite.SendToNodes = []clients.Implementation{clients.NodeScala}
}

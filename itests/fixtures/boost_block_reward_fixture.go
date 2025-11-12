package fixtures

import "github.com/wavesplatform/gowaves/itests/config"

type BoostBlockRewardSuite struct {
	BaseSuite
}

func (suite *BoostBlockRewardSuite) BlockchainOpts() []config.BlockchainOption {
	return []config.BlockchainOption{
		config.WithFeatureSettingFromFile(
			featureSettingsFolder,
			baseSettingsConfigFolder,
			"boost_block_reward_fixture.json",
		),
		config.WithRewardSettingFromFile(
			featureSettingsFolder,
			baseSettingsConfigFolder,
			"boost_block_reward_fixture.json"),
		config.WithNoScalaMining(),
		//config.WithNoGoMining(),
	}
}

func (suite *BoostBlockRewardSuite) SetupSuite() {
	suite.BaseSetupWithImages("go-node", "latest",
		"wavesplatform/wavesnode", "latest", suite.BlockchainOpts()...)
}

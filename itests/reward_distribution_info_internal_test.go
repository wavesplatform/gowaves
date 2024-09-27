//go:build !smoke

package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/utilities/reward"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// "NODE-855. /blockchain/rewards returns correct values for term,
// nextCheck and votingIntervalStart after CappedReward activation".
type RewardDistributionAPIRewardInfoPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionAPIRewardInfoPreactivatedSuite) Test_NODE855() {
	const node855 = "/blockchain/rewards returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	suite.Run(node855, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		td := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermAfter20Cfg,
			utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionAPIRewardInfoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoPreactivatedSuite))
}

type RewardDistributionAPIRewardInfoSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupported20Suite
}

func (suite *RewardDistributionAPIRewardInfoSupportedSuite) Test_NODE855_2() {
	const node855 = "/blockchain/rewards returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	suite.Run(node855, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		tdBefore20 := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermCfg,
			utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, tdBefore20)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		tdAfter20 := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermAfter20Cfg,
			utl.GetHeight(&suite.BaseSuite))
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, tdAfter20)
	})
}

func TestRewardDistributionAPIRewardInfoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoSupportedSuite))
}

// "NODE-856. /blockchain/rewards/{height} returns correct values for term,
// nextCheck and votingIntervalStart after CappedReward activation".
type RewardDistributionAPIRewardInfoAtHeightPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionAPIRewardInfoAtHeightPreactivatedSuite) Test_NODE856() {
	const node856 = "/blockchain/rewards/{height} returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	suite.Run(node856, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		h := utl.GetFeatureActivationHeight(&suite.BaseSuite, settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)) + 1
		td := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermAfter20Cfg, h)
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, td, h)
	})
}

func TestRewardDistributionAPIRewardInfoAtHeightPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoAtHeightPreactivatedSuite))
}

type RewardDistributionAPIRewardInfoAtHeightSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupported20Suite
}

func (suite *RewardDistributionAPIRewardInfoAtHeightSupportedSuite) Test_NODE856_2() {
	const node856 = "/blockchain/rewards/{height} returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	suite.Run(node856, func() {
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		h := utl.GetFeatureActivationHeight(&suite.BaseSuite, settings.BlockRewardDistribution,
			utl.GetHeight(&suite.BaseSuite)) + 1
		tdBefore20 := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermCfg, h)
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdBefore20, h)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		h = utl.GetFeatureActivationHeight(&suite.BaseSuite, settings.CappedRewards, utl.GetHeight(&suite.BaseSuite))
		tdAfter20 := testdata.ExpectedRewardInfoAPITestData(&suite.BaseSuite, utl.GetRewardTermAfter20Cfg, h)
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdAfter20, h)
	})
}

func TestRewardDistributionAPIRewardInfoAtHeightSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoAtHeightSupportedSuite))
}

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
		utl.WaitForHeight(&suite.BaseSuite, uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1)
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiAfterPreactivated20TestData)
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
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiBefore20TestData)
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		utl.WaitForHeight(&suite.BaseSuite, uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+utl.GetRewardTermAfter20Cfg(&suite.BaseSuite))
		reward.GetRewardInfoAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiAfterSupported20TestData)
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
		utl.WaitForHeight(&suite.BaseSuite, uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1)
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiAfterPreactivated20TestData,
			uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
				settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+
				utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1)
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
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiBefore20TestData,
			uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, settings.BlockRewardDistribution,
				utl.GetHeight(&suite.BaseSuite))+1))
		utl.GetActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		utl.WaitForHeight(&suite.BaseSuite, uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+utl.GetRewardTermAfter20Cfg(&suite.BaseSuite))
		reward.GetRewardInfoAtHeightAndChecks(&suite.BaseSuite, testdata.GetRewardInfoApiAfterSupported20TestData,
			uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
				settings.CappedRewards, utl.GetHeight(&suite.BaseSuite)))+utl.GetRewardTermAfter20Cfg(&suite.BaseSuite))
	})
}

func TestRewardDistributionAPIRewardInfoAtHeightSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoAtHeightSupportedSuite))
}

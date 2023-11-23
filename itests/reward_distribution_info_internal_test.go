package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func getRewardInfoAndChecks(suite *f.BaseSuite,
	td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues]) {
	rewardInfoGo, rewardInfoScala := utl.GetRewards(suite)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCkeckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart,
		rewardInfoScala.VotingIntervalStart)
}

func getRewardInfoAtHeightAndChecks(suite *f.BaseSuite,
	td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues], height uint64) {
	rewardInfoGo, rewardInfoScala := utl.GetRewardsAtHeight(suite, height)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCkeckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart,
		rewardInfoScala.VotingIntervalStart)
}

// "NODE-855. /blockchain/rewards returns correct values for term,
// nextCheck and votingIntervalStart after CappedReward activation".
type RewardDistributionAPIRewardInfoPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionAPIRewardInfoPreactivatedSuite) Test_NODE855() {
	const node855 = "/blockchain/rewards returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(node855, func() {
		getActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		getRewardInfoAndChecks(&suite.BaseSuite, td)
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
	tdBefore20 := testdata.GetRewardInfoApiBefore20TestData(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardInfoApiAfterSupported20TestData(&suite.BaseSuite)
	suite.Run(node855, func() {
		getActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		getRewardInfoAndChecks(&suite.BaseSuite, tdBefore20)
		getActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		getRewardInfoAndChecks(&suite.BaseSuite, tdAfter20)
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
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(node856, func() {
		getActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution,
			settings.CappedRewards)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, td, utl.GetHeight(&suite.BaseSuite))
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
	tdBefore20 := testdata.GetRewardInfoApiBefore20TestData(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardInfoApiAfterSupported20TestData(&suite.BaseSuite)
	suite.Run(node856, func() {
		getActivationOfFeatures(&suite.BaseSuite, settings.BlockReward, settings.BlockRewardDistribution)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdBefore20, utl.GetHeight(&suite.BaseSuite))
		getActivationOfFeatures(&suite.BaseSuite, settings.CappedRewards)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdAfter20, utl.GetHeight(&suite.BaseSuite))
	})
}

func TestRewardDistributionAPIRewardInfoAtHeightSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoAtHeightSupportedSuite))
}

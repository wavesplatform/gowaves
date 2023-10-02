package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
)

func getRewardInfoAndChecks(suite *f.BaseSuite, td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues]) {
	rewardInfoGo, rewardInfoScala := utl.GetRewards(suite)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCkeckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart, rewardInfoScala.VotingIntervalStart)
}

func getRewardInfoAtHeightAndChecks(suite *f.BaseSuite, td testdata.RewardDistributionApiTestData[testdata.RewardInfoApiExpectedValues], height uint64) {
	rewardInfoGo, rewardInfoScala := utl.GetRewardsAtHeight(suite, height)
	utl.TermCheck(suite.T(), td.Expected.Term, rewardInfoGo.Term, rewardInfoScala.Term)
	utl.NextCkeckParameterCheck(suite.T(), td.Expected.NextCheck, rewardInfoGo.NextCheck, rewardInfoScala.NextCheck)
	utl.VotingIntervalStartCheck(suite.T(), td.Expected.VotingIntervalStart, rewardInfoGo.VotingIntervalStart, rewardInfoScala.VotingIntervalStart)
}

// "NODE-855. /blockchain/rewards returns correct values for term, nextCheck and votingIntervalStart after CappedReward activation"
type RewardDistributionApiRewardInfoPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionApiRewardInfoPreactivatedSuite) Test_NODE855() {
	name := "NODE-855. /blockchain/rewards returns correct values for term, nextCheck and votingIntervalStart " +
		"after CappedReward activation"
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardInfoAndChecks(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionApiRewardInfoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRewardInfoPreactivatedSuite))
}

type RewardDistributionApiRewardInfoSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupported20Suite
}

func (suite *RewardDistributionApiRewardInfoSupportedSuite) Test_NODE855_2() {
	name := "NODE-855. /blockchain/rewards returns correct values for term, nextCheck and votingIntervalStart " +
		"after CappedReward activation"
	tdBefore20 := testdata.GetRewardInfoApiBefore20TestData(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardInfoApiAfterSupported20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardInfoAndChecks(&suite.BaseSuite, tdBefore20)
		getActivationOfFeatures(&suite.BaseSuite, 20)
		getRewardInfoAndChecks(&suite.BaseSuite, tdAfter20)
	})
}

func TestRewardDistributionApiRewardInfoSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRewardInfoSupportedSuite))
}

// "NODE-856. /blockchain/rewards/{height} returns correct values for term, nextCheck and votingIntervalStart after CappedReward activation"
type RewardDistributionApiRewardInfoAtHeightPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionApiRewardInfoAtHeightPreactivatedSuite) Test_NODE856() {
	name := "NODE-856. /blockchain/rewards/{height} returns correct values for term, nextCheck and votingIntervalStart " +
		"after CappedReward activation"
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, td, utl.GetHeight(&suite.BaseSuite))
	})
}

func TestRewardDistributionApiRewardInfoAtHeightPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRewardInfoAtHeightPreactivatedSuite))
}

type RewardDistributionApiRewardInfoAtHeightSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupported20Suite
}

func (suite *RewardDistributionApiRewardInfoAtHeightSupportedSuite) Test_NODE856_2() {
	name := "NODE-856. /blockchain/rewards/{height} returns correct values for term, nextCheck and votingIntervalStart " +
		"after CappedReward activation"
	tdBefore20 := testdata.GetRewardInfoApiBefore20TestData(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardInfoApiAfterSupported20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdBefore20, utl.GetHeight(&suite.BaseSuite))
		getActivationOfFeatures(&suite.BaseSuite, 20)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdAfter20, utl.GetHeight(&suite.BaseSuite))
	})
}

func TestRewardDistributionApiRewardInfoAtHeightSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRewardInfoAtHeightSupportedSuite))
}

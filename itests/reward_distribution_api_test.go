package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward_utilities"
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

// NODE - 858. Rollback on height before BlockRewardDistribution feature activation should be correct
type RewardDistributionApiRollbackBeforeF19Suite struct {
	f.RewardDaoXtnSupported19Suite
}

func (suite *RewardDistributionApiRollbackBeforeF19Suite) Test_NODE858() {
	name := "NODE-858. Rollback on height before BlockRewardDistribution feature activation should be correct"
	tdBefore19 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	tdAfter19 := testdata.GetRewardDaoXtnSupported19TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		getActivationOfFeatures(&suite.BaseSuite, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter19)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19-3), true)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore19)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH19))
		getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter19)
	})
}

func TestRewardDistributionApiRollbackBeforeF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRollbackBeforeF19Suite))
}

// NODE - 859. Rollback on height after BlockRewardDistribution feature activation should be correct
type RewardDistributionApiRollbackAfterF19Suite struct {
	f.RewardDaoXtnPreactivatedWithout20Suite
}

func (suite *RewardDistributionApiRollbackAfterF19Suite) Test_NODE859() {
	name := "NODE-859. Rollback on height after BlockRewardDistribution feature activation should be correct"
	tdAfter19 := testdata.GetRewardDaoXtnPreactivated19TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter19)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH19+2))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19+1), true)
		getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter19)
	})
}

func TestRewardDistributionApiRollbackAfterF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRollbackAfterF19Suite))
}

// NODE - 860. Rollback on height before CappedReward feature activation should be correct
type RewardDistributionApiRollbackBeforeF20Suite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionApiRollbackBeforeF20Suite) Test_NODE860() {
	name := " NODE - 860. Rollback on height before CappedReward feature activation should be correct"
	tdBefore20 := testdata.GetRewardDistributionAfter14Before19(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardIncreaseDaoXtnSupportedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdBefore20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdBefore20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdBefore20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdBefore20)
		//getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore20)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 20, utl.GetHeight(&suite.BaseSuite))
		//getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter20)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH20-4), true)
		//getRewardDistributionAndChecks(&suite.BaseSuite, tdBefore20)
		//utl.WaitForHeight(&suite.BaseSuite, uint64(activationH20))
		//getRewardDistributionAndChecks(&suite.BaseSuite, tdAfter20)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, tdAfter20)
		utl.WaitForNewHeight(&suite.BaseSuite)
	})
}

func TestRewardDistributionApiRollbackBeforeF20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionApiRollbackBeforeF20Suite))
}

type SimpleRollbackSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *SimpleRollbackSuite) Test_Simple_Rollback() {
	name := "Test simple rollback"
	td := testdata.GetRewardIncreaseDaoXtnPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(2), true)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		utl.WaitForNewHeight(&suite.BaseSuite)
		reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestSimpleRollbackSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SimpleRollbackSuite))
}

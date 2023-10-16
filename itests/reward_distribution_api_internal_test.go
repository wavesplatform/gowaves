package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
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
	name := "NODE-855. /blockchain/rewards returns correct values for term, nextCheck and votingIntervalStart " +
		"after CappedReward activation"
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
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
	name := "NODE-856. /blockchain/rewards/{height} returns correct values for term," +
		" nextCheck and votingIntervalStart after CappedReward activation"
	td := testdata.GetRewardInfoApiAfterPreactivated20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
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
	name := "NODE-856. /blockchain/rewards/{height} returns correct values for term, " +
		"nextCheck and votingIntervalStart after CappedReward activation"
	tdBefore20 := testdata.GetRewardInfoApiBefore20TestData(&suite.BaseSuite)
	tdAfter20 := testdata.GetRewardInfoApiAfterSupported20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdBefore20, utl.GetHeight(&suite.BaseSuite))
		getActivationOfFeatures(&suite.BaseSuite, 20)
		getRewardInfoAtHeightAndChecks(&suite.BaseSuite, tdAfter20, utl.GetHeight(&suite.BaseSuite))
	})
}

func TestRewardDistributionAPIRewardInfoAtHeightSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRewardInfoAtHeightSupportedSuite))
}

// NODE - 858. Rollback (/debug/rollback) on height before BlockRewardDistribution feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF19Suite struct {
	f.RewardDaoXtnSupported19Suite
}

func (suite *RewardDistributionAPIRollbackBeforeF19Suite) Test_NODE858() {
	name := "NODE-858. Rollback on height before BlockRewardDistribution feature activation should be correct"
	td := testdata.GetRollbackBeforeF19TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		getActivationOfFeatures(&suite.BaseSuite, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19-3), true)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		getActivationOfFeatures(&suite.BaseSuite, 19)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
	})
}

func TestRewardDistributionAPIRollbackBeforeF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF19Suite))
}

// NODE - 859. Rollback (/debug/rollback) on height after BlockRewardDistribution feature activation should be correct.
type RewardDistributionAPIRollbackAfterF19Suite struct {
	f.RewardDaoXtnPreactivatedWithout20Suite
}

func (suite *RewardDistributionAPIRollbackAfterF19Suite) Test_NODE859() {
	name := "NODE-859. Rollback on height after BlockRewardDistribution feature activation should be correct"
	td := testdata.GetRollbackAfterF19TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		activationH19 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH19+4))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH19+1), true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
	})
}

func TestRewardDistributionAPIRollbackAfterF19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackAfterF19Suite))
}

// NODE - 860. Rollback (/debug/rollback) on height before CappedReward feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF20Suite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionAPIRollbackBeforeF20Suite) Test_NODE860() {
	name := " NODE - 860. Rollback on height before CappedReward feature activation should be correct"
	td := testdata.GetRollbackBeforeF20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 20, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH20)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1, true)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		getActivationOfFeatures(&suite.BaseSuite, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
	})
}

func TestRewardDistributionAPIRollbackBeforeF20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF20Suite))
}

// NODE - 861. Rollback (/debug/rollback) on height after CappedReward feature activation should be correct.
type RewardDistributionAPIRollbackAfterF20Suite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionAPIRollbackAfterF20Suite) Test_NODE861() {
	name := "NODE - 861. Rollback on height after CappedReward feature activation should be correct"
	td := testdata.GetRollbackAfterF20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		activationH20 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 20, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
		utl.WaitForHeight(&suite.BaseSuite, uint64(activationH20+4))
		utl.GetRollbackToHeight(&suite.BaseSuite, uint64(activationH20+1), true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature)
	})
}

func TestRewardDistributionAPIRollbackAfterF20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackAfterF20Suite))
}

// NODE - 862. Rollback on height before CeaseXTNBuyback feature activation should be correct.
type RewardDistributionAPIRollbackBeforeF21Suite struct {
	f.RewardDistributionRollbackBefore21Suite
}

func (suite *RewardDistributionAPIRollbackBeforeF21Suite) Test_NODE862() {
	name := "NODE - 862. Rollback on height before CeaseXTNBuyback feature activation should be correct"
	td := testdata.GetRollbackBeforeF21TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite,
			19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriodCfg(&suite.BaseSuite)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		activationH21 := utl.GetFeatureActivationHeight(&suite.BaseSuite, 21, utl.GetHeight(&suite.BaseSuite))
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature.AfterXtnBuyBackPeriod)
		utl.GetRollbackToHeight(&suite.BaseSuite,
			uint64(activationH21)-utl.GetRewardTermAfter20Cfg(&suite.BaseSuite)-1, true)
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeFeature)
		getActivationOfFeatures(&suite.BaseSuite, 21)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterFeature.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDistributionAPIRollbackBeforeF21Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionAPIRollbackBeforeF21Suite))
}

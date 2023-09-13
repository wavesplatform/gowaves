package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward_utilities"
)

// Test steps
type GetTestData func(suite *f.BaseSuite) testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValues]

func getRewardDistribution(suite *f.BaseSuite, testdata GetTestData, featureIds ...int) {
	h := utl.GetHeight(suite)
	//ids of features that should be activated
	utl.FeatureShouldBeActivated(suite, 14, h)

	//additional features that should be activated
	for _, i := range featureIds {
		utl.FeatureShouldBeActivated(suite, i, h)
	}

	td := testdata(suite)
	//get reward for 1 block
	rewardDistributions, term := reward_utilities.GetBlockRewardDistribution(suite, td)
	//check results
	utl.TermCheck(suite.T(), td.Expected.Term, term.TermGo, term.TermScala)
	utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
		rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo, rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala)
	utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
		rewardDistributions.DAODiffBalance.BalanceInWavesGo, rewardDistributions.DAODiffBalance.BalanceInWavesScala)
	utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
		rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo, rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala)
}

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
type RewardDistributionIncreaseDaoXtnPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnPreactivatedSuite) Test_NODE815() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardIncreaseDaoXtnPreactivatedTestData, 19, 20)
	})

}

func TestRewardDistributionIncreaseDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnPreactivatedSuite))
}

type RewardDistributionUnchangedDaoXtnPreactivatedSuite struct {
	f.RewardUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionUnchangedDaoXtnPreactivatedSuite) Test_NODE815_2() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardUnchangedDaoXtnPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionUnchangedDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedDaoXtnPreactivatedSuite))
}

// NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
type RewardDistributionDecreaseDaoXtnPreactivatedSuite struct {
	f.RewardDecreaseDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionDecreaseDaoXtnPreactivatedSuite) Test_NODE816() {
	name := "NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseDaoXtnPreactivatedTestData, 19, 20)
	})

}

func TestRewardDistributionDecreaseDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoXtnPreactivatedSuite))
}

// NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES after
// CappedReward feature (20) activation
type RewardDistributionIncreaseDaoPreactivatedSuite struct {
	f.RewardIncreaseDaoPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoPreactivatedSuite) Test_NODE817() {
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardIncreaseDaoPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionIncreaseDaoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoPreactivatedSuite))
}

type RewardDistributionUnchangedXtnPreactivatedSuite struct {
	f.RewardUnchangedXtnPreactivatedSuite
}

func (suite *RewardDistributionUnchangedXtnPreactivatedSuite) Test_NODE817_2() {
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardUnchangedXtnPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionUnchangedXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedXtnPreactivatedSuite))
}

// NODE-818. Single XTN buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation
type RewardDistributionDecreaseXtnPreactivatedSuite struct {
	f.RewardDecreaseXtnPreactivatedSuite
}

func (suite *RewardDistributionDecreaseXtnPreactivatedSuite) Test_NODE818() {
	name := "NODE-818. Single XTN buyback address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseXtnPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionDecreaseXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnPreactivatedSuite))
}

// NODE-818_2. Single XTN Buyback or DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation
type RewardDistributionDecreaseDaoPreactivatedSuite struct {
	f.RewardDecreaseDaoPreactivatedSuite
}

func (suite *RewardDistributionDecreaseDaoPreactivatedSuite) Test_NODE818_2() {
	name := "NODE-818. Single DAO address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDecreaseDaoPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionDecreaseDaoPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseDaoPreactivatedSuite))
}

// NODE-818_3 If reward R <= 2 Waves mainer gets all reward
type RewardDistribution2WUnchangedDaoXtnPreactivatedSuite struct {
	f.Reward2WUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnPreactivatedSuite) Test_NODE818_3() {
	name := "NODE-818. mainer gets all reward If reward R <= 2 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetReward2WUnchangedDaoXtnPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistribution2WUnchangedDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistribution2WUnchangedDaoXtnPreactivatedSuite))
}

// NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined
// after CappedReward feature (20) activation
type RewardDistributionIncreasePreactivatedSuite struct {
	f.RewardIncreasePreactivatedSuite
}

func (suite *RewardDistributionIncreasePreactivatedSuite) Test_NODE820() {
	name := "NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardPreactivatedTestData, 19, 20)
	})
}

func TestRewardDistributionIncreasePreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreasePreactivatedSuite))
}

// NODE-821. Miner should get full block reward after CappedReward feature (20) activation
// if BlockRewardDistribution feature (19) is not activated
type RewardDistributionDaoXtnPreactivatedWithout19Suite struct {
	f.RewardDaoXtnPreactivatedWithout19Suite
}

func (suite *RewardDistributionDaoXtnPreactivatedWithout19Suite) Test_NODE821() {
	name := "NODE-821. Miner should get full block reward after CappedReward feature (20) activation " +
		"if BlockRewardDistribution feature (19) is not activated"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, testdata.GetRewardDaoXtnPreactivatedWithout19TestData, 20)
	})
}

func TestRewardDistributionDaoXtnPreactivatedWithout19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnPreactivatedWithout19Suite))
}

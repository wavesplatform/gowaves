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
func getRewardDistribution(suite *f.BaseSuite, td testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValuesPositive]) {
	h := utl.GetHeight(suite)
	//feature 14 should be activated
	utl.FeatureShouldBeActivated(suite, 14, h)
	//feature 19 should be activated
	utl.FeatureShouldBeActivated(suite, 19, h)
	//feature 20 should be activated
	utl.FeatureShouldBeActivated(suite, 20, utl.WaitForFeatureActivation(suite, h, 20))
	//get reward for 1 block
	rewardDistributions := reward_utilities.GetBlockRewardDistribution(suite, td)
	//check results
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
	td := testdata.GetRewardIncreaseDaoXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})

}

func TestRewardDistributionIncreaseDaoXtnPreactivated(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnPreactivatedSuite))
}

type RewardDistributionUnchangedDaoXtnPreactivatedSuite struct {
	f.RewardUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistributionUnchangedDaoXtnPreactivatedSuite) Test_NODE815_2() {
	td := testdata.GetRewardUnchangedDaoXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
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
	td := testdata.GetRewardDecreaseDaoXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
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
	td := testdata.GetRewardIncreaseDaoPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
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
	td := testdata.GetRewardUnchangedXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-817. Single XTN buyback or dao address should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionUnchangedXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionUnchangedXtnPreactivatedSuite))
}

// NODE-818. Single XTN buyback or dao address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES
// after CappedReward feature (20) activation
/*type RewardDistributionDecreaseXtnPreactivatedSuite struct {
	f.RewardDecreaseXtnPreactivatedSuite
}

func (suite *RewardDistributionDecreaseXtnPreactivatedSuite) Test_NODE818() {
	td := testdata.GetRewardDecreaseXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-818. Single XTN buyback or dao address should get max((R - 2)/2, 0) WAVES when full block reward < 6 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionDecreaseXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDecreaseXtnPreactivatedSuite))
}*/

// NODE-818_2 If reward R <= 2 Waves mainer gets all reward
type RewardDistribution2WUnchangedDaoXtnPreactivatedSuite struct {
	f.Reward2WUnchangedDaoXtnPreactivatedSuite
}

func (suite *RewardDistribution2WUnchangedDaoXtnPreactivatedSuite) Test_NODE818_2() {
	td := testdata.GetRewardDecreaseXtnPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-818. mainer gets all reward If reward R <= 2 WAVES"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistribution2WUnchangedDaoXtnPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistribution2WUnchangedDaoXtnPreactivatedSuite))
}

// NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined
// after CappedReward feature (20) activation
/*type RewardDistributionIncreasePreactivatedSuite struct {
	f.RewardIncreasePreactivatedSuite
}

func (suite *RewardDistributionIncreasePreactivatedSuite) Test_NODE820() {
	td := testdata.GetRewardPreactivatedTestData(&suite.BaseSuite)
	name := "NODE-820. Miner should get full block reward when daoAddress and xtnBuybackAddress are not defined"
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionIncreasePreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreasePreactivatedSuite))
}*/

// NODE-821. Miner should get full block reward after CappedReward feature (20) activation
// if BlockRewardDistribution feature (19) is not activated
type RewardDistributionDaoXtnPreactivatedWithout19Suite struct {
	f.RewardDaoXtnPreactivatedWithout19Suite
}

func (suite *RewardDistributionDaoXtnPreactivatedWithout19Suite) Test_NODE821() {
	td := testdata.GetRewardDaoXtnPreactivatedWithout19TestData(&suite.BaseSuite)
	name := "NODE-821. Miner should get full block reward after CappedReward feature (20) activation " +
		"if BlockRewardDistribution feature (19) is not activated"
	suite.Run(name, func() {
		h := utl.GetHeight(&suite.BaseSuite)
		//feature 14 should be activated
		utl.FeatureShouldBeActivated(&suite.BaseSuite, 14, h)
		//feature 20 should be activated
		utl.FeatureShouldBeActivated(&suite.BaseSuite, 20, utl.WaitForFeatureActivation(&suite.BaseSuite, h, 20))
		//get reward for 1 block
		rewardDistributions := reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td)
		//check results
		utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
			rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo, rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala)
		utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
			rewardDistributions.DAODiffBalance.BalanceInWavesGo, rewardDistributions.DAODiffBalance.BalanceInWavesScala)
		utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
			rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo, rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala)
	})
}

func TestRewardDistribution(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnPreactivatedWithout19Suite))
}

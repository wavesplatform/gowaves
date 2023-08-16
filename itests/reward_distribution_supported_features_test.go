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
func getRewardDistributionSupported(suite *f.BaseSuite, td testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValuesPositive]) {
	h := utl.GetHeight(suite)
	//features with ids should be activated
	utl.FeatureShouldBeActivated(suite, 14, h)
	utl.FeatureShouldBeActivated(suite, 19, h)

	utl.FeatureShouldBeActivated(suite, 20, h)
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
type RewardDistributionIncreaseDaoXtnSupportedSuite struct {
	f.RewardIncreaseDaoXtnSupportedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnSupportedSuite) Test_NODE815() {
	td := testdata.GetRewardIncreaseDaoXtnSupportedTestData(&suite.BaseSuite)
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		getRewardDistributionSupported(&suite.BaseSuite, td)
	})

}

func TestRewardDistributionIncreaseDaoXtnSupportedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnSupportedSuite))
}

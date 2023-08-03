package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/testdata"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward_utilities"
)

type RewardDistributionSuite struct {
	f.RewardPreactivatedFeaturesSuite
}

// test steps with predefined features 14, 19, 20
func getRewardDistribution(suite *f.BaseSuite, td testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValuesPositive]) {
	h := utl.GetHeight(suite)
	//feature 14 should be activated
	utl.FeatureShouldBeActivated(suite, 14, h)
	//feature 19 should be activated
	utl.FeatureShouldBeActivated(suite, 19, h)
	//feature 20 should be activated
	utl.WaitForHeight(suite, 15)
	fmt.Println(suite.Clients.StateHashCmp(suite.T(), h))
	utl.FeatureShouldBeActivated(suite, 20, utl.WaitForFeatureActivation(suite, h, 20))
	fmt.Println(suite.Clients.StateHashCmp(suite.T(), h))
	fmt.Println(utl.GetAvailableBalanceInWaves(suite, td.DaoAccount.Address))
	fmt.Println(utl.GetAvailableBalanceInWaves(suite, td.XtnBuyBackAccount.Address))
	//get reward distribution for 1 block
	rewardDistributions := reward_utilities.GetBlockRewardDistribution(suite, td)

	utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
		rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo, rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala)
	utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
		rewardDistributions.DAODiffBalance.BalanceInWavesGo, rewardDistributions.DAODiffBalance.BalanceInWavesScala)
	utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
		rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo, rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala)
}

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
func (suite *RewardDistributionSuite) Test_RewardDistributionPositive() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	td := testdata.GetRewardIncreaseDaoXtnTestDataPositive(&suite.BaseSuite)
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionSuite))
}

type RewardDistributionSuite2 struct {
	f.RewardSupportedFeaturesSuite
}

// NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
func (suite *RewardDistributionSuite2) Test_RewardDistributionPositive2() {
	name := "NODE-816. XTN buyback and dao addresses should get (R-2)/2 WAVES when full block reward < 6 WAVES"
	td := testdata.GetRewardIncreaseDaoXtnTestDataPositive(&suite.BaseSuite)
	suite.Run(name, func() {
		getRewardDistribution(&suite.BaseSuite, td)
	})
}

func TestRewardDistributionSuite2(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionSuite2))
}

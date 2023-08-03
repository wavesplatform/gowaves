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
	h = utl.WaitForHeight(suite, 19)
	h = utl.WaitForNewHeight(suite)
	fmt.Println(h)
	//fmt.Println(suite.Clients.StateHashCmp(suite.T(), h))
	utl.FeatureShouldBeActivated(suite, 20, utl.WaitForFeatureActivation(suite, h, 20))
	h = utl.GetHeight(suite)
	h = utl.WaitForNewHeight(suite)
	fmt.Println(h)
	//fmt.Println(suite.Clients.StateHashCmp(suite.T(), h))
	fmt.Println("Height when getting balances", utl.GetHeight(suite))
	miner1Go, miner1Scala := utl.GetAvailableBalanceInWaves(suite, td.Miner1Account.Address)
	miner2Go, miner2Scala := utl.GetAvailableBalanceInWaves(suite, td.Miner2Account.Address)
	daoGo, daoScala := utl.GetAvailableBalanceInWaves(suite, td.DaoAccount.Address)
	xtnGo, xtnScala := utl.GetAvailableBalanceInWaves(suite, td.XtnBuyBackAccount.Address)
	fmt.Println("Balance miner1 Go: ", miner1Go, "Balance miner1 Scala: ", miner1Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance miner2 Go: ", miner2Go, "Balance miner2 Scala: ", miner2Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance dao Go: ", daoGo, "Balance dao Scala: ", daoScala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance xtn Go: ", xtnGo, "Balance xtn Scala: ", xtnScala, "current height: ", utl.GetHeight(suite))
	//get reward distribution for 1 block
	rewardDistributions := reward_utilities.GetBlockRewardDistribution(suite, td)
	fmt.Println("Height when getting balances after waiting reward for 1 block", utl.GetHeight(suite))
	miner1Go, miner1Scala = utl.GetAvailableBalanceInWaves(suite, td.Miner1Account.Address)
	miner2Go, miner2Scala = utl.GetAvailableBalanceInWaves(suite, td.Miner2Account.Address)
	daoGo, daoScala = utl.GetAvailableBalanceInWaves(suite, td.DaoAccount.Address)
	xtnGo, xtnScala = utl.GetAvailableBalanceInWaves(suite, td.XtnBuyBackAccount.Address)
	fmt.Println("Balance miner1 Go: ", miner1Go, "Balance miner1 Scala: ", miner1Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance miner2 Go: ", miner2Go, "Balance miner2 Scala: ", miner2Scala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance dao Go: ", daoGo, "Balance dao Scala: ", daoScala, "current height: ", utl.GetHeight(suite))
	fmt.Println("Balance xtn Go: ", xtnGo, "Balance xtn Scala: ", xtnScala, "current height: ", utl.GetHeight(suite))

	fmt.Println("Expected sum diff miners: ", td.Expected.MinersSumDiffBalance,
		"Current sum diff miners Go: ", rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo,
		"Current sum diff miners Scala: ", rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala, "current height: ", utl.GetHeight(suite))

	fmt.Println("Expected diff dao: ", td.Expected.DaoDiffBalance,
		"Current diff dao Go: ", rewardDistributions.DAODiffBalance.BalanceInWavesGo,
		"Current diff dao Scala: ", rewardDistributions.DAODiffBalance.BalanceInWavesScala, "current height: ", utl.GetHeight(suite))

	fmt.Println("Expected diff xtn: ", td.Expected.XtnDiffBalance,
		"Current diff xtn Go: ", rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo,
		"Current diff xtn Scala: ", rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala, "current height: ", utl.GetHeight(suite))

	/*utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
		rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo, rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala)
	utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
		rewardDistributions.DAODiffBalance.BalanceInWavesGo, rewardDistributions.DAODiffBalance.BalanceInWavesScala)
	utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
		rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo, rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala)*/
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

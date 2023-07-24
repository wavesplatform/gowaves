package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/testdata"

	f "github.com/wavesplatform/gowaves/itests/fixtures"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reward_utilities"
)

type RewardDistributionSuite struct {
	f.RewardSuite
}

//tests with "BaseSuite" cfg

// NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES
// after Capped XTN buy-back & DAO amounts Feature activated (feature 20)
func (suite *RewardDistributionSuite) Test_RewardDistributionPositive() {
	name := "NODE-815. XTN buyback and dao addresses should get 2 WAVES when full block reward >= 6 WAVES"
	suite.Run(name, func() {
		td := testdata.GetRewardIncreaseDaoXtnTestDataPositive(&suite.BaseSuite)
		h := utl.GetHeight(&suite.BaseSuite)
		//feature 14 should be activated
		utl.FeatureShouldBeActivated(&suite.BaseSuite, 14, h)
		//feature 19 should be activated
		utl.FeatureShouldBeActivated(&suite.BaseSuite, 19, h)
		//feature 20 should be activated
		utl.FeatureShouldBeActivated(&suite.BaseSuite, 20, h)

		//get reward distribution for 1 block
		rewardDistributions := reward_utilities.GetBlockRewardDistribution(&suite.BaseSuite, td, h)

		utl.MinersSumDiffBalanceInWavesCheck(suite.T(), td.Expected.MinersSumDiffBalance,
			uint64(rewardDistributions.MinersSumDiffBalance.BalanceInWavesGo), uint64(rewardDistributions.MinersSumDiffBalance.BalanceInWavesScala))
		utl.DaoDiffBalanceInWavesCheck(suite.T(), td.Expected.DaoDiffBalance,
			uint64(rewardDistributions.DAODiffBalance.BalanceInWavesGo), uint64(rewardDistributions.DAODiffBalance.BalanceInWavesScala))
		utl.XtnBuyBackDiffBalanceInWavesCheck(suite.T(), td.Expected.XtnDiffBalance,
			uint64(rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesGo), uint64(rewardDistributions.XTNBuyBackDiffBalance.BalanceInWavesScala))
	})
}

func TestRewardDistributionSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionSuite))
}

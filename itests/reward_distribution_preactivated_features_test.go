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
func getActivationOfFeatures(suite *f.BaseSuite, featureIds ...int) {
	h := utl.GetHeight(suite)
	//features that should be activated
	for _, i := range featureIds {
		utl.FeatureShouldBeActivated(suite, i, h)
	}
}

func getRewardDistributionAndChecks(suite *f.BaseSuite, td testdata.RewardDistributionTestData[testdata.RewardDistributionExpectedValues]) {
	//td := testdata(suite)
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardIncreaseDaoXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardUnchangedDaoXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardDecreaseDaoXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardIncreaseDaoPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardUnchangedXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardDecreaseXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardDecreaseDaoPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetReward2WUnchangedDaoXtnPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardPreactivatedTestData(&suite.BaseSuite))
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
		getActivationOfFeatures(&suite.BaseSuite, 14, 20)
		getRewardDistributionAndChecks(&suite.BaseSuite, testdata.GetRewardDaoXtnPreactivatedWithout19TestData(&suite.BaseSuite))
	})
}

func TestRewardDistributionDaoXtnPreactivatedWithout19Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionDaoXtnPreactivatedWithout19Suite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward >= 6 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825() {
	name := "NODE-825, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward > 6 WAVES)"
	td := testdata.GetRewardIncreaseDaoXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})

}

func TestRewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward > 6 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardIncreaseXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825_2() {
	name := "NODE-825, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward > 6 WAVES)"
	td := testdata.GetRewardIncreaseXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})

}

func TestRewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDistributionIncreaseXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE-825. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward = 6 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE825_3() {
	name := "NODE-825, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward = 6 WAVES)"
	td := testdata.GetRewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward < 6 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826() {
	name := "NODE - 826, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward < 6 WAVES)"
	td := testdata.GetRewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward < 6 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826_2() {
	name := "NODE - 826, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward < 6 WAVES)"
	td := testdata.GetRewardDecreaseXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDecreaseXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 826. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward = 2 WAVES)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite struct {
	f.Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite
}

func (suite *Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite) Test_NODE826_3() {
	name := "NODE - 826, NODE-828. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after f19 activation height" +
		" + xtnBuybackRewardPeriod (full reward = 2 WAVES)"
	td := testdata.GetReward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestReward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward2WUnchangedDaoXtnCeaseXTNBuybackPreactivatedSuite))
}

// "NODE - 829. XTN buyback reward should be cancelled when CeaseXtnBuyback activated after xtnBuybackRewardPeriod blocks
// starting from BlockRewardDistribution activation height (full reward changes from 5 W to 7 W)"
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite struct {
	f.Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite
}

func (suite *Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite) Test_NODE829() {
	name := "NODE - 829, NODE-828. Miner should get full block reward if daoAddress and xtnBuybackAddress are not defined " +
		"after f19 activation height + xtnBuybackRewardPeriod"
	td := testdata.GetReward5W2MinersIncreaseCeaseXTNBuybackPreactivatedTestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 20, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestReward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Reward5W2MinersIncreaseCeaseXTNBuybackPreactivatedSuite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDaoXtnPreactivatedWithout20Suite struct {
	f.RewardDaoXtnPreactivatedWithout20Suite
}

func (suite *RewardDaoXtnPreactivatedWithout20Suite) Test_NODE830() {
	name := "NODE - 830, NODE-828. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated"
	td := testdata.GetRewardDaoXtnPreactivatedWithout20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 19, 21)
		ceaseXtnBuybackHeight := uint64(utl.GetFeatureActivationHeight(&suite.BaseSuite, 19, utl.GetHeight(&suite.BaseSuite))) + utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDaoXtnPreactivatedWithout20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnPreactivatedWithout20Suite))
}

// NODE-830. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated
// "NODE-828. Reward Distribution changed after h19+xtnBuybackRewardPeriod (h21 < h19+xtnBuybackRewardPeriod)"
type RewardDaoXtnPreactivatedWithout19And20Suite struct {
	f.RewardDaoXtnPreactivatedWithout19And20Suite
}

func (suite *RewardDaoXtnPreactivatedWithout19And20Suite) Test_NODE830_2() {
	name := "NODE - 830, NODE-828. Block reward distribution should not change after CeaseXtnBuyback activation if CappedReward not activated"
	td := testdata.GetRewardDaoXtnPreactivatedWithout19And20TestData(&suite.BaseSuite)
	suite.Run(name, func() {
		getActivationOfFeatures(&suite.BaseSuite, 14, 21)
		ceaseXtnBuybackHeight := utl.GetXtnBuybackPeriod(&suite.BaseSuite)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.BeforeXtnBuyBackPeriod)
		utl.WaitForHeight(&suite.BaseSuite, ceaseXtnBuybackHeight)
		getRewardDistributionAndChecks(&suite.BaseSuite, td.AfterXtnBuyBackPeriod)
	})
}

func TestRewardDaoXtnPreactivatedWithout19And20Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RewardDaoXtnPreactivatedWithout19And20Suite))
}
